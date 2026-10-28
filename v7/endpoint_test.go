package v7_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7"
)

func TestEndpointIdentifiersAreDistinct(t *testing.T) {
	seen := map[qwx.Endpoint]bool{}
	for _, ep := range []qwx.Endpoint{
		v7.EndpointWarnings,
		v7.EndpointTide,
		v7.EndpointIndices,
		v7.EndpointMinutely,
	} {
		if seen[ep] {
			t.Errorf("duplicate endpoint identifier %q", ep)
		}
		seen[ep] = true
	}
}

func TestWarningsReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := v7.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Warnings(context.Background(), loc, "zh"); err != nil {
		t.Fatalf("Warnings: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != v7.EndpointWarnings {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, v7.EndpointWarnings)
	}
	if req.TTL != 5*time.Minute {
		t.Errorf("ttl = %v, want 5m", req.TTL)
	}
}

func TestTideReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := v7.New(d)

	if _, err := c.Tide(context.Background(), "P2352", "20260101"); err != nil {
		t.Fatalf("Tide: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != v7.EndpointTide {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, v7.EndpointTide)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

func TestIndicesReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := v7.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Indices(context.Background(), loc, 0, 3, "zh"); err != nil {
		t.Fatalf("Indices: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != v7.EndpointIndices {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, v7.EndpointIndices)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

func TestMinutelyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := v7.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Minutely(context.Background(), loc, "zh"); err != nil {
		t.Fatalf("Minutely: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != v7.EndpointMinutely {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, v7.EndpointMinutely)
	}
	if req.TTL != 5*time.Minute {
		t.Errorf("ttl = %v, want 5m", req.TTL)
	}
}
