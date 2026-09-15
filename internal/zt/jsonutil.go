package zt

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// mustJSON 序列化任意值为 json.RawMessage；失败时退化为 null，避免上层到处处理 error。
func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("null")
	}
	return json.RawMessage(b)
}

// decodeObject 把 JSON 对象解成「键 → 原始字节」，从而保住数字的字面写法
// （creationTime、revision 等大整数经 float64 往返会被写成科学计数法）。
func decodeObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	if !isObject(raw) {
		return nil, fmt.Errorf("期望是 JSON 对象，实际为 %s", truncate(string(raw), 120))
	}
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// isObject 判断原始 JSON 是否为对象。
func isObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}

// isNull 判断原始 JSON 是否为字面量 null，用于对齐 PHP 的 ?? 语义（缺失或 null 都回落）。
func isNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}

// stripKeys 删除对象里的指定键（只读字段剥离）。
func stripKeys(obj map[string]json.RawMessage, keys []string) {
	for _, k := range keys {
		delete(obj, k)
	}
}

// mergeObjects 用 override 覆盖 base，override 非法或为空时原样返回 base 的解析结果。
func mergeObjects(base, override json.RawMessage) (map[string]json.RawMessage, error) {
	merged, err := decodeObject(base)
	if err != nil {
		return nil, fmt.Errorf("缺省模板解析失败：%w", err)
	}
	if len(bytes.TrimSpace(override)) == 0 {
		return merged, nil
	}
	extra, err := decodeObject(override)
	if err != nil {
		return nil, fmt.Errorf("提交内容必须是 JSON 对象：%w", err)
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged, nil
}

// withNodeID 在成员对象尾部追加 nodeId 键，直接改写原始字节以保持其余字段顺序与写法不变。
func withNodeID(member json.RawMessage, nodeID string) json.RawMessage {
	trimmed := bytes.TrimSpace(member)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return member
	}
	key := mustJSON("nodeId")
	value := mustJSON(nodeID)
	if len(trimmed) == 2 { // 空对象 {}
		return json.RawMessage(append(append(append([]byte("{"), key...), ':'), append(value, '}')...))
	}
	head := trimmed[:len(trimmed)-1]
	out := make([]byte, 0, len(head)+len(key)+len(value)+2)
	out = append(out, head...)
	out = append(out, ',')
	out = append(out, key...)
	out = append(out, ':')
	out = append(out, value...)
	out = append(out, '}')
	return json.RawMessage(out)
}

// randomHex 生成 n 字节加密安全随机数的 hex 串（长度为 2n）。
func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成随机 nwid 后缀失败：%w", err)
	}
	return hex.EncodeToString(buf), nil
}

// truncate 截断长文本，供错误消息使用。
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
