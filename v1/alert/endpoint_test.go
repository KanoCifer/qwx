package alert_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v1/alert"
)

func TestCurrentReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := alert.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Current(context.Background(), loc, alert.Options{}); err != nil {
		t.Fatalf("Current: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != alert.Endpoint {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, alert.Endpoint)
	}
	if req.TTL != 5*time.Minute {
		t.Errorf("ttl = %v, want 5m", req.TTL)
	}
}
