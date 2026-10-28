package geo_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/geo"
	"github.com/KanoCifer/qwx/internal/testutil"
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

func TestLookupCityReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.LookupCity(context.Background(), loc, geo.Options{}); err != nil {
		t.Fatalf("LookupCity: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != geo.EndpointCity {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, geo.EndpointCity)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

func TestLookupPOIReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.LookupPOI(context.Background(), loc, geo.POIScenic, geo.Options{}); err != nil {
		t.Fatalf("LookupPOI: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != geo.EndpointPOI {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, geo.EndpointPOI)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

func TestPOIRangeReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.POIRange(context.Background(), loc, geo.POIScenic, geo.POIRangeOptions{}); err != nil {
		t.Fatalf("POIRange: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != geo.EndpointPOIRange {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, geo.EndpointPOIRange)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

func TestTopCitiesReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	if _, err := c.TopCities(context.Background(), geo.Options{}); err != nil {
		t.Fatalf("TopCities: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != geo.EndpointTopCities {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, geo.EndpointTopCities)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
//
// 改了会让已写入的缓存条目全部失效。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	idLoc, err := qwx.ParseLocation("101011600")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.LookupCity(context.Background(), idLoc, geo.Options{Lang: "zh"}); err != nil {
		t.Fatalf("LookupCity: %v", err)
	}
	if want := "qwx:geo:city:101011600:lang=zh"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	coordLoc := qwx.Location{Lat: 39.92, Lon: 116.41}
	if _, err := c.LookupPOI(context.Background(), coordLoc, geo.POIScenic, geo.Options{}); err != nil {
		t.Fatalf("LookupPOI: %v", err)
	}
	if want := "qwx:geo:poi:scenic:116.41,39.92"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.POIRange(context.Background(), coordLoc, geo.POIScenic, geo.POIRangeOptions{Lang: "zh", Number: 5, Radius: 20}); err != nil {
		t.Fatalf("POIRange: %v", err)
	}
	if want := "qwx:geo:poirange:scenic:116.41,39.92:lang=zh:n=5:r=20"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	// 半径为 0 时键里保留 :r=0，请求里不发 radius —— 既有键字面量的约定。
	if _, err := c.POIRange(context.Background(), coordLoc, geo.POIScenic, geo.POIRangeOptions{}); err != nil {
		t.Fatalf("POIRange: %v", err)
	}
	if want := "qwx:geo:poirange:scenic:116.41,39.92:r=0"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.TopCities(context.Background(), geo.Options{Lang: "zh", Number: 5}); err != nil {
		t.Fatalf("TopCities: %v", err)
	}
	if want := "qwx:geo:topcity:lang=zh:n=5"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}
