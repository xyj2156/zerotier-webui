// Package zt 是 zerotier-one 内置控制器本地 REST API 的客户端与业务组装层。
//
// 契约（与原 PHP 实现一致）：
//   - 基址默认 http://127.0.0.1:9993，鉴权头 X-ZT1-Auth: <authtoken.secret>；
//   - GET /controller/network 只返回 nwid 字符串数组，需逐个取 config；
//   - 新建网络 nwid 必须 = 控制器节点地址(10hex) + 6 位随机 hex；
//   - 没有独立的 /route、/node、/peer drop 端点，路由在网络 config 内编辑；
//   - network config 与 member 走「整体透传」，只剥离服务端只读字段。
//
// 数值保真：上游 JSON 一律以 json.RawMessage 原样承载，需要改写的对象用
// map[string]json.RawMessage 处理，避免 float64 往返把 creationTime 这类
// 毫秒时间戳或 revision 写成科学计数法。
package zt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// maxBodyBytes 限制单次上游响应体积，防异常大响应拖垮内存。
const maxBodyBytes = 32 << 20

// Result 是一次上游调用的结果，Success/Message 会被上层翻译成统一信封。
type Result struct {
	Success bool
	Body    json.RawMessage // 上游原始 JSON；2xx 空体或非法 JSON 时为字面量 true
	Message string
}

// okResult / failResult 构造结果，避免各处手写字面量。
func okResult(body json.RawMessage) Result {
	return Result{Success: true, Body: body}
}

func failResult(message string) Result {
	return Result{Message: message}
}

// Client 绑定单次请求所用的控制器地址与令牌，按请求构造，无需内部缓存。
type Client struct {
	BaseURL string
	Token   string
	Timeout time.Duration
	// HTTP 可注入以便测试；为 nil 时使用共享客户端。
	HTTP *http.Client
}

var sharedHTTP = &http.Client{}

// Get 读取控制器路径（必须以 / 开头）。
func (c *Client) Get(ctx context.Context, path string) Result {
	return c.do(ctx, http.MethodGet, path, nil)
}

// Post 以 JSON 体写入控制器路径。
func (c *Client) Post(ctx context.Context, path string, body json.RawMessage) Result {
	return c.do(ctx, http.MethodPost, path, body)
}

// Delete 删除控制器路径。
func (c *Client) Delete(ctx context.Context, path string) Result {
	return c.do(ctx, http.MethodDelete, path, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body json.RawMessage) Result {
	if c.Token == "" {
		return failResult("未配置控制器令牌：请在「连接」页填写 authtoken.secret 内容")
	}
	if !strings.HasPrefix(path, "/") {
		return failResult(fmt.Sprintf("控制器路径必须以 / 开头，当前 %q", path))
	}

	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rdr)
	if err != nil {
		return failResult(err.Error())
	}
	req.Header.Set("X-ZT1-Auth", c.Token)
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = sharedHTTP
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return failResult(err.Error())
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return failResult(err.Error())
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return okResult(normalizeSuccessBody(raw))
	}

	return failResult(errorMessage(resp.StatusCode, raw))
}

// normalizeSuccessBody 与原 PHP 的 json() === null ? true : json 语义对齐：
// 空响应体与非 JSON 响应体都归一成 true。
func normalizeSuccessBody(raw []byte) json.RawMessage {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return json.RawMessage("true")
	}
	return json.RawMessage(trimmed)
}

// errorMessage 复刻原实现的取值顺序：JSON 的 message 字段 → 原始文本 → HTTP 状态码。
func errorMessage(status int, raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && json.Valid(trimmed) {
		var doc struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(trimmed, &doc) == nil && doc.Message != "" {
			return doc.Message
		}
	}
	if len(trimmed) > 0 {
		return string(trimmed)
	}
	return fmt.Sprintf("HTTP %d", status)
}
