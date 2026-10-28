package airquality_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
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

func TestCurrentReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Current(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Current: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != airquality.EndpointCurrent {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, airquality.EndpointCurrent)
	}
	if req.TTL != 30*time.Minute {
		t.Errorf("ttl = %v, want 30m", req.TTL)
	}
}

func TestDailyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Daily(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != airquality.EndpointDaily {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, airquality.EndpointDaily)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

func TestHourlyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Hourly(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != airquality.EndpointHourly {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, airquality.EndpointHourly)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

func TestStationReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)

	if _, err := c.Station(context.Background(), "P51762", airquality.Options{}); err != nil {
		t.Fatalf("Station: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != airquality.EndpointStation {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, airquality.EndpointStation)
	}
	if req.TTL != 30*time.Minute {
		t.Errorf("ttl = %v, want 30m", req.TTL)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
//
// 改了会让已写入的缓存条目全部失效。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Current(context.Background(), loc, airquality.Options{Lang: "zh"}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	if want := "qwx:aq:v1:current:39.92,116.41:lang=zh"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Daily(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if want := "qwx:aq:v1:daily:39.92,116.41"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Hourly(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}
	if want := "qwx:aq:v1:hourly:39.92,116.41"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Station(context.Background(), "P51762", airquality.Options{Lang: "zh"}); err != nil {
		t.Fatalf("Station: %v", err)
	}
	if want := "qwx:aq:v1:station:P51762:lang=zh"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}
