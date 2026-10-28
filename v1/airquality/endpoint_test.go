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
