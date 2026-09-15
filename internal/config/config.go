// Package config 负责命令行参数与环境变量的解析归一。
//
// 设计约束（单执行文件形态）：二进制自身不落盘任何凭据。控制器令牌默认由浏览器
// 在「连接」页输入并保存在 localStorage，随每个 /api 请求以请求头下发；这里的
// Token 只是无浏览器场景（curl、健康检查）的兜底，可用 --token / ZT_TOKEN 显式给出。
package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	DefaultListen        = "127.0.0.1:9090"
	DefaultControllerURL = "http://127.0.0.1:9993"
	DefaultTimeout       = 15 * time.Second
)

// Config 是解析完成后的运行期配置。
type Config struct {
	Listen        string        // 桥自身监听地址
	ControllerURL string        // zerotier-one 本地控制器 API 基址
	Token         string        // 兜底令牌（可为空，交由前端请求头提供）
	NodeAddress   string        // 控制器节点地址覆盖（10 位 hex）；空则查 /status
	Timeout       time.Duration // 上游请求超时
	OpenBrowser   bool          // 启动后自动打开浏览器
	ShowVersion   bool          // 只打印版本后退出
}

// Parse 用给定参数集解析 args，返回校验后的配置。
// flag.ErrHelp 会原样上抛，由调用方决定退出码。
func Parse(args []string) (Config, error) {
	cfg := Config{
		Listen:        envOr("ZT_LISTEN", DefaultListen),
		ControllerURL: envOr("ZT_URL", DefaultControllerURL),
		Token:         strings.TrimSpace(os.Getenv("ZT_TOKEN")),
		NodeAddress:   strings.TrimSpace(os.Getenv("ZT_NODE_ADDRESS")),
		Timeout:       DefaultTimeout,
	}

	fs := flag.NewFlagSet("zerotier-webui", flag.ContinueOnError)
	fs.StringVar(&cfg.Listen, "listen", cfg.Listen, "监听地址（局域网访问需显式改为 0.0.0.0:端口，注意暴露风险）")
	fs.StringVar(&cfg.ControllerURL, "url", cfg.ControllerURL, "zerotier-one 本地控制器 API 基址")
	fs.StringVar(&cfg.Token, "token", cfg.Token, "兜底 X-ZT1-Auth 令牌（正常用法由前端「连接」页提供）")
	fs.StringVar(&cfg.NodeAddress, "node-address", cfg.NodeAddress, "控制器节点地址 10 位 hex，留空则取 /status 的 address")
	fs.DurationVar(&cfg.Timeout, "timeout", DefaultTimeout, "上游请求超时，如 15s")
	fs.BoolVar(&cfg.OpenBrowser, "open", false, "启动后自动打开默认浏览器")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "打印版本信息后退出")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if cfg.Timeout <= 0 {
		return Config{}, fmt.Errorf("timeout 必须为正数，当前为 %s", cfg.Timeout)
	}

	url, err := NormalizeControllerURL(cfg.ControllerURL)
	if err != nil {
		return Config{}, err
	}
	cfg.ControllerURL = url

	if cfg.NodeAddress != "" {
		cfg.NodeAddress = strings.ToLower(strings.TrimSpace(cfg.NodeAddress))
		if len(cfg.NodeAddress) != 10 {
			return Config{}, fmt.Errorf("node-address 必须是 10 位 hex 节点地址，当前 %q", cfg.NodeAddress)
		}
	}

	return cfg, nil
}

// NormalizeControllerURL 校验并去掉末尾斜杠，只放行 http/https 绝对地址。
// 该函数同时用于校验前端下发的 X-ZT-Base 覆盖值，防 SSRF。
func NormalizeControllerURL(raw string) (string, error) {
	url := strings.TrimSpace(raw)
	if url == "" {
		return "", fmt.Errorf("控制器地址不能为空")
	}
	low := strings.ToLower(url)
	if !strings.HasPrefix(low, "http://") && !strings.HasPrefix(low, "https://") {
		return "", fmt.Errorf("控制器地址必须以 http:// 或 https:// 开头，当前 %q", raw)
	}
	return strings.TrimRight(url, "/"), nil
}

// envOr 读取环境变量，空值（含仅空白）回落到默认值。
func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
