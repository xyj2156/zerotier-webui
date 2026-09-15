package config

import (
	"errors"
	"flag"
	"testing"
	"time"
)

func TestParseDefaults(t *testing.T) {
	t.Setenv("ZT_LISTEN", "")
	t.Setenv("ZT_URL", "")
	t.Setenv("ZT_TOKEN", "")

	cfg, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != DefaultListen || cfg.ControllerURL != DefaultControllerURL {
		t.Errorf("默认值不符：%+v", cfg)
	}
	if cfg.Timeout != DefaultTimeout {
		t.Errorf("默认超时 = %s", cfg.Timeout)
	}
}

func TestParseFlagsAndEnvFallback(t *testing.T) {
	t.Setenv("ZT_URL", "http://10.0.0.7:9993/")
	t.Setenv("ZT_TOKEN", "  secret  ")
	cfg, err := Parse([]string{"-listen", "0.0.0.0:8080", "-timeout", "3s", "-open"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != "0.0.0.0:8080" || !cfg.OpenBrowser || cfg.Timeout != 3*time.Second {
		t.Errorf("参数解析不符：%+v", cfg)
	}
	if cfg.ControllerURL != "http://10.0.0.7:9993" {
		t.Errorf("环境变量基址应去掉末尾斜杠：%s", cfg.ControllerURL)
	}
	if cfg.Token != "secret" {
		t.Errorf("令牌应去空白：%q", cfg.Token)
	}

	// 命令行优先级高于环境变量
	cfg, err = Parse([]string{"-url", "http://127.0.0.1:19993"})
	if err != nil || cfg.ControllerURL != "http://127.0.0.1:19993" {
		t.Errorf("命令行应覆盖环境变量：%+v %v", cfg, err)
	}
}

func TestParseRejectsBadInput(t *testing.T) {
	if _, err := Parse([]string{"-url", "127.0.0.1:9993"}); err == nil {
		t.Error("缺少 scheme 的基址应报错")
	}
	if _, err := Parse([]string{"-timeout", "0"}); err == nil {
		t.Error("非正超时应报错")
	}
	if _, err := Parse([]string{"-node-address", "abc"}); err == nil {
		t.Error("节点地址长度不对应报错")
	}
	cfg, err := Parse([]string{"-node-address", "AABBCCDDEE"})
	if err != nil || cfg.NodeAddress != "aabbccddee" {
		t.Errorf("合法地址应小写保留：%q %v", cfg.NodeAddress, err)
	}
	if _, err := Parse([]string{"-h"}); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-h 应上抛 ErrHelp，实际 %v", err)
	}
}

func TestNormalizeControllerURL(t *testing.T) {
	cases := map[string]string{
		"http://127.0.0.1:9993/":      "http://127.0.0.1:9993",
		"  https://ctrl.example:9993": "https://ctrl.example:9993",
	}
	for in, want := range cases {
		got, err := NormalizeControllerURL(in)
		if err != nil || got != want {
			t.Errorf("NormalizeControllerURL(%q) = %q, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "   ", "ftp://x", "127.0.0.1:9993", "//evil.example"} {
		if _, err := NormalizeControllerURL(bad); err == nil {
			t.Errorf("应拒绝 %q", bad)
		}
	}
}
