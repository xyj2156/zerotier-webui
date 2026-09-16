package zt

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeController 启动一个假控制器：按原样记录每个请求（含请求体），再把请求体交还给业务 handler。
func fakeController(t *testing.T, handler http.HandlerFunc) (*Client, *[]string) {
	t.Helper()
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(raw))
		seen = append(seen, r.Method+" "+r.URL.Path+" "+string(raw))
		w.Header().Set("Content-Type", "application/json")
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	return &Client{BaseURL: server.URL, Token: "secret", Timeout: 2 * time.Second, HTTP: server.Client()}, &seen
}

func TestStatusPassesAuthHeaderAndPreservesRawJSON(t *testing.T) {
	var gotAuth string
	client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("X-ZT1-Auth")
		_, _ = w.Write([]byte(`{"address":"00fc166136","version":"1.16.1","tune":{"maxConnsPerInterface":0}}`))
	})
	svc := NewService(client, "")

	res := svc.Status(context.Background())
	if !res.Success {
		t.Fatalf("期望成功，实际失败：%s", res.Message)
	}
	if gotAuth != "secret" {
		t.Errorf("X-ZT1-Auth 未透传，实际 %q", gotAuth)
	}
	if !strings.Contains(string(res.Body), `"maxConnsPerInterface":0`) {
		t.Errorf("上游 JSON 未原样透传：%s", res.Body)
	}
}

func TestNetworksProjectsKeyFieldsWithoutExponentiatingTimestamp(t *testing.T) {
	client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/controller/network":
			_, _ = w.Write([]byte(`["8056c240dc8048e1"]`))
		case "/controller/network/8056c240dc8048e1":
			_, _ = w.Write([]byte(`{"id":"8056c240dc8048e1","name":"lab","private":false,"active":true,"creationTime":1758234567890,"revision":7,"enableBroadcast":true}`))
		default:
			t.Errorf("未预期的路径 %s", r.URL.Path)
		}
	})

	res := NewService(client, "").Networks(context.Background())
	if !res.Success {
		t.Fatalf("网络列表失败：%s", res.Message)
	}
	body := string(res.Body)
	// 大整数经 float64 往返会被写成科学计数法，这里必须是原始字面量
	if !strings.Contains(body, "1758234567890") {
		t.Errorf("creationTime 未保真：%s", body)
	}
	if strings.Contains(body, "revision") || strings.Contains(body, "enableBroadcast") {
		t.Errorf("列表项应只保留关键摘要字段：%s", body)
	}

	var list []map[string]any
	if err := json.Unmarshal(res.Body, &list); err != nil {
		t.Fatalf("结果不是数组：%v", err)
	}
	if len(list) != 1 || list[0]["nwid"] != "8056c240dc8048e1" || list[0]["private"] != false {
		t.Fatalf("投影结果不符：%v", list)
	}
}

func TestNetworksKeepsEntryWhenDetailFails(t *testing.T) {
	client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/controller/network" {
			_, _ = w.Write([]byte(`["8056c240dc8048e1"]`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	})

	res := NewService(client, "").Networks(context.Background())
	if !res.Success {
		t.Fatalf("列表整体应成功，实际：%s", res.Message)
	}
	var list []map[string]json.RawMessage
	if err := json.Unmarshal(res.Body, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("应保留一条坏数据，实际 %d 条", len(list))
	}
	if _, ok := list[0]["error"]; !ok {
		t.Errorf("缺少 error 字段：%v", list[0])
	}
	if string(list[0]["private"]) != "null" {
		t.Errorf("private 应为 null，实际 %s", list[0]["private"])
	}
}

func TestNetworksDefaultsMissingFlagsToTrue(t *testing.T) {
	client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/controller/network" {
			_, _ = w.Write([]byte(`["a"]`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"a","private":null}`))
	})

	res := NewService(client, "").Networks(context.Background())
	var list []map[string]any
	if err := json.Unmarshal(res.Body, &list); err != nil {
		t.Fatal(err)
	}
	// 对齐 PHP 的 (bool)($c['private'] ?? true)：null 与缺失都回落 true
	if list[0]["private"] != true || list[0]["active"] != true {
		t.Errorf("缺省回落不符：%v", list[0])
	}
}

func TestNetworksCountOnlyLists(t *testing.T) {
	client, seen := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`["a","b","c"]`))
	})

	res := NewService(client, "").NetworksCount(context.Background())
	if !res.Success || string(res.Body) != "3" {
		t.Fatalf("计数错误：%s / %s", res.Message, res.Body)
	}
	if len(*seen) != 1 {
		t.Errorf("计数不应逐个取详情，实际请求 %v", *seen)
	}
}

func TestMembersNormalizesMapToArrayAndAppendsNodeID(t *testing.T) {
	client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"107b2c4b12":{"authorized":true,"creationTime":1758000000000},"20a4c9f0aa":{"nodeId":"20a4c9f0aa","name":"x"},"30b1cd44ee":{}}`))
	})

	res := NewService(client, "").Members(context.Background(), "8056c240dc8048e1")
	if !res.Success {
		t.Fatalf("成员列表失败：%s", res.Message)
	}
	body := string(res.Body)
	if !strings.HasPrefix(body, "[") {
		t.Fatalf("应规整为数组：%s", body)
	}
	if !strings.Contains(body, `"nodeId":"107b2c4b12"`) {
		t.Errorf("缺失的 nodeId 应由键名补上：%s", body)
	}
	if !strings.Contains(body, `"nodeId":"30b1cd44ee"`) {
		t.Errorf("空对象也应带 nodeId：%s", body)
	}
	if strings.Count(body, `"nodeId"`) != 3 {
		t.Errorf("已有 nodeId 的成员不应重复注入：%s", body)
	}
	if !strings.Contains(body, "1758000000000") {
		t.Errorf("成员数值未保真：%s", body)
	}
}

func TestMembersSkipsNonObjectEntries(t *testing.T) {
	client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`["107b2c4b12","20a4c9f0aa"]`))
	})
	res := NewService(client, "").Members(context.Background(), "nwid")
	if !res.Success || string(res.Body) != "[]" {
		t.Fatalf("旧版地址数组应被跳过，实际 %s / %s", res.Body, res.Message)
	}
}

// TestMembersCompactFormReturnsNativePlaceholdersWithoutHydration 复现真实 bug 的另一半：
// zerotier-one 1.16.x 顶层 member 列表把值写成 revision 整数（{"4a7090b201":6}）。
// 旧实现据此整表丢弃 → 界面「成员为空」。现改为「原生透传、不回源」：紧凑条目产出一行
// 仅带 nodeId 的占位，完整字段交由前端逐行调 networks.members.show 补全。
// 关键点：本请求不得再逐个打 GET /member/<nodeId>，否则成员一多就整体超时。
func TestMembersCompactFormReturnsNativePlaceholdersWithoutHydration(t *testing.T) {
	client, seen := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/controller/network/8056c240dc8048e1/member":
			_, _ = w.Write([]byte(`{"20a4c9f0aa":7,"107b2c4b12":6,"30b1cd44ee":5}`))
		default:
			// 任何 /member/<nodeId> 子路径都说明后端仍在回源，属于本次要消除的行为。
			t.Errorf("不应发生回源补全请求：%s", r.URL.Path)
		}
	})

	res := NewService(client, "").Members(context.Background(), "8056c240dc8048e1")
	if !res.Success {
		t.Fatalf("成员列表失败：%s", res.Message)
	}
	var list []map[string]any
	if err := json.Unmarshal(res.Body, &list); err != nil {
		t.Fatalf("结果不是数组：%s (%v)", res.Body, err)
	}
	if len(list) != 3 {
		t.Fatalf("紧凑形态应产出 3 行占位，实际 %d：%s", len(list), res.Body)
	}
	// 按 nodeId 字典序：107b < 20a4 < 30b1
	for i, want := range []string{"107b2c4b12", "20a4c9f0aa", "30b1cd44ee"} {
		if list[i]["nodeId"] != want {
			t.Errorf("第 %d 行应为 nodeId=%s 的占位，实际 %v", i, want, list[i])
		}
	}
	if _, hasAuth := list[0]["authorized"]; hasAuth {
		t.Errorf("紧凑占位不应含回源字段：%v", list[0])
	}
	if strings.Count(string(res.Body), `"nodeId"`) != 3 {
		t.Errorf("每行应恰好一个 nodeId：%s", res.Body)
	}
	hydrations := 0
	for _, line := range *seen {
		if strings.Contains(line, "/member/") {
			hydrations++
		}
	}
	if hydrations != 0 {
		t.Errorf("紧凑成员不应触发任何回源，实际 %d：%v", hydrations, *seen)
	}
}

func TestCreateNetworkBuildsNwidFromControllerAddressAndMergesDefaults(t *testing.T) {
	var captured []string
	client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			_, _ = w.Write([]byte(`{"address":"00FC166136"}`))
			return
		}
		body, _ := io.ReadAll(r.Body)
		captured = append(captured, r.URL.Path+"|"+string(body))
		_, _ = w.Write([]byte(`{}`))
	})

	payload := json.RawMessage(`{"name":"lab","private":true,"revision":99,"creationTime":1,"ipAssignmentPools":[{"ipRangeStart":"10.0.0.1","ipRangeEnd":"10.0.0.99"}]}`)
	res := NewService(client, "").CreateNetwork(context.Background(), payload)
	if !res.Success {
		t.Fatalf("创建失败：%s", res.Message)
	}
	if len(captured) != 1 {
		t.Fatalf("应只发一次写请求：%v", captured)
	}
	path, body, _ := strings.Cut(captured[0], "|")

	// nwid = 控制器地址(小写 10hex) + 6 位随机 hex
	if !strings.HasPrefix(path, "/controller/network/00fc166136") || len(path) != len("/controller/network/")+16 {
		t.Errorf("nwid 生成不合法：%s", path)
	}
	var sent map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &sent); err != nil {
		t.Fatalf("写请求体不是对象：%s (%v)", body, err)
	}
	if string(sent["id"]) != string(mustJSON(strings.TrimPrefix(path, "/controller/network/"))) {
		t.Errorf("id 应等于 nwid：%s", body)
	}
	for _, key := range []string{"mtu", "multicastTTL", "enableBroadcast", "v4AssignMode", "routes"} {
		if _, ok := sent[key]; !ok {
			t.Errorf("缺省模板字段 %s 未合并进请求体：%s", key, body)
		}
	}
	for _, key := range []string{"revision", "creationTime"} {
		if _, ok := sent[key]; ok {
			t.Errorf("只读字段 %s 应被剥离：%s", key, body)
		}
	}
	if string(sent["name"]) != `"lab"` {
		t.Errorf("透传字段被改动：%s", body)
	}
}

func TestCreateNetworkWithoutAddressFails(t *testing.T) {
	client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	})
	res := NewService(client, "").CreateNetwork(context.Background(), json.RawMessage(`{}`))
	if res.Success || !strings.Contains(res.Message, "无法获取控制器节点地址") {
		t.Fatalf("应因取不到节点地址而失败，实际：%+v", res)
	}
}

func TestCreateNetworkUsesConfiguredAddressWithoutStatusCall(t *testing.T) {
	client, seen := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	res := NewService(client, "AABBCCDDEE").CreateNetwork(context.Background(), json.RawMessage(`{"name":"x"}`))
	if !res.Success {
		t.Fatalf("创建失败：%s", res.Message)
	}
	for _, line := range *seen {
		if strings.Contains(line, " /status ") {
			t.Errorf("显式给出地址时不该再问 /status：%v", *seen)
		}
	}
	if !strings.Contains((*seen)[0], "/controller/network/aabbccdde") {
		t.Errorf("nwid 前缀应来自配置地址：%v", *seen)
	}
}

func TestUpdateNetworkStripsReadonlyAndInjectsIdentity(t *testing.T) {
	client, seen := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	res := NewService(client, "").UpdateNetwork(context.Background(), "8056c240dc8048e1",
		json.RawMessage(`{"name":"lab","stats":{"brd":1},"id":"deadbeef","mtu":2700}`))
	if !res.Success {
		t.Fatalf("更新失败：%s", res.Message)
	}
	body := lastBody(*seen)
	var sent map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &sent); err != nil {
		t.Fatal(err)
	}
	if _, ok := sent["stats"]; ok {
		t.Errorf("stats 应被剥离：%s", body)
	}
	if string(sent["id"]) != `"8056c240dc8048e1"` || string(sent["nwid"]) != `"8056c240dc8048e1"` {
		t.Errorf("id/nwid 应由路径注入：%s", body)
	}
	if _, ok := sent["active"]; ok {
		t.Errorf("更新不该强塞 active 缺省值：%s", body)
	}
}

func TestUpdateMemberStripsReadonlyFields(t *testing.T) {
	client, seen := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	res := NewService(client, "").UpdateMember(context.Background(), "8056c240dc8048e1", "107b2c4b12",
		json.RawMessage(`{"authorized":true,"name":"pc1","online":true,"clientVersion":{"app":"1.16.1"},"latency":6}`))
	if !res.Success {
		t.Fatalf("更新成员失败：%s", res.Message)
	}
	body := lastBody(*seen)
	var sent map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &sent); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"online", "clientVersion", "latency"} {
		if _, ok := sent[key]; ok {
			t.Errorf("只读字段 %s 未剥离：%s", key, body)
		}
	}
	if _, ok := sent["authorized"]; !ok {
		t.Errorf("可写字段丢失：%s", body)
	}
}

func TestDeleteNetworkAndMemberPaths(t *testing.T) {
	client, seen := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(``))
	})
	svc := NewService(client, "")
	if res := svc.DeleteNetwork(context.Background(), "aaa"); !res.Success {
		t.Fatalf("删除网络失败：%s", res.Message)
	}
	if res := svc.DeleteMember(context.Background(), "aaa", "bbb"); !res.Success {
		t.Fatalf("删除成员失败：%s", res.Message)
	}
	if len(*seen) != 2 || !strings.HasPrefix((*seen)[0], "DELETE /controller/network/aaa") || !strings.HasPrefix((*seen)[1], "DELETE /controller/network/aaa/member/bbb") {
		t.Fatalf("删除路径不符：%v", *seen)
	}
}

func TestEmptySuccessBodyNormalizesToTrue(t *testing.T) {
	client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	res := NewService(client, "").DeleteNetwork(context.Background(), "aaa")
	if !res.Success || string(res.Body) != "true" {
		t.Fatalf("空体应归一成 true，实际 %+v", res)
	}
}

func TestMissingTokenFailsLocally(t *testing.T) {
	client, seen := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("无令牌时不应发起上游请求")
	})
	client.Token = ""
	res := NewService(client, "").Status(context.Background())
	if res.Success || !strings.Contains(res.Message, "未配置控制器令牌") {
		t.Fatalf("应本地拒绝，实际 %+v", res)
	}
	if len(*seen) != 0 {
		t.Errorf("不应有上游请求：%v", *seen)
	}
}

func TestErrorMessagePrecedence(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"json message", http.StatusForbidden, `{"message":"invalid auth"}`, "invalid auth"},
		{"plain body", http.StatusForbidden, "forbidden", "forbidden"},
		{"empty body", http.StatusNotFound, "", "HTTP 404"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			res := NewService(client, "").Status(context.Background())
			if res.Success || res.Message != tc.want {
				t.Fatalf("期望失败消息 %q，实际 %+v", tc.want, res)
			}
		})
	}
}

func TestRawGetRejectsUnsafePaths(t *testing.T) {
	client, seen := fakeController(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	svc := NewService(client, "")

	for _, bad := range []string{"", "peer", "//evil.example.com/x", "http://evil.example.com", "/../secret", "/a/../../b"} {
		if res := svc.RawGet(context.Background(), bad); res.Success {
			t.Errorf("path %q 应被拒绝", bad)
		}
	}
	if len(*seen) != 0 {
		t.Errorf("非法 path 不应打到上游：%v", *seen)
	}
	if res := svc.RawGet(context.Background(), "/peer?json=true"); !res.Success {
		t.Fatalf("合法 path 应通过：%s", res.Message)
	}
	if !strings.HasPrefix((*seen)[0], "GET /peer ") {
		t.Errorf("上游路径不符：%v", *seen)
	}
}

func TestSafeRawPath(t *testing.T) {
	if got, err := SafeRawPath(" /controller/network/aaa "); err != nil || got != "/controller/network/aaa" {
		t.Errorf("应去除首尾空白，实际 %q / %v", got, err)
	}
	if got, err := SafeRawPath("/status"); err != nil || got != "/status" {
		t.Errorf("正常路径失败：%q %v", got, err)
	}
}

func TestNormalizeNetworkInputConvenienceFields(t *testing.T) {
	res, err := NormalizeNetworkInput([]byte(`{"type":"public","cidr":"10.5.0.0/16","_token":"x","name":"n1"}`))
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(res, &obj); err != nil {
		t.Fatal(err)
	}
	if string(obj["private"]) != "false" {
		t.Errorf("type=public 应映射 private=false，实际 %s", obj["private"])
	}
	for _, key := range []string{"type", "cidr", "_token"} {
		if _, ok := obj[key]; ok {
			t.Errorf("便捷字段 %s 不应透传：%s", key, res)
		}
	}
	if got := string(obj["ipAssignmentPools"]); got != `[{"ipRangeEnd":"10.5.255.254","ipRangeStart":"10.5.0.1"}]` {
		t.Errorf("分配池展开不符：%s", got)
	}
	if got := string(obj["routes"]); got != `[{"target":"10.5.0.0/16","via":null}]` {
		t.Errorf("路由展开不符：%s", got)
	}
}

func TestNormalizeNetworkInputKeepsExplicitPools(t *testing.T) {
	res, err := NormalizeNetworkInput([]byte(`{"cidr":"10.5.0.0/16","ipAssignmentPools":[{"ipRangeStart":"192.168.9.1","ipRangeEnd":"192.168.9.9"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(res), "192.168.9.1") || strings.Contains(string(res), "10.5.0.1") {
		t.Errorf("显式 pools 存在时不该由 cidr 展开：%s", res)
	}
}

func TestNormalizeNetworkInputValidates(t *testing.T) {
	if _, err := NormalizeNetworkInput([]byte(`{"type":"both"}`)); err == nil {
		t.Error("非法 type 应报错")
	}
	if _, err := NormalizeNetworkInput([]byte(`{"private":"yes"}`)); err == nil {
		t.Error("非布尔 private 应报错")
	}
	if _, err := NormalizeNetworkInput([]byte(`["a"]`)); err == nil {
		t.Error("数组请求体应报错")
	}
	res, err := NormalizeNetworkInput(nil)
	if err != nil || string(res) != "{}" {
		t.Errorf("空请求体应归一成 {}，实际 %s / %v", res, err)
	}
}

func TestExpandCIDREdges(t *testing.T) {
	for _, bad := range []string{"", "10.0.0.1", "10.0.0.1/32", "10.0.0.1/0", "::1/64", "10.0.0.1/abc"} {
		if _, _, ok := expandCIDR(bad); !ok {
			continue
		}
		t.Errorf("非法 cidr 应被拒绝：%q", bad)
	}
	pools, routes, ok := expandCIDR("192.168.1.130/24")
	if !ok {
		t.Fatal("/24 应合法")
	}
	if !strings.Contains(string(pools), `"ipRangeStart":"192.168.1.1"`) || !strings.Contains(string(pools), `"ipRangeEnd":"192.168.1.254"`) {
		t.Errorf("池边界应为网络内首末可用地址：%s", pools)
	}
	if !strings.Contains(string(routes), `"target":"192.168.1.0/24"`) {
		t.Errorf("路由目标应取网络号：%s", routes)
	}
}

// lastBody 取最近一次请求的请求体（记录格式：METHOD path body）。
func lastBody(seen []string) string {
	if len(seen) == 0 {
		return ""
	}
	line := seen[len(seen)-1]
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return ""
	}
	return parts[2]
}
