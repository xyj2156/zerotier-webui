// Command zerotier-webui 是单执行文件形态的 zerotier-one 本地控制器管理面板：
// 前端产物经 go:embed 内置，HTTP 层同时提供静态页面、路由元信息文档与控制器透传 API。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/xyj2156/zerotier-webui/internal/buildinfo"
	"github.com/xyj2156/zerotier-webui/internal/config"
	"github.com/xyj2156/zerotier-webui/internal/forge"
	"github.com/xyj2156/zerotier-webui/internal/server"
	"github.com/xyj2156/zerotier-webui/web"
)

const endpointPrefix = "/_forge/routes"

func main() {
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "参数错误:", err)
		os.Exit(2)
	}
	if cfg.ShowVersion {
		fmt.Println(buildinfo.String())
		return
	}

	listener, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "监听 %s 失败：%v\n", cfg.Listen, err)
		os.Exit(1)
	}

	url := accessURL(listener.Addr().String())
	srv := server.New(cfg, forge.NewRegistry(endpointPrefix, forge.BuiltinLevels(publicRoutes(), adminRoutes())...), web.Dist())
	httpServer := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	printBanner(url, cfg)

	go func() {
		if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "服务异常退出:", err)
			os.Exit(1)
		}
	}()

	if cfg.OpenBrowser {
		openBrowser(url)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	fmt.Println("已退出。")
}

// publicRoutes 是免令牌即可访问的接口（只用于探测桥自身状态）。
func publicRoutes() []forge.Route {
	return []forge.Route{
		{Name: "info", URI: "api/info", Methods: []string{"GET", "HEAD"}},
	}
}

// adminRoutes 是需要控制器令牌的管理接口，命名与 URI 保持与原 Laravel 版一致，
// 使前端 call(name) 调用点零改动。
func adminRoutes() []forge.Route {
	get := []string{"GET", "HEAD"}
	post := []string{"POST"}
	del := []string{"DELETE"}
	nwid := []string{"nwid"}
	node := []string{"nwid", "id"}

	return []forge.Route{
		{Name: "status", URI: "api/status", Methods: get},
		{Name: "peers", URI: "api/peers", Methods: get},
		{Name: "networks", URI: "api/networks", Methods: get},
		{Name: "network-count", URI: "api/network-count", Methods: get},
		{Name: "networks.store", URI: "api/networks", Methods: post},
		{Name: "networks.show", URI: "api/networks/{nwid}", Methods: get, Params: nwid},
		{Name: "networks.update", URI: "api/networks/{nwid}", Methods: post, Params: nwid},
		{Name: "networks.destroy", URI: "api/networks/{nwid}", Methods: del, Params: nwid},
		{Name: "networks.members", URI: "api/networks/{nwid}/members", Methods: get, Params: nwid},
		{Name: "networks.members.show", URI: "api/networks/{nwid}/members/{id}", Methods: get, Params: node},
		{Name: "networks.members.update", URI: "api/networks/{nwid}/members/{id}", Methods: post, Params: node},
		{Name: "networks.members.destroy", URI: "api/networks/{nwid}/members/{id}", Methods: del, Params: node},
		{Name: "raw", URI: "api/raw", Methods: get},
	}
}

// accessURL 把监听地址整理成可点击的 URL，通配绑定回落到 127.0.0.1 便于直接访问。
func accessURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%s", host, port)
}

func printBanner(url string, cfg config.Config) {
	fmt.Printf("zerotier-webui %s\n", buildinfo.Version)
	fmt.Printf("管理面板: %s\n", url)
	fmt.Printf("控制器:   %s\n", cfg.ControllerURL)
	if cfg.Token == "" {
		fmt.Println("令牌:     未内置，首次打开页面时在「连接」页填入 authtoken.secret")
	}
	if !web.Built() {
		fmt.Println("提示:     本二进制未嵌入前端产物，请先 pnpm build 后重新 go build")
	}
	if listensOnAllInterfaces(cfg.Listen) {
		fmt.Println("安全提示: 监听地址包含通配网卡，同网段任何人都能访问这个面板")
	}
}

// listensOnAllInterfaces 报告监听地址是否绑到了所有网卡（而非仅本机）。
func listensOnAllInterfaces(listen string) bool {
	host, _, err := net.SplitHostPort(listen)
	return err == nil && (host == "0.0.0.0" || host == "::" || host == "*")
}

// openBrowser 尽力打开默认浏览器，失败不打断服务。
func openBrowser(url string) {
	var name string
	var args []string
	switch runtime.GOOS {
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		name, args = "open", []string{url}
	default:
		name, args = "xdg-open", []string{url}
		if _, err := exec.LookPath(name); err != nil {
			return
		}
	}
	_ = exec.Command(name, args...).Start()
}
