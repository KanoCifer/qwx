package indices_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7/indices"
)

func TestIndicesReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := indices.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Indices(context.Background(), loc, 0, 3, "zh"); err != nil {
		t.Fatalf("Indices: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != indices.Endpoint {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, indices.Endpoint)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
	// 缓存键字面量在已发布版本里定死，改了会让旧缓存条目全部失效。
	if want := "qwx:indices:116.41,39.92:t0:d3:lang=zh"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}
}
