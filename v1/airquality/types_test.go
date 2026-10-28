package airquality_test

import (
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/airquality"
)

func TestEndpointIdentifiersAreDistinct(t *testing.T) {
	seen := map[qwx.Endpoint]bool{}
	for _, ep := range []qwx.Endpoint{
		airquality.EndpointCurrent,
		airquality.EndpointDaily,
		airquality.EndpointHourly,
		airquality.EndpointStation,
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
		{string(airquality.EndpointCurrent), "airquality/v1/current"},
		{string(airquality.EndpointDaily), "airquality/v1/daily"},
		{string(airquality.EndpointHourly), "airquality/v1/hourly"},
		{string(airquality.EndpointStation), "airquality/v1/stations"},
	} {
		if tc.got != tc.want {
			t.Errorf("endpoint = %q, want %q", tc.got, tc.want)
		}
	}
}
