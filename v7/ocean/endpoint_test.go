package ocean_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7/ocean"
)

func TestTideReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := ocean.New(d)

	if _, err := c.Tide(context.Background(), "P2352", "20260101"); err != nil {
		t.Fatalf("Tide: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != ocean.Endpoint {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, ocean.Endpoint)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
	// 缓存键字面量在已发布版本里定死，改了会让旧缓存条目全部失效。
	if want := "qwx:tide:P2352:20260101"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}
}
