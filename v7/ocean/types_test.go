package ocean_test

import (
	"testing"

	"github.com/KanoCifer/qwx/v7/ocean"
)

// TestEndpointValuesAreFrozen 端点常量的值在已发布版本里定死。
//
// TTLPolicy 按值匹配，改值会让用户的 TTL 配置静默失效。
func TestEndpointValuesAreFrozen(t *testing.T) {
	for _, tc := range []struct {
		got  string
		want string
	}{
		{string(ocean.Endpoint), "v7/ocean/tide"},
	} {
		if tc.got != tc.want {
			t.Errorf("endpoint = %q, want %q", tc.got, tc.want)
		}
	}
}
