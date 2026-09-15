package forge

import (
	"encoding/json"
	"strings"
	"testing"
)

func testRegistry() *Registry {
	public := []Route{{Name: "info", URI: "api/info", Methods: []string{"GET", "HEAD"}}}
	admin := []Route{
		{Name: "status", URI: "api/status", Methods: []string{"GET", "HEAD"}},
		{Name: "networks.show", URI: "api/networks/{nwid}", Methods: []string{"GET", "HEAD"}, Params: []string{"nwid"}},
		{Name: "networks.destroy", URI: "api/networks/{nwid}", Methods: []string{"DELETE"}, Params: []string{"nwid"}},
	}
	return NewRegistry("/_forge/routes", BuiltinLevels(public, admin)...)
}

// TestLevelDocMatchesLaravelShape 逐字节比对 route-forge/laravel 实测下发的层级文档结构
// （样本抓自 F:\tmp\forge-admin.json，字段名与顺序一致）。
func TestLevelDocMatchesLaravelShape(t *testing.T) {
	doc, ok := testRegistry().LevelDoc("admin")
	if !ok {
		t.Fatal("admin 层级应存在")
	}
	want := `{"level":"admin","routes":{` +
		`"status":{"uri":"api/status","methods":["GET","HEAD"],"parameters":[],"parameter_defaults":{}},` +
		`"networks.show":{"uri":"api/networks/{nwid}","methods":["GET","HEAD"],"parameters":["nwid"],"parameter_defaults":{}},` +
		`"networks.destroy":{"uri":"api/networks/{nwid}","methods":["DELETE"],"parameters":["nwid"],"parameter_defaults":{}}` +
		`}}`
	if string(doc) != want {
		t.Errorf("层级文档与 Laravel 形态不符：\n实际 %s\n期望 %s", doc, want)
	}
}

func TestLevelDocUnknownLevel(t *testing.T) {
	if _, ok := testRegistry().LevelDoc("nope"); ok {
		t.Error("未知层级应返回 false")
	}
	// 空层级也要能出文档，摘要里声明的 client/manage/unassigned 都必须是可取的
	for _, name := range []string{"client", "manage", "unassigned"} {
		doc, ok := testRegistry().LevelDoc(name)
		if !ok || !strings.Contains(string(doc), `"routes":{}`) {
			t.Errorf("空层级 %s 文档不符：%s", name, doc)
		}
	}
}

func TestSummaryDocFields(t *testing.T) {
	reg := testRegistry()
	raw := reg.SummaryDoc(3600)

	var doc struct {
		SchemeVersion int                        `json:"schemeVersion"`
		Levels        map[string]json.RawMessage `json:"levels"`
		Config        map[string]any             `json:"config"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("摘要文档不是合法 JSON：%v", err)
	}
	if doc.SchemeVersion != SchemeVersion {
		t.Errorf("schemeVersion = %d，期望 %d", doc.SchemeVersion, SchemeVersion)
	}
	if len(doc.Levels) != 5 {
		t.Errorf("层级数 = %d，期望 5", len(doc.Levels))
	}

	var admin struct {
		Description string `json:"description"`
		Load        string `json:"load"`
		RouteCount  int    `json:"route_count"`
		Route       struct {
			URI     string   `json:"uri"`
			Methods []string `json:"methods"`
		} `json:"route"`
	}
	if err := json.Unmarshal(doc.Levels["admin"], &admin); err != nil {
		t.Fatal(err)
	}
	if admin.Load != "lazy" || admin.RouteCount != 3 {
		t.Errorf("admin 摘要不符：%+v", admin)
	}
	if admin.Route.URI != "/_forge/routes/admin" {
		t.Errorf("层级端点 URI 不符：%s", admin.Route.URI)
	}

	// config.url_prefix 必须是 null（与 Laravel 默认下发一致），cache_ttl 由调用方给出
	if !strings.Contains(string(raw), `"url_prefix":null`) {
		t.Errorf("url_prefix 应为 null：%s", raw)
	}
	if !strings.Contains(string(raw), `"cache_ttl":3600`) || !strings.Contains(string(raw), `"strict_mode":false`) {
		t.Errorf("config 字段不符：%s", raw)
	}
}

func TestSummaryKeepsLevelOrder(t *testing.T) {
	raw := string(testRegistry().SummaryDoc(0))
	public := strings.Index(raw, `"public":`)
	admin := strings.Index(raw, `"admin":`)
	if public < 0 || admin < 0 || public > admin {
		t.Errorf("levels 应沿用注册顺序（public 在 admin 前）：%s", raw)
	}
}

func TestRouteLookup(t *testing.T) {
	reg := testRegistry()
	if route, ok := reg.Route("networks.show"); !ok || len(route.Params) != 1 {
		t.Errorf("按名查路由失败：%+v %v", route, ok)
	}
	if _, ok := reg.Route("不存在"); ok {
		t.Error("未知路由名不应命中")
	}
}
