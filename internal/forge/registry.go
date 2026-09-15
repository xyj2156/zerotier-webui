// Package forge 提供路由元登记表：同一份表既驱动 HTTP 路由注册，也生成
// 前端 @route-forge/vue builtin adapter 消费的 /_forge/routes 文档。
//
// 文档格式与 route-forge/laravel 下发给前端的结构逐字段对齐（见 F:\tmp\forge-*.json
// 抓取的实测样本）：摘要端点带 schemeVersion/levels/config，层级端点带 level/routes，
// 每条路由是 {uri, methods, parameters, parameter_defaults}。
package forge

import (
	"bytes"
	"encoding/json"
)

// SchemeVersion 是摘要端点的格式版本，与 Laravel 侧 FORGE_SCHEME_VERSION 默认值一致。
const SchemeVersion = 1

// Route 是一条对外暴露的 API 路由。URI 不带前导斜杠（沿用 Laravel 的下发写法，
// 如 "api/networks/{nwid}"），Name 是前端 call(name, ...) 使用的路由名。
type Route struct {
	Name    string
	URI     string
	Methods []string
	Params  []string
}

// Level 是一个层级，Load 为 eager/lazy，决定前端是否预取该层级。
type Level struct {
	Name        string
	Description string
	Load        string
	Routes      []Route
}

// Registry 持有全部层级，是路由表与元信息文档的唯一事实源。
type Registry struct {
	endpointPrefix string
	levels         []Level
}

// NewRegistry 用给定前缀与层级构造登记表。
func NewRegistry(endpointPrefix string, levels ...Level) *Registry {
	return &Registry{endpointPrefix: endpointPrefix, levels: levels}
}

// Levels 返回层级定义（注册顺序）。
func (r *Registry) Levels() []Level { return r.levels }

// EndpointPrefix 返回元信息端点前缀。
func (r *Registry) EndpointPrefix() string { return r.endpointPrefix }

// Route 按名字查路由，供处理器注册时定位 pattern；不存在时返回 false。
func (r *Registry) Route(name string) (Route, bool) {
	for _, level := range r.levels {
		for _, route := range level.Routes {
			if route.Name == name {
				return route, true
			}
		}
	}
	return Route{}, false
}

// metaJSON 是一条路由在层级文档里的形态。
type metaJSON struct {
	URI               string         `json:"uri"`
	Methods           []string       `json:"methods"`
	Parameters        []string       `json:"parameters"`
	ParameterDefaults map[string]any `json:"parameter_defaults"`
}

func (route Route) meta() metaJSON {
	params := route.Params
	if params == nil {
		params = []string{}
	}
	methods := route.Methods
	if methods == nil {
		methods = []string{}
	}
	return metaJSON{URI: route.URI, Methods: methods, Parameters: params, ParameterDefaults: map[string]any{}}
}

// LevelDoc 生成 GET /{prefix}/{level} 的响应体：{"level":"...","routes":{...}}。
// routes 的键顺序沿用注册顺序，便于与 Laravel 实测样本逐字节比对。
func (r *Registry) LevelDoc(name string) (json.RawMessage, bool) {
	for _, level := range r.levels {
		if level.Name != name {
			continue
		}
		buf := bytes.NewBufferString(`{"level":`)
		buf.Write(mustMarshal(name))
		buf.WriteString(`,"routes":{`)
		for i, route := range level.Routes {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.Write(mustMarshal(route.Name))
			buf.WriteByte(':')
			buf.Write(mustMarshal(route.meta()))
		}
		buf.WriteString("}}")
		return json.RawMessage(buf.Bytes()), true
	}
	return nil, false
}

// SummaryDoc 生成 GET /{prefix} 的响应体：schemeVersion + 各层级摘要 + config。
func (r *Registry) SummaryDoc(cacheTTL int) json.RawMessage {
	type levelSummary struct {
		Description string    `json:"description"`
		Load        string    `json:"load"`
		RouteCount  int       `json:"route_count"`
		Route       routeRefs `json:"route"`
	}
	type config struct {
		StrictMode     bool   `json:"strict_mode"`
		EndpointPrefix string `json:"endpoint_prefix"`
		URLPrefix      any    `json:"url_prefix"`
		CacheTTL       int    `json:"cache_ttl"`
	}
	type summary struct {
		SchemeVersion int             `json:"schemeVersion"`
		Levels        json.RawMessage `json:"levels"`
		Config        config          `json:"config"`
	}

	buf := bytes.NewBufferString("{")
	for i, level := range r.levels {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(mustMarshal(level.Name))
		buf.WriteByte(':')
		buf.Write(mustMarshal(levelSummary{
			Description: level.Description,
			Load:        level.Load,
			RouteCount:  len(level.Routes),
			Route:       routeRefs{URI: r.endpointPrefix + "/" + level.Name, Methods: []string{"GET", "HEAD"}},
		}))
	}
	buf.WriteByte('}')

	doc, err := json.Marshal(summary{
		SchemeVersion: SchemeVersion,
		Levels:        json.RawMessage(buf.Bytes()),
		Config:        config{StrictMode: false, EndpointPrefix: r.endpointPrefix, URLPrefix: nil, CacheTTL: cacheTTL},
	})
	if err != nil {
		return json.RawMessage(`{"schemeVersion":1,"levels":{},"config":{}}`)
	}
	return json.RawMessage(doc)
}

// routeRefs 是摘要里指向层级端点的路由引用。
type routeRefs struct {
	URI     string   `json:"uri"`
	Methods []string `json:"methods"`
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`null`)
	}
	return json.RawMessage(b)
}

// BuiltinLevels 是本项目实际使用的两个层级：public 免令牌，admin 需带控制器令牌。
// client/manage/unassigned 保留为空层级，使摘要文档与 Laravel 版逐字段可比。
func BuiltinLevels(public, admin []Route) []Level {
	return []Level{
		{Name: "public", Description: "公共接口（无需登录）", Load: "eager", Routes: public},
		{Name: "client", Description: "客户端用户接口", Load: "lazy", Routes: nil},
		{Name: "manage", Description: "运营管理接口", Load: "lazy", Routes: nil},
		{Name: "admin", Description: "系统管理接口", Load: "lazy", Routes: admin},
		{Name: "unassigned", Description: "未命中任何层级的路由", Load: "lazy", Routes: nil},
	}
}
