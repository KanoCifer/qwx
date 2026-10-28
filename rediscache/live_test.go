//go:build live

// 本文件是打真实 Redis 的冒烟测试，默认不参与 `go test ./...`。
//
// 用法：
//
//	REDIS_ADDR=127.0.0.1:6379 go test -tags live -v -run TestLive ./...
package rediscache

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLive(t *testing.T) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		t.Skip("未设置 REDIS_ADDR，跳过")
	}

	c, err := New(Options{
		Addr:      addr,
		Password:  os.Getenv("REDIS_PASSWORD"),
		KeyPrefix: "qwx:live:",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	ctx := context.Background()
	if err := c.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	key := "smoke"
	if err := c.Set(ctx, key, []byte("hello"), 10*time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}
	data, ok, err := c.Get(ctx, key)
	if err != nil || !ok || string(data) != "hello" {
		t.Fatalf("Get = %q, %v, %v", data, ok, err)
	}
	t.Logf("写入并读回：%s", data)
}
