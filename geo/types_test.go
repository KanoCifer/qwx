package geo_test

import (
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/geo"
)

func TestEndpointIdentifiersAreDistinct(t *testing.T) {
	seen := map[qwx.Endpoint]bool{}
	for _, ep := range []qwx.Endpoint{
		geo.EndpointCity,
		geo.EndpointPOI,
		geo.EndpointPOIRange,
		geo.EndpointTopCities,
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
		{string(geo.EndpointCity), "geo/v2/city/lookup"},
		{string(geo.EndpointPOI), "geo/v2/poi/lookup"},
		{string(geo.EndpointPOIRange), "geo/v2/poi/range"},
		{string(geo.EndpointTopCities), "geo/v2/city/top"},
	} {
		if tc.got != tc.want {
			t.Errorf("endpoint = %q, want %q", tc.got, tc.want)
		}
	}
}
