package rediscache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/alicebob/miniredis/v2"
)

// newTestCache 构造一个连着内存 Redis 的缓存。
func newTestCache(t *testing.T) (*Cache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	c, err := New(Options{Addr: mr.Addr()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, mr
}

func TestCacheGetSet(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	data, ok, err := c.Get(ctx, "k")
	if err != nil || !ok || string(data) != "v" {
		t.Errorf("Get = %q, %v, %v", data, ok, err)
	}
}

func TestCacheMiss(t *testing.T) {
	c, _ := newTestCache(t)

	// 未命中不是错误：返回 (nil, false, nil)。
	data, ok, err := c.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("Get on missing key: %v", err)
	}
	if ok {
		t.Error("missing key should not hit")
	}
	if data != nil {
		t.Errorf("data = %q, want nil", data)
	}
}

func TestCacheTTL(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	// 写入时带了过期时间。
	if ttl := mr.TTL("k"); ttl != time.Minute {
		t.Errorf("ttl = %s, want %s", ttl, time.Minute)
	}

	// 时间推进后条目应过期，读成未命中。
	mr.FastForward(2 * time.Minute)
	if _, ok, err := c.Get(ctx, "k"); err != nil || ok {
		t.Errorf("expired key: ok=%v err=%v, want miss", ok, err)
	}
}

func TestCacheSetNonPositiveTTL(t *testing.T) {
	c, mr := newTestCache(t)
	ctx := context.Background()

	for _, ttl := range []time.Duration{0, -time.Second} {
		err := c.Set(ctx, "k", []byte("v"), ttl)
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("Set with ttl=%s: err = %v, want ErrInvalidParameter", ttl, err)
		}
	}
	// 校验没过就不该留下条目。
	if mr.Exists("k") {
		t.Error("invalid ttl should not write to redis")
	}
}

func TestCacheKeyPrefix(t *testing.T) {
	mr := miniredis.RunT(t)
	c, err := New(Options{Addr: mr.Addr(), KeyPrefix: "myapp:"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !mr.Exists("myapp:k") {
		t.Error("key prefix not applied")
	}
	if mr.Exists("k") {
		t.Error("prefixed key should not be written without prefix")
	}

	data, ok, err := c.Get(ctx, "k")
	if err != nil || !ok || string(data) != "v" {
		t.Errorf("Get = %q, %v, %v", data, ok, err)
	}
}

func TestCacheNoPrefix(t *testing.T) {
	c, mr := newTestCache(t)

	if err := c.Set(context.Background(), "k", []byte("v"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !mr.Exists("k") {
		t.Error("key should be written as-is without prefix")
	}
}

func TestCachePing(t *testing.T) {
	c, _ := newTestCache(t)
	if err := c.Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestCachePingUnreachable(t *testing.T) {
	// 端口没人监听，Ping 应当报错而不是 panic。
	c, err := New(Options{Addr: "127.0.0.1:1", DialTimeout: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = c.Close() }()

	if err := c.Ping(context.Background()); err == nil {
		t.Error("Ping to unreachable redis should fail")
	}
}

func TestCacheConcurrent(t *testing.T) {
	c, _ := newTestCache(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
				t.Errorf("Set: %v", err)
				return
			}
			if _, ok, err := c.Get(ctx, "k"); err != nil || !ok {
				t.Errorf("Get: ok=%v err=%v", ok, err)
			}
		}()
	}
	wg.Wait()
}

func TestCacheImplementsQwxCache(t *testing.T) {
	// 编译期断言之外的运行时确认：它就是 SDK 认的缓存。
	var c qwx.Cache = &Cache{}
	_ = c
}
