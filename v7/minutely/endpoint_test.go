package minutely_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7/minutely"
)

func TestMinutelyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := minutely.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Minutely(context.Background(), loc, "zh"); err != nil {
		t.Fatalf("Minutely: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != minutely.Endpoint {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, minutely.Endpoint)
	}
	if req.TTL != 5*time.Minute {
		t.Errorf("ttl = %v, want 5m", req.TTL)
	}
	// 缓存键字面量在已发布版本里定死，改了会让旧缓存条目全部失效。
	if want := "qwx:minutely:116.41,39.92:lang=zh"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}
}
