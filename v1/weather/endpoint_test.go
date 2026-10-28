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
