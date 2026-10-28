package qwx

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	c := NewMemoryCache()
	ctx := context.Background()

	if _, ok, err := c.Get(ctx, "missing"); err != nil || ok {
		t.Errorf("missing key: ok=%v err=%v", ok, err)
	}

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	data, ok, err := c.Get(ctx, "k")
	if err != nil || !ok || string(data) != "v" {
		t.Errorf("Get = %q, %v, %v", data, ok, err)
	}

	// 过期后应当 miss 并清理条目。
	c.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if _, ok, _ := c.Get(ctx, "k"); ok {
		t.Error("expired key should miss")
	}
	if len(c.entries) != 0 {
		t.Errorf("entries = %d, want 0 after expiry", len(c.entries))
	}
}
