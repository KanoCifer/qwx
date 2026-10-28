package qwx_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
)

// stubDoer 是 Doer 的第三方实现：端点子包只依赖这个接口，
// 不依赖 *qwx.Client 的具体实现。
type stubDoer struct {
	got qwx.Request
}

func (d *stubDoer) Do(_ context.Context, req qwx.Request, _ any) error {
	d.got = req
	return nil
}

func TestClientImplementsDoer(t *testing.T) {
	// 基础客户端必须能当 Doer 用：端点子包只认这个接口。
	var _ qwx.Doer = (*qwx.Client)(nil)
}

func TestDoerSeamReceivesRequest(t *testing.T) {
	// Doer 是端点子包与基础客户端之间唯一的接缝：
	// 替身实现收到的必须是调用方构造的 Request 原样。
	d := &stubDoer{}
	req := qwx.Request{
		Method:   "GET",
		Path:     "/weather/v1/current/39.92/116.41",
		Endpoint: "weather/v1/current",
		CacheKey: "qwx:v1:current:39.92,116.41",
		TTL:      10 * time.Minute,
	}
	if err := d.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if d.got.Path != req.Path {
		t.Errorf("path = %q, want %q", d.got.Path, req.Path)
	}
	if d.got.Endpoint != req.Endpoint {
		t.Errorf("endpoint = %q, want %q", d.got.Endpoint, req.Endpoint)
	}
	if d.got.CacheKey != req.CacheKey {
		t.Errorf("cacheKey = %q, want %q", d.got.CacheKey, req.CacheKey)
	}
}
