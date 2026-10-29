package historical_test

import (
	"testing"

	"github.com/KanoCifer/qwx/v7/historical"
)

// TestEndpointValuesAreFrozen 端点常量的值在已发布版本里定死。
//
// TTLPolicy 按值匹配，改值会让用户的 TTL 配置静默失效。
func TestEndpointValuesAreFrozen(t *testing.T) {
	if got, want := string(historical.Endpoint), "v7/historical/weather"; got != want {
		t.Errorf("endpoint = %q, want %q", got, want)
	}
}
