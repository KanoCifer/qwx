package console_test

import (
	"encoding/json"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/console"
)

func TestEndpointIdentifiersAreDistinct(t *testing.T) {
	seen := map[qwx.Endpoint]bool{}
	for _, ep := range []qwx.Endpoint{
		console.EndpointFinance,
		console.EndpointStats,
	} {
		if seen[ep] {
			t.Errorf("duplicate endpoint identifier %q", ep)
		}
		seen[ep] = true
	}
}

// TestEndpointValuesAreFrozen 端点常量的值在已发布版本里定死。
//
// TTLPolicy 按值匹配，改值会让用户的 TTL 配置静默失效。
func TestEndpointValuesAreFrozen(t *testing.T) {
	for _, tc := range []struct {
		got  string
		want string
	}{
		{string(console.EndpointFinance), "finance/v1/summary"},
		{string(console.EndpointStats), "metrics/v1/stats"},
	} {
		if tc.got != tc.want {
			t.Errorf("endpoint = %q, want %q", tc.got, tc.want)
		}
	}
}

// TestResourcePlanRequestsAcceptsBothForms 资源包的总请求量两种形态都要能解析。
//
// 上游的返回样例给的是字符串（"1000000"），官方字段表标的是数字 ——
// 按其中一种写死，另一种来的时候整个响应都解不出来。
func TestResourcePlanRequestsAcceptsBothForms(t *testing.T) {
	for _, body := range []string{`{"requests":"1000000"}`, `{"requests":1000000}`} {
		var p console.ResourcePlan
		if err := json.Unmarshal([]byte(body), &p); err != nil {
			t.Fatalf("unmarshal %s: %v", body, err)
		}
		got, err := p.Requests.Int64()
		if err != nil {
			t.Fatalf("requests %v: %v", p.Requests, err)
		}
		if got != 1000000 {
			t.Errorf("requests = %d, want 1000000", got)
		}
	}
}

// TestTimeFieldsPassThrough 时间字符串原样透传。
//
// 控制台 API 给的是 UTC（如 "2024-04-03T17:13Z"），本库不改写时区。
func TestTimeFieldsPassThrough(t *testing.T) {
	var r console.FinanceResult
	if err := json.Unmarshal([]byte(`{"asOf":"2024-04-03T17:13Z"}`), &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.AsOf != "2024-04-03T17:13Z" {
		t.Errorf("asOf = %q, want 2024-04-03T17:13Z", r.AsOf)
	}
}

// TestHoursLengthNotAssumed 上游可能给少于 24 条。
func TestHoursLengthNotAssumed(t *testing.T) {
	var c console.APICount
	if err := json.Unmarshal([]byte(`{"api":"Weather","hours":[1,2,3]}`), &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.API != "Weather" {
		t.Errorf("api = %q, want Weather", c.API)
	}
	if len(c.Hours) != 3 {
		t.Errorf("hours = %d, want 3", len(c.Hours))
	}
}
