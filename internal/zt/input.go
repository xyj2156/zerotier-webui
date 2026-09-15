package zt

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

// convenienceKeys 是仅用于前端便捷输入的字段，归一化后不透传给控制器。
var convenienceKeys = []string{"type", "cidr", "_token"}

// NormalizeNetworkInput 归一化网络写入负载：整体透传（尽量保留全部可配置字段），只处理
//   - type: "private"|"public" → private 布尔
//   - cidr: 便捷字段，未显式给出 pools/routes 时展开成 ipAssignmentPools + routes
//
// 与原 PHP ZerotierController::normalizeNetworkInput 等价；校验失败返回 error，
// 由上层翻译成统一信封（status=1 + message），不再走 Laravel 的 422 结构。
func NormalizeNetworkInput(raw []byte) (json.RawMessage, error) {
	obj := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(raw)) > 0 {
		decoded, err := decodeObject(json.RawMessage(raw))
		if err != nil {
			return nil, err
		}
		obj = decoded
	}

	if v, ok := obj["private"]; ok && !isNull(v) && string(bytes.TrimSpace(v)) != "true" && string(bytes.TrimSpace(v)) != "false" {
		return nil, fmt.Errorf("private 必须是布尔值")
	}

	if v, ok := obj["type"]; ok {
		var kind string
		if err := json.Unmarshal(v, &kind); err != nil {
			return nil, fmt.Errorf("type 必须是字符串 private 或 public")
		}
		kind = strings.ToLower(strings.TrimSpace(kind))
		if kind != "private" && kind != "public" {
			return nil, fmt.Errorf("type 只能是 private 或 public，当前 %q", kind)
		}
		obj["private"] = mustJSON(kind == "private")
	}

	cidr := ""
	if v, ok := obj["cidr"]; ok {
		_ = json.Unmarshal(v, &cidr)
	}
	stripKeys(obj, convenienceKeys)

	if strings.TrimSpace(cidr) != "" && isEmptyJSON(obj["ipAssignmentPools"]) && isEmptyJSON(obj["routes"]) {
		pools, routes, ok := expandCIDR(cidr)
		if ok {
			obj["ipAssignmentPools"] = pools
			obj["routes"] = routes
		}
	}

	return mustJSON(obj), nil
}

// expandCIDR 把 a.b.c.d/n 展开成 zerotier 的分配池与路由；非法或不支持时 ok=false（与原实现一致，静默跳过）。
func expandCIDR(cidr string) (pools, routes json.RawMessage, ok bool) {
	ip, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil || ip.To4() == nil {
		return nil, nil, false
	}
	bits, _ := network.Mask.Size()
	if bits < 1 || bits > 31 {
		return nil, nil, false
	}

	networkU32 := ipToU32(network.IP)
	hostMask := ^netMaskU32(bits)
	broadcast := networkU32 | hostMask

	pools = mustJSON([]map[string]any{{
		"ipRangeStart": u32ToIP(networkU32 + 1),
		"ipRangeEnd":   u32ToIP(broadcast - 1),
	}})
	routes = mustJSON([]map[string]any{{
		"target": fmt.Sprintf("%s/%d", u32ToIP(networkU32), bits),
		"via":    nil,
	}})
	return pools, routes, true
}

// netMaskU32 取前缀长度对应的 32 位网络掩码。
func netMaskU32(bits int) uint32 {
	return ^uint32(0) << (32 - bits)
}

func ipToU32(ip net.IP) uint32 {
	v4 := ip.To4()
	if v4 == nil {
		return 0
	}
	return binary.BigEndian.Uint32(v4)
}

func u32ToIP(v uint32) string {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, v)
	return net.IP(buf).String()
}

// isEmptyJSON 对齐 PHP 的 empty()：缺失、null、空数组、空对象都算空。
func isEmptyJSON(v json.RawMessage) bool {
	if len(bytes.TrimSpace(v)) == 0 {
		return true
	}
	switch string(bytes.TrimSpace(v)) {
	case "null", "[]", "{}", `""`, "0", "false":
		return true
	}
	return false
}
