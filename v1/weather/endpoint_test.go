package weather_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
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

func TestCurrentReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Current(context.Background(), loc, weather.Options{}); err != nil {
		t.Fatalf("Current: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != weather.EndpointCurrent {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, weather.EndpointCurrent)
	}
	if req.TTL != 10*time.Minute {
		t.Errorf("ttl = %v, want 10m", req.TTL)
	}
}

func TestHourlyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Hourly(context.Background(), loc, 24, weather.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != weather.EndpointHourly {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, weather.EndpointHourly)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

func TestDailyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Daily(context.Background(), loc, 3, weather.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != weather.EndpointDaily {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, weather.EndpointDaily)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
//
// 改了会让已写入的缓存条目全部失效。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Current(context.Background(), loc, weather.Options{Lang: "zh", LocalTime: true}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	if want := "qwx:v1:current:39.92,116.41:lang=zh:localTime=1"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	// 没有可选参数时键里不留后缀。
	if _, err := c.Current(context.Background(), loc, weather.Options{}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	if want := "qwx:v1:current:39.92,116.41"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Hourly(context.Background(), loc, 24, weather.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}
	if want := "qwx:v1:hourly:39.92,116.41:h24"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Daily(context.Background(), loc, 7, weather.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if want := "qwx:v1:daily:39.92,116.41:d7"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}

// TestCacheKeySeparatesAffectingParams 影响结果的参数必须进键。
func TestCacheKeySeparatesAffectingParams(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	keys := map[string]bool{}
	for _, opts := range []weather.Options{{Lang: "zh"}, {Lang: "en"}, {Lang: "zh", LocalTime: true}} {
		if _, err := c.Current(context.Background(), loc, opts); err != nil {
			t.Fatalf("Current: %v", err)
		}
		keys[d.Last(t).CacheKey] = true
	}
	if len(keys) != 3 {
		t.Errorf("keys = %v, want 3 个不同的键", keys)
	}

	hourly := map[string]bool{}
	for _, hours := range []int{24, 48} {
		if _, err := c.Hourly(context.Background(), loc, hours, weather.Options{}); err != nil {
			t.Fatalf("Hourly: %v", err)
		}
		hourly[d.Last(t).CacheKey] = true
	}
	if len(hourly) != 2 {
		t.Errorf("keys = %v, want 2 个不同的键", hourly)
	}
}
