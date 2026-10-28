package weather_test

import (
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/weather"
)

// TestEndpointIdentifiersAreDistinct 端点标识是公开契约，同一包里不能重名。
func TestEndpointIdentifiersAreDistinct(t *testing.T) {
	seen := map[qwx.Endpoint]bool{}
	for _, ep := range []qwx.Endpoint{
		weather.EndpointCurrent,
		weather.EndpointHourly,
		weather.EndpointDaily,
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
		{string(weather.EndpointCurrent), "weather/v1/current"},
		{string(weather.EndpointHourly), "weather/v1/hourly"},
		{string(weather.EndpointDaily), "weather/v1/daily"},
	} {
		if tc.got != tc.want {
			t.Errorf("endpoint = %q, want %q", tc.got, tc.want)
		}
	}
}
