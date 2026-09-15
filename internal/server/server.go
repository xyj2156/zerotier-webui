// Package server 把三类响应组装进一个进程：前端静态产物、路由元信息文档、控制器透传 API。
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/xyj2156/zerotier-webui/internal/buildinfo"
	"github.com/xyj2156/zerotier-webui/internal/config"
	"github.com/xyj2156/zerotier-webui/internal/forge"
	"github.com/xyj2156/zerotier-webui/internal/zt"
)

const (
	// HeaderToken 携带 authtoken.secret 内容，由前端「连接」页存入 localStorage 后随请求下发。
	HeaderToken = "X-ZT-Token"
	// HeaderBase 可选的控制器基址覆盖，用于控制器不在本机的场景。
	HeaderBase = "X-ZT-Base"
)

// maxRequestBytes 限制请求体体积。
const maxRequestBytes = 8 << 20

// Server 持有配置、路由登记表与前端产物文件系统。
// Dist 作为字段注入，便于测试用内存文件系统替身。
type Server struct {
	Cfg  config.Config
	Reg  *forge.Registry
	Dist fs.FS
	// Logf 非空时接管访问日志，测试可注入空实现保持输出干净。
	Logf func(format string, args ...any)
}

// New 构造服务器。
func New(cfg config.Config, reg *forge.Registry, dist fs.FS) *Server {
	return &Server{Cfg: cfg, Reg: reg, Dist: dist}
}

// Handler 依据路由登记表装配 mux，保证「文档」与「实际路由」同源，不会漂移。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	handlers := map[string]http.HandlerFunc{
		"info":                     s.handleInfo,
		"status":                   s.handleStatus,
		"networks":                 s.handleNetworks,
		"network-count":            s.handleNetworkCount,
		"networks.store":           s.handleNetworkCreate,
		"networks.show":            s.handleNetworkShow,
		"networks.update":          s.handleNetworkUpdate,
		"networks.destroy":         s.handleNetworkDelete,
		"networks.members":         s.handleMembers,
		"networks.members.show":    s.handleMemberShow,
		"networks.members.update":  s.handleMemberUpdate,
		"networks.members.destroy": s.handleMemberDelete,
		"raw":                      s.handleRaw,
	}

	for _, level := range s.Reg.Levels() {
		for _, route := range level.Routes {
			handler, ok := handlers[route.Name]
			if !ok {
				panic(fmt.Sprintf("路由 %s（%s）缺少处理器实现", route.Name, route.URI))
			}
			pattern := "/" + strings.TrimPrefix(route.URI, "/")
			for _, method := range route.Methods {
				// HEAD 由 net/http 依据 GET 处理器自动提供，无需重复注册。
				if method == http.MethodHead {
					continue
				}
				mux.HandleFunc(method+" "+pattern, handler)
			}
		}
	}

	prefix := strings.TrimSuffix(s.Reg.EndpointPrefix(), "/")
	mux.HandleFunc("GET "+prefix, s.handleForgeSummary)
	mux.HandleFunc("GET "+prefix+"/{level}", s.handleForgeLevel)
	mux.HandleFunc("GET /up", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /", s.handleStatic)

	return s.accessLog(mux)
}

// ==================== 前端产物 ====================

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeResult(w, zt.Result{Message: "接口不存在：" + r.URL.Path})
		return
	}
	if r.URL.Path != "/" && s.serveFile(w, r, strings.TrimPrefix(r.URL.Path, "/")) {
		return
	}
	if !s.frontendBuilt() {
		http.Error(w, "二进制内没有前端产物：先执行 pnpm build 再 go build。", http.StatusServiceUnavailable)
		return
	}
	s.serveFile(w, r, "index.html")
}

// frontendBuilt 报告当前托管的文件系统里是否含 SPA 入口 index.html。
func (s *Server) frontendBuilt() bool {
	info, err := fs.Stat(s.Dist, "index.html")
	return err == nil && !info.IsDir()
}

// serveFile 托管单个嵌入文件，返回是否成功写出。
func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, name string) bool {
	info, err := fs.Stat(s.Dist, name)
	if err != nil || info.IsDir() {
		return false
	}
	// assets/ 下的文件名带内容哈希，可长期强缓存；其余（index.html）必须每次校验。
	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeFileFS(w, r, s.Dist, name)
	return true
}

// ==================== 路由元信息 ====================

func (s *Server) handleForgeSummary(w http.ResponseWriter, r *http.Request) {
	writeRaw(w, http.StatusOK, s.Reg.SummaryDoc(0))
}

func (s *Server) handleForgeLevel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("level")
	doc, ok := s.Reg.LevelDoc(name)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "层级不存在：" + name})
		return
	}
	writeRaw(w, http.StatusOK, doc)
}

// ==================== 控制器透传 API ====================

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	svc, err := s.serviceFor(r)
	if err != nil {
		writeError(w, err)
		return
	}
	base := s.Cfg.ControllerURL
	if overridden := svc.Client.BaseURL; overridden != "" {
		base = overridden
	}
	writeResult(w, zt.Result{Success: true, Body: mustJSON(map[string]any{
		"version":        buildinfo.Version,
		"commit":         buildinfo.Commit,
		"listen":         s.Cfg.Listen,
		"controller":     base,
		"tokenProvided":  svc.Client.Token != "",
		"frontendBuilt":  s.frontendBuilt(),
		"runtime":        fmt.Sprintf("%s/%s %s", runtime.GOOS, runtime.GOARCH, runtime.Version()),
		"timeoutSeconds": int(s.Cfg.Timeout / time.Second),
	})})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.Status(r.Context()) })
}

func (s *Server) handleNetworks(w http.ResponseWriter, r *http.Request) {
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.Networks(r.Context()) })
}

func (s *Server) handleNetworkCount(w http.ResponseWriter, r *http.Request) {
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.NetworksCount(r.Context()) })
}

func (s *Server) handleNetworkShow(w http.ResponseWriter, r *http.Request) {
	nwid := r.PathValue("nwid")
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.GetNetwork(r.Context(), nwid) })
}

func (s *Server) handleNetworkCreate(w http.ResponseWriter, r *http.Request) {
	payload, err := s.body(r)
	if err != nil {
		writeError(w, err)
		return
	}
	normalized, err := zt.NormalizeNetworkInput(payload)
	if err != nil {
		writeError(w, err)
		return
	}
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.CreateNetwork(r.Context(), normalized) })
}

func (s *Server) handleNetworkUpdate(w http.ResponseWriter, r *http.Request) {
	nwid := r.PathValue("nwid")
	payload, err := s.body(r)
	if err != nil {
		writeError(w, err)
		return
	}
	normalized, err := zt.NormalizeNetworkInput(payload)
	if err != nil {
		writeError(w, err)
		return
	}
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.UpdateNetwork(r.Context(), nwid, normalized) })
}

func (s *Server) handleNetworkDelete(w http.ResponseWriter, r *http.Request) {
	nwid := r.PathValue("nwid")
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.DeleteNetwork(r.Context(), nwid) })
}

func (s *Server) handleMembers(w http.ResponseWriter, r *http.Request) {
	nwid := r.PathValue("nwid")
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.Members(r.Context(), nwid) })
}

func (s *Server) handleMemberShow(w http.ResponseWriter, r *http.Request) {
	nwid, id := r.PathValue("nwid"), r.PathValue("id")
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.GetMember(r.Context(), nwid, id) })
}

func (s *Server) handleMemberUpdate(w http.ResponseWriter, r *http.Request) {
	nwid, id := r.PathValue("nwid"), r.PathValue("id")
	payload, err := s.body(r)
	if err != nil {
		writeError(w, err)
		return
	}
	s.call(w, r, func(svc *zt.Service) zt.Result {
		return svc.UpdateMember(r.Context(), nwid, id, json.RawMessage(payload))
	})
}

func (s *Server) handleMemberDelete(w http.ResponseWriter, r *http.Request) {
	nwid, id := r.PathValue("nwid"), r.PathValue("id")
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.DeleteMember(r.Context(), nwid, id) })
}

func (s *Server) handleRaw(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	s.call(w, r, func(svc *zt.Service) zt.Result { return svc.RawGet(r.Context(), path) })
}

// ==================== 请求与响应工具 ====================

// call 统一处理凭据解析与信封写出，处理器只提供一次 Service 调用。
func (s *Server) call(w http.ResponseWriter, r *http.Request, fn func(*zt.Service) zt.Result) {
	svc, err := s.serviceFor(r)
	if err != nil {
		writeError(w, err)
		return
	}
	writeResult(w, fn(svc))
}

// serviceFor 从请求头取出令牌与可选控制器基址，构造本次请求的服务实例。
func (s *Server) serviceFor(r *http.Request) (*zt.Service, error) {
	token := strings.TrimSpace(r.Header.Get(HeaderToken))
	if token == "" {
		token = strings.TrimSpace(s.Cfg.Token)
	}
	base := s.Cfg.ControllerURL
	if override := strings.TrimSpace(r.Header.Get(HeaderBase)); override != "" {
		normalized, err := config.NormalizeControllerURL(override)
		if err != nil {
			return nil, err
		}
		base = normalized
	}
	client := &zt.Client{BaseURL: base, Token: token, Timeout: s.Cfg.Timeout}
	return zt.NewService(client, s.Cfg.NodeAddress), nil
}

// body 读取请求体原始字节，超限即拒绝。
func (s *Server) body(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取请求体失败：%w", err)
	}
	if len(raw) > maxRequestBytes {
		return nil, fmt.Errorf("请求体超过 %d 字节上限", maxRequestBytes)
	}
	return raw, nil
}

// writeResult 输出统一信封 { status, message, result }，HTTP 状态码恒为 200，
// 与原 Laravel 实现一致（业务失败也用 status=1 表达）。
func writeResult(w http.ResponseWriter, res zt.Result) {
	result := json.RawMessage("null")
	if res.Success && len(res.Body) > 0 {
		result = res.Body
	}
	status := 1
	if res.Success {
		status = 0
	}
	writeRaw(w, http.StatusOK, mustJSON(map[string]any{
		"status":  status,
		"message": res.Message,
		"result":  result,
	}))
}

// writeError 把本地校验错误写成同一信封。
func writeError(w http.ResponseWriter, err error) {
	writeRaw(w, http.StatusOK, mustJSON(map[string]any{
		"status":  1,
		"message": err.Error(),
		"result":  json.RawMessage("null"),
	}))
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	writeRaw(w, code, mustJSON(body))
}

func writeRaw(w http.ResponseWriter, code int, body json.RawMessage) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}

// mustJSON 集中一份序列化兜底，避免各处重复 err 处理。
func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{"status":1,"message":"响应序列化失败","result":null}`)
	}
	return json.RawMessage(b)
}

// accessLog 给每个请求补一行访问日志，单文件跑起来时能看见控制台的动静。
func (s *Server) accessLog(next http.Handler) http.Handler {
	logf := s.Logf
	if logf == nil {
		logf = func(format string, args ...any) { fmt.Printf(format, args...) }
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logf("%s %s %s %d %s\n", time.Now().Format("15:04:05"), r.Method, r.URL.Path, rec.status, time.Since(started).Round(time.Millisecond))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
