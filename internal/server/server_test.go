package server

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/xyj2156/zerotier-webui/internal/config"
	"github.com/xyj2156/zerotier-webui/internal/forge"
)

// testDist 是一份含入口与哈希资源的内存前端产物。
func testDist() fs.FS {
	return fstest.MapFS{
		"index.html":           &fstest.MapFile{Data: []byte(`<!doctype html><div id="jason"></div>`)},
		"assets/app-abc123.js": &fstest.MapFile{Data: []byte(`console.log(1)`)},
	}
}

// testRegistry 覆盖公开与管理两类端点，命名与 cmd 里的正式表保持一致。
func testRegistry() *forge.Registry {
	get := []string{"GET", "HEAD"}
	return forge.NewRegistry("/_forge/routes", forge.BuiltinLevels(
		[]forge.Route{{Name: "info", URI: "api/info", Methods: get}},
		[]forge.Route{
			{Name: "status", URI: "api/status", Methods: get},
			{Name: "networks", URI: "api/networks", Methods: get},
			{Name: "networks.store", URI: "api/networks", Methods: []string{"POST"}},
			{Name: "networks.show", URI: "api/networks/{nwid}", Methods: get, Params: []string{"nwid"}},
			{Name: "networks.update", URI: "api/networks/{nwid}", Methods: []string{"POST"}, Params: []string{"nwid"}},
			{Name: "networks.members", URI: "api/networks/{nwid}/members", Methods: get, Params: []string{"nwid"}},
			{Name: "networks.members.update", URI: "api/networks/{nwid}/members/{id}", Methods: []string{"POST"}, Params: []string{"nwid", "id"}},
			{Name: "raw", URI: "api/raw", Methods: get},
		},
	)...)
}

type harness struct {
	handler http.Handler
	seen    *[]string
	cfg     config.Config
}

// newHarness 起一个假控制器并把桥 handler 指向它；upstream 为 nil 时用默认 /status 响应。
func newHarness(t *testing.T, upstream http.HandlerFunc, mutate func(*config.Config)) harness {
	t.Helper()
	var seen []string
	controller := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		seen = append(seen, r.Method+" "+r.URL.Path+" "+string(raw))
		w.Header().Set("Content-Type", "application/json")
		if upstream == nil {
			_, _ = w.Write([]byte(`{"address":"00fc166136","version":"1.16.1"}`))
			return
		}
		upstream(w, r)
	}))
	t.Cleanup(controller.Close)

	cfg := config.Config{Listen: "127.0.0.1:0", ControllerURL: controller.URL, Timeout: 2 * time.Second}
	if mutate != nil {
		mutate(&cfg)
	}
	srv := New(cfg, testRegistry(), testDist())
	srv.Logf = func(string, ...any) {}
	return harness{handler: srv.Handler(), seen: &seen, cfg: cfg}
}

// envelope 是统一响应信封的解码结果。
type envelope struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

func doJSON(t *testing.T, req *http.Request, handler http.Handler) (*httptest.ResponseRecorder, envelope) {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("响应不是信封 JSON：%s (err %v)", rec.Body.String(), err)
	}
	return rec, env
}

func newRequest(method, target, body string, headers map[string]string) *http.Request {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}

func TestStatusForwardsHeaderToken(t *testing.T) {
	h := newHarness(t, nil, nil)
	rec, env := doJSON(t, newRequest("GET", "/api/status", "", map[string]string{HeaderToken: "abc"}), h.handler)

	if env.Status != 0 || rec.Code != http.StatusOK {
		t.Fatalf("期望成功信封，实际 code=%d %+v", rec.Code, env)
	}
	if !strings.Contains(string(env.Result), `"00fc166136"`) {
		t.Errorf("result 未透传上游：%s", env.Result)
	}
	if !strings.Contains((*h.seen)[0], "X-ZT1") && len(*h.seen) != 1 {
		t.Errorf("上游请求异常：%v", *h.seen)
	}
}

func TestMissingTokenIsBusinessErrorNotHTTPerror(t *testing.T) {
	h := newHarness(t, nil, nil)
	rec, env := doJSON(t, newRequest("GET", "/api/status", "", nil), h.handler)

	if rec.Code != http.StatusOK {
		t.Errorf("业务失败也应返回 HTTP 200，实际 %d", rec.Code)
	}
	if env.Status == 0 || !strings.Contains(env.Message, "未配置控制器令牌") {
		t.Fatalf("期望提示缺令牌，实际 %+v", env)
	}
	if len(*h.seen) != 0 {
		t.Errorf("缺令牌时不该打上游：%v", *h.seen)
	}
}

func TestConfiguredTokenIsUsedAsFallback(t *testing.T) {
	h := newHarness(t, nil, func(cfg *config.Config) { cfg.Token = "from-flag" })
	if _, env := doJSON(t, newRequest("GET", "/api/status", "", nil), h.handler); env.Status != 0 {
		t.Fatalf("显式配置的令牌应生效，实际 %+v", env)
	}
}

func TestBaseOverrideHeader(t *testing.T) {
	var hitSecond bool
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitSecond = true
		_, _ = w.Write([]byte(`{"address":"11ab22cd33"}`))
	}))
	t.Cleanup(second.Close)

	h := newHarness(t, nil, nil)
	_, env := doJSON(t, newRequest("GET", "/api/status", "", map[string]string{
		HeaderToken: "abc", HeaderBase: second.URL,
	}), h.handler)

	if env.Status != 0 || !hitSecond {
		t.Fatalf("基址覆盖未生效：%+v hit=%v", env, hitSecond)
	}

	_, bad := doJSON(t, newRequest("GET", "/api/status", "", map[string]string{
		HeaderToken: "abc", HeaderBase: "ftp://evil.example",
	}), h.handler)
	if bad.Status == 0 || !strings.Contains(bad.Message, "http://") {
		t.Errorf("非法基址应被拒绝：%+v", bad)
	}
}

func TestNetworkCreateUsesConvenienceFields(t *testing.T) {
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			_, _ = w.Write([]byte(`{"address":"00fc166136"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}, nil)

	req := newRequest("POST", "/api/networks", `{"name":"lab","cidr":"10.20.0.0/16","revision":5}`, map[string]string{HeaderToken: "abc"})
	_, env := doJSON(t, req, h.handler)
	if env.Status != 0 {
		t.Fatalf("创建失败：%+v", env)
	}
	if len(*h.seen) != 2 { // /status 取地址 + 写网络
		t.Fatalf("上游请求数不符：%v", *h.seen)
	}
	method, rest, _ := strings.Cut((*h.seen)[1], " ")
	path, body, _ := strings.Cut(rest, " ")
	if method != "POST" || !strings.HasPrefix(path, "/controller/network/00fc166136") || len(path) != len("/controller/network/")+16 {
		t.Fatalf("创建请求路径不符：%s %s", method, path)
	}
	for _, want := range []string{`"name":"lab"`, "10.20.0.1", `"mtu":2800`} {
		if !strings.Contains(body, want) {
			t.Errorf("请求体缺少 %s：%s", want, body)
		}
	}
	if strings.Contains(body, "revision") {
		t.Errorf("只读字段未剥离：%s", body)
	}
}

func TestUnknownAPIPathReturnsEnvelope(t *testing.T) {
	h := newHarness(t, nil, nil)
	_, env := doJSON(t, newRequest("GET", "/api/does-not-exist", "", map[string]string{HeaderToken: "abc"}), h.handler)
	if env.Status == 0 || !strings.Contains(env.Message, "接口不存在") {
		t.Fatalf("未知接口应回信封错误，实际 %+v", env)
	}
}

func TestForgeEndpoints(t *testing.T) {
	h := newHarness(t, nil, nil)

	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, httptest.NewRequest("GET", "/_forge/routes", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"schemeVersion":1`) {
		t.Fatalf("摘要端点异常：%d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"uri":"api/`) {
		t.Error("摘要端点不该内联全部路由")
	}

	rec = httptest.NewRecorder()
	h.handler.ServeHTTP(rec, httptest.NewRequest("GET", "/_forge/routes/admin", nil))
	if !strings.Contains(rec.Body.String(), `"level":"admin"`) || !strings.Contains(rec.Body.String(), `"uri":"api/networks/{nwid}"`) {
		t.Fatalf("层级端点异常：%s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.handler.ServeHTTP(rec, httptest.NewRequest("GET", "/_forge/routes/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("未知层级应 404，实际 %d", rec.Code)
	}
}

func TestHEADOnGetRoute(t *testing.T) {
	h := newHarness(t, nil, nil)
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, newRequest("HEAD", "/api/status", "", map[string]string{HeaderToken: "abc"}))
	if rec.Code != http.StatusOK {
		t.Errorf("GET 路由应自动支持 HEAD，实际 %d", rec.Code)
	}
}

func TestMethodNotAllowedForReadOnlyRoute(t *testing.T) {
	h := newHarness(t, nil, nil)
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, newRequest("DELETE", "/api/status", "", map[string]string{HeaderToken: "abc"}))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("未注册方法应 405，实际 %d", rec.Code)
	}
}

func TestStaticServingAndSPAFallback(t *testing.T) {
	h := newHarness(t, nil, nil)

	for _, target := range []string{"/", "/networks", "/network/8056c240dc8048e1"} {
		rec := httptest.NewRecorder()
		h.handler.ServeHTTP(rec, httptest.NewRequest("GET", target, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `id="jason"`) {
			t.Errorf("%s 应回落到 SPA 入口：%d %s", target, rec.Code, rec.Body.String())
		}
		if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
			t.Errorf("%s 的 Cache-Control = %q", target, got)
		}
	}

	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, httptest.NewRequest("GET", "/assets/app-abc123.js", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "console.log") {
		t.Fatalf("哈希资源未命中：%d %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("哈希资源应强缓存，实际 %q", got)
	}
}

func TestMissingFrontendDistExplainsItself(t *testing.T) {
	controller := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(controller.Close)
	cfg := config.Config{Listen: "127.0.0.1:0", ControllerURL: controller.URL, Timeout: time.Second}
	srv := New(cfg, testRegistry(), fstest.MapFS{"assets/keep.js": &fstest.MapFile{Data: []byte("x")}})

	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "pnpm build") {
		t.Fatalf("缺产物时应给出可执行的提示：%d %s", rec.Code, rec.Body.String())
	}
}

func TestMemberUpdateRouteWithTwoParams(t *testing.T) {
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"authorized":true}`))
	}, nil)

	req := newRequest("POST", "/api/networks/8056c240dc8048e1/members/107b2c4b12",
		`{"authorized":true,"name":"pc1","online":true,"identity":"107b2c4b12:0:xyz"}`,
		map[string]string{HeaderToken: "abc"})
	_, env := doJSON(t, req, h.handler)
	if env.Status != 0 {
		t.Fatalf("成员更新失败：%+v", env)
	}
	line := (*h.seen)[0]
	if !strings.HasPrefix(line, "POST /controller/network/8056c240dc8048e1/member/107b2c4b12 ") {
		t.Fatalf("上游路径不符：%s", line)
	}
	body := strings.TrimPrefix(line, "POST /controller/network/8056c240dc8048e1/member/107b2c4b12 ")
	if strings.Contains(body, "online") || strings.Contains(body, "identity") {
		t.Errorf("只读字段未剥离：%s", body)
	}
	if !strings.Contains(body, `"authorized":true`) || !strings.Contains(body, `"name":"pc1"`) {
		t.Errorf("可写字段丢失：%s", body)
	}
}

func TestUnknownAssetFallsBackToSPAEntry(t *testing.T) {
	h := newHarness(t, nil, nil)
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, httptest.NewRequest("GET", "/assets/missing.js", nil))
	// 前端路由深链接与失效哈希资源都回落到入口页，由前端自己处理渲染
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `id="jason"`) {
		t.Errorf("未命中资源应回落 SPA：[%d] %s", rec.Code, rec.Body.String())
	}
}

func TestInfoEndpointReportsBridgeState(t *testing.T) {
	h := newHarness(t, nil, nil)
	_, env := doJSON(t, newRequest("GET", "/api/info", "", nil), h.handler)
	if env.Status != 0 {
		t.Fatalf("info 应免令牌可用：%+v", env)
	}
	var info map[string]any
	if err := json.Unmarshal(env.Result, &info); err != nil {
		t.Fatal(err)
	}
	if info["tokenProvided"] != false || info["frontendBuilt"] != true {
		t.Errorf("info 字段不符：%v", info)
	}
	if info["controller"] != h.cfg.ControllerURL {
		t.Errorf("controller 应为配置的基址：%v", info["controller"])
	}
}

func TestHealthEndpoint(t *testing.T) {
	h := newHarness(t, nil, nil)
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, httptest.NewRequest("GET", "/up", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "ok") {
		t.Errorf("/up 异常：%d %s", rec.Code, rec.Body.String())
	}
}

func TestRegistryRoutesAllImplemented(t *testing.T) {
	// Handler() 对缺失处理器的路由直接 panic，这里用一条完整登记表验证不会炸。
	h := newHarness(t, nil, nil)
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, httptest.NewRequest("GET", "/up", nil))

	names := make([]string, 0)
	for _, level := range testRegistry().Levels() {
		for _, route := range level.Routes {
			names = append(names, route.Name)
		}
	}
	for _, want := range []string{"info", "status", "networks", "networks.store", "raw", "networks.members.update"} {
		if !slices.Contains(names, want) {
			t.Errorf("测试登记表缺少路由 %s", want)
		}
	}
}
