package zt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// memberReadonly 是成员对象里由服务端维护的只读字段，回写前剥离。
var memberReadonly = []string{
	"id", "address", "nodeId", "networkId", "creationTime", "lastOnline",
	"lastActive", "online", "physicalAddress", "publicAddress",
	"clientVersion", "identity", "allowedIps", "updatedAt", "errors", "latency",
}

// networkReadonly 是网络 config 里由服务端维护的只读字段，回写前剥离。
// id/nwid 由本层注入，revision 与 stats 一律不回传控制器。
var networkReadonly = []string{
	"id", "nwid", "creationTime", "lastModified", "revision", "stats",
}

// Service 在 Client 之上做原 PHP ZerotierService 的等价组装。
// NodeAddress 为空时按 /status.address 现取（与原实现的每请求一次一致）。
type Service struct {
	Client      *Client
	NodeAddress string
}

// NewService 构造组装层。
func NewService(client *Client, nodeAddress string) *Service {
	return &Service{Client: client, NodeAddress: nodeAddress}
}

// Status 返回控制器 /status。
func (s *Service) Status(ctx context.Context) Result {
	return s.Client.Get(ctx, "/status")
}

// Networks 把 nwid 数组逐个补全成列表所需的关键字段。
func (s *Service) Networks(ctx context.Context) Result {
	res := s.Client.Get(ctx, "/controller/network")
	if !res.Success {
		return res
	}

	var nwids []json.RawMessage
	if err := json.Unmarshal(res.Body, &nwids); err != nil {
		return failResult("控制器返回的网络列表格式异常：" + err.Error())
	}

	list := make([]map[string]json.RawMessage, 0, len(nwids))
	for _, raw := range nwids {
		var nwid string
		if err := json.Unmarshal(raw, &nwid); err != nil {
			nwid = strings.TrimPrefix(string(raw), "\"")
			nwid = strings.TrimSuffix(nwid, "\"")
		}

		detail := s.GetNetwork(ctx, nwid)
		if !detail.Success {
			// 详情取不到也至少给出 id，便于前端定位坏数据
			list = append(list, map[string]json.RawMessage{
				"id":      json.RawMessage(mustJSON(nwid)),
				"nwid":    json.RawMessage(mustJSON(nwid)),
				"name":    json.RawMessage(mustJSON("")),
				"private": json.RawMessage("null"),
				"error":   json.RawMessage(mustJSON(detail.Message)),
			})
			continue
		}

		cfg, err := decodeObject(detail.Body)
		if err != nil {
			return failResult(fmt.Sprintf("网络 %s 配置格式异常：%v", nwid, err))
		}
		list = append(list, projectNetwork(nwid, cfg))
	}

	return okResult(mustJSON(list))
}

// NetworksCount 只数 nwid 个数，不逐个取详情。
func (s *Service) NetworksCount(ctx context.Context) Result {
	res := s.Client.Get(ctx, "/controller/network")
	if !res.Success {
		return res
	}
	var nwids []json.RawMessage
	if err := json.Unmarshal(res.Body, &nwids); err != nil {
		return failResult("控制器返回的网络列表格式异常：" + err.Error())
	}
	return okResult(mustJSON(len(nwids)))
}

// GetNetwork 返回完整网络 config（原样透传）。
func (s *Service) GetNetwork(ctx context.Context, nwid string) Result {
	return s.Client.Get(ctx, "/controller/network/"+nwid)
}

// CreateNetwork 生成合法 nwid 并用默认模板补齐缺省字段后 POST。
func (s *Service) CreateNetwork(ctx context.Context, payload json.RawMessage) Result {
	address := s.nodeAddressOverride()
	if address == "" {
		addr, err := s.lookupNodeAddress(ctx)
		if err != nil {
			return failResult("无法获取控制器节点地址，不能生成合法 nwid（可用 --node-address 指定）：" + err.Error())
		}
		address = addr
	}
	if len(address) != 10 {
		return failResult(fmt.Sprintf("控制器节点地址必须是 10 位 hex，当前 %q", address))
	}

	suffix, err := randomHex(3)
	if err != nil {
		return failResult(err.Error())
	}
	nwid := address + suffix

	merged, err := mergeObjects(DefaultNetwork(), payload)
	if err != nil {
		return failResult(err.Error())
	}
	stripKeys(merged, networkReadonly)
	merged["id"] = mustJSON(nwid)
	merged["nwid"] = mustJSON(nwid)
	if _, ok := merged["active"]; !ok {
		merged["active"] = mustJSON(true)
	}

	return s.Client.Post(ctx, "/controller/network/"+nwid, mustJSON(merged))
}

// UpdateNetwork 透传前端给的完整 config，仅剥离只读字段并注入 id/nwid。
func (s *Service) UpdateNetwork(ctx context.Context, nwid string, payload json.RawMessage) Result {
	obj, err := decodeObject(payload)
	if err != nil {
		return failResult(err.Error())
	}
	stripKeys(obj, networkReadonly)
	obj["id"] = mustJSON(nwid)
	obj["nwid"] = mustJSON(nwid)

	return s.Client.Post(ctx, "/controller/network/"+nwid, mustJSON(obj))
}

// DeleteNetwork 删除网络。
func (s *Service) DeleteNetwork(ctx context.Context, nwid string) Result {
	return s.Client.Delete(ctx, "/controller/network/"+nwid)
}

// Members 把本地 API 的「地址为键」对象规整成数组，并保证每项带 nodeId。
// 顶层值可能是完整对象（部分版本），也可能是紧凑形态的 revision 整数
// （zerotier-one 1.16.x 等），后者逐个回源 GET /member/<nodeId> 补全。
// 输出按 nodeId 字典序排列（Go map 无序，排序是为结果可复现）。
func (s *Service) Members(ctx context.Context, nwid string) Result {
	res := s.Client.Get(ctx, "/controller/network/"+nwid+"/member")
	if !res.Success {
		return res
	}

	if !isObject(res.Body) {
		// 旧版可能只返回地址数组：每项都不是成员对象，按原实现等价产出空列表
		if trimmed := bytes.TrimSpace(res.Body); len(trimmed) > 0 && trimmed[0] == '[' {
			return okResult(json.RawMessage("[]"))
		}
		return failResult("成员列表格式异常：期望是 JSON 对象")
	}

	raw, err := decodeObject(res.Body)
	if err != nil {
		return failResult("成员列表格式异常：" + err.Error())
	}

	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	list := make([]json.RawMessage, 0, len(keys))
	for _, key := range keys {
		member := raw[key]

		// 顶层已给完整对象：保证带 nodeId 后直接使用（部分版本走这条）。
		if isObject(member) {
			if obj, err := decodeObject(member); err == nil {
				if _, ok := obj["nodeId"]; ok {
					list = append(list, member)
					continue
				}
			}
			list = append(list, withNodeID(member, key))
			continue
		}

		// 紧凑形态：zerotier-one 1.16.x 等构建的顶层列表把成员值写成 revision 整数
		// （如 {"4a7090b201":6}），必须逐个回源 GET /member/<nodeId> 取回完整对象，
		// 否则整表被当成空、界面「成员为空」。回源失败也至少产出一行占位，避免误判为空。
		if full := s.GetMember(ctx, nwid, key); full.Success && isObject(full.Body) {
			body := full.Body
			if obj, err := decodeObject(body); err == nil {
				if _, ok := obj["nodeId"]; !ok {
					body = withNodeID(body, key)
				}
			}
			list = append(list, body)
			continue
		}
		list = append(list, json.RawMessage(mustJSON(map[string]string{"nodeId": key})))
	}

	return okResult(mustJSON(list))
}

// GetMember 返回单个成员（原样透传）。
func (s *Service) GetMember(ctx context.Context, nwid, nodeID string) Result {
	return s.Client.Get(ctx, "/controller/network/"+nwid+"/member/"+nodeID)
}

// UpdateMember 透传任意可写字段，剥离只读字段。
func (s *Service) UpdateMember(ctx context.Context, nwid, nodeID string, payload json.RawMessage) Result {
	obj, err := decodeObject(payload)
	if err != nil {
		return failResult(err.Error())
	}
	stripKeys(obj, memberReadonly)

	return s.Client.Post(ctx, "/controller/network/"+nwid+"/member/"+nodeID, mustJSON(obj))
}

// DeleteMember 移除成员。
func (s *Service) DeleteMember(ctx context.Context, nwid, nodeID string) Result {
	return s.Client.Delete(ctx, "/controller/network/"+nwid+"/member/"+nodeID)
}

// RawGet 只读透传未封装的控制器路径。
func (s *Service) RawGet(ctx context.Context, path string) Result {
	safe, err := SafeRawPath(path)
	if err != nil {
		return failResult(err.Error())
	}
	return s.Client.Get(ctx, safe)
}

// RawPost 写入透传（如 /admin 命令）。
func (s *Service) RawPost(ctx context.Context, path string, payload json.RawMessage) Result {
	safe, err := SafeRawPath(path)
	if err != nil {
		return failResult(err.Error())
	}
	return s.Client.Post(ctx, safe, payload)
}

// nodeAddressOverride 返回配置里显式给出的控制器地址。
func (s *Service) nodeAddressOverride() string {
	return strings.ToLower(strings.TrimSpace(s.NodeAddress))
}

// lookupNodeAddress 从 /status.address 取控制器节点地址。
func (s *Service) lookupNodeAddress(ctx context.Context) (string, error) {
	res := s.Status(ctx)
	if !res.Success {
		return "", fmt.Errorf("GET /status 失败：%s", res.Message)
	}
	obj, err := decodeObject(res.Body)
	if err != nil {
		return "", err
	}
	var address string
	if err := json.Unmarshal(obj["address"], &address); err != nil {
		return "", fmt.Errorf("控制器未返回 address 字段")
	}
	return strings.ToLower(strings.TrimSpace(address)), nil
}

// projectNetwork 从完整 config 提取列表所需字段，缺省值语义与 PHP 版一致。
func projectNetwork(nwid string, cfg map[string]json.RawMessage) map[string]json.RawMessage {
	pick := func(key string, fallback json.RawMessage) json.RawMessage {
		if v, ok := cfg[key]; ok && !isNull(v) {
			return v
		}
		return fallback
	}

	return map[string]json.RawMessage{
		"id":           pick("id", mustJSON(nwid)),
		"nwid":         pick("nwid", mustJSON(nwid)),
		"name":         pick("name", mustJSON("")),
		"private":      pick("private", mustJSON(true)),
		"active":       pick("active", mustJSON(true)),
		"creationTime": pick("creationTime", json.RawMessage("null")),
	}
}

// DefaultNetwork 是新建网络的缺省模板，与 config/zerotier.php 的 default_network 对齐。
func DefaultNetwork() json.RawMessage {
	return mustJSON(map[string]any{
		"active":            true,
		"private":           true,
		"enableBroadcast":   true,
		"mtu":               2800,
		"multicastTTL":      128,
		"multicastLimit":    32,
		"v4AssignMode":      map[string]any{"zt": true},
		"v6AssignMode":      map[string]any{"6plane": false, "rfc4193": false, "zt": false},
		"routes":            []any{},
		"ipAssignmentPools": []any{},
	})
}

// SafeRawPath 校验透传路径：只接受本站相对路径，禁协议相对与穿越，避免把令牌发往第三方主机。
func SafeRawPath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	switch {
	case path == "":
		return "", fmt.Errorf("path 不能为空")
	case !strings.HasPrefix(path, "/"):
		return "", fmt.Errorf("path 必须以 / 开头，当前 %q", raw)
	case strings.HasPrefix(path, "//"):
		return "", fmt.Errorf("path 不能是协议相对地址：%q", raw)
	case strings.Contains(path, "://"):
		return "", fmt.Errorf("path 不能包含 scheme：%q", raw)
	}

	cleaned := strings.TrimSpace(path)
	if i := strings.IndexAny(cleaned, "?#"); i >= 0 {
		cleaned = cleaned[:i]
	}
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path 不允许包含 ..")
	}
	return cleaned, nil
}
