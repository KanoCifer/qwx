package qwx_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
)

func TestTTLMapUsesOverrideThenFallback(t *testing.T) {
	m := qwx.TTLMap{
		"weather/v1/current": 30 * time.Second,
	}
	if got := m.TTL("weather/v1/current", time.Hour); got != 30*time.Second {
		t.Errorf("TTL(override) = %v, want 30s", got)
	}
	if got := m.TTL("weather/v1/hourly", time.Hour); got != time.Hour {
		t.Errorf("TTL(fallback) = %v, want 1h", got)
	}
}

func TestTTLMapZeroDisablesCache(t *testing.T) {
	m := qwx.TTLMap{"weather/v1/current": 0}
	if got := m.TTL("weather/v1/current", 10*time.Minute); got != 0 {
		t.Errorf("TTL(disabled) = %v, want 0", got)
	}
}

func TestTTLPolicyFunc(t *testing.T) {
	f := qwx.TTLPolicyFunc(func(ep qwx.Endpoint, fallback time.Duration) time.Duration {
		if ep == "weather/v1/current" {
			return time.Second
		}
		return fallback
	})
	if got := f.TTL("weather/v1/current", time.Hour); got != time.Second {
		t.Errorf("TTL = %v, want 1s", got)
	}
	if got := f.TTL("weather/v1/daily", time.Hour); got != time.Hour {
		t.Errorf("TTL = %v, want 1h", got)
	}
}

func TestPolicyOverridesEndpointDefault(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	cache := testutil.NewRecordingCache()
	c := testutil.NewClient(t, fs, qwx.WithCache(cache),
		qwx.WithTTLPolicy(qwx.TTLMap{"weather/v1/current": 30 * time.Second}))

	req := qwx.Request{
		Path:     "/weather/v1/current/39.92/116.41",
		CacheKey: "k",
		Endpoint: "weather/v1/current",
		TTL:      10 * time.Minute,
	}
	if err := c.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	got, ok := cache.TTL("k")
	if !ok {
		t.Fatal("cache key not written")
	}
	if got != 30*time.Second {
		t.Errorf("ttl = %v, want 30s", got)
	}
}

func TestPolicyDoesNotAffectUnlistedEndpoint(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	cache := testutil.NewRecordingCache()
	c := testutil.NewClient(t, fs, qwx.WithCache(cache),
		qwx.WithTTLPolicy(qwx.TTLMap{"weather/v1/current": 30 * time.Second}))

	req := qwx.Request{
		Path:     "/weather/v1/hourly/39.92/116.41",
		CacheKey: "k",
		Endpoint: "weather/v1/hourly",
		TTL:      time.Hour,
	}
	if err := c.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	got, ok := cache.TTL("k")
	if !ok {
		t.Fatal("cache key not written")
	}
	if got != time.Hour {
		t.Errorf("ttl = %v, want 1h", got)
	}
}

func TestPolicyZeroDisablesCacheEntirely(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	cache := testutil.NewRecordingCache()
	c := testutil.NewClient(t, fs, qwx.WithCache(cache),
		qwx.WithTTLPolicy(qwx.TTLMap{"weather/v1/current": 0}))

	req := qwx.Request{
		Path:     "/weather/v1/current/39.92/116.41",
		CacheKey: "k",
		Endpoint: "weather/v1/current",
		TTL:      10 * time.Minute,
	}
	if err := c.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if _, ok := cache.TTL("k"); ok {
		t.Error("cache should not be written when policy disables it")
	}
}

func TestContextOverrideBeatsPolicy(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	cache := testutil.NewRecordingCache()
	c := testutil.NewClient(t, fs, qwx.WithCache(cache),
		qwx.WithTTLPolicy(qwx.TTLMap{"weather/v1/current": 30 * time.Second}))

	ctx := qwx.WithTTLOverride(context.Background(), 5*time.Second)
	req := qwx.Request{
		Path:     "/weather/v1/current/39.92/116.41",
		CacheKey: "k",
		Endpoint: "weather/v1/current",
		TTL:      10 * time.Minute,
	}
	if err := c.Do(ctx, req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	got, ok := cache.TTL("k")
	if !ok {
		t.Fatal("cache key not written")
	}
	if got != 5*time.Second {
		t.Errorf("ttl = %v, want 5s", got)
	}
}

func TestContextOverrideZeroDisablesCache(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	cache := testutil.NewRecordingCache()
	c := testutil.NewClient(t, fs, qwx.WithCache(cache))

	ctx := qwx.WithTTLOverride(context.Background(), 0)
	req := qwx.Request{
		Path:     "/weather/v1/current/39.92/116.41",
		CacheKey: "k",
		Endpoint: "weather/v1/current",
		TTL:      10 * time.Minute,
	}
	if err := c.Do(ctx, req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if _, ok := cache.TTL("k"); ok {
		t.Error("cache should not be written when override disables it")
	}
}

func TestNoPolicyKeepsEndpointDefault(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	cache := testutil.NewRecordingCache()
	c := testutil.NewClient(t, fs, qwx.WithCache(cache))

	req := qwx.Request{
		Path:     "/weather/v1/current/39.92/116.41",
		CacheKey: "k",
		Endpoint: "weather/v1/current",
		TTL:      10 * time.Minute,
	}
	if err := c.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	got, ok := cache.TTL("k")
	if !ok {
		t.Fatal("cache key not written")
	}
	if got != 10*time.Minute {
		t.Errorf("ttl = %v, want 10m", got)
	}
}

func TestPolicyReceivesEndpointDefault(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	var seen qwx.Endpoint
	var fallback time.Duration
	c := testutil.NewClient(t, fs, qwx.WithTTLPolicy(qwx.TTLPolicyFunc(
		func(ep qwx.Endpoint, def time.Duration) time.Duration {
			seen = ep
			fallback = def
			return def
		})))

	req := qwx.Request{
		Path:     "/weather/v1/current/39.92/116.41",
		CacheKey: "k",
		Endpoint: "weather/v1/current",
		TTL:      10 * time.Minute,
	}
	if err := c.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if seen != "weather/v1/current" {
		t.Errorf("endpoint = %q, want weather/v1/current", seen)
	}
	if fallback != 10*time.Minute {
		t.Errorf("fallback = %v, want 10m", fallback)
	}
}

func TestNilContextDoesNotPanic(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	c := testutil.NewClient(t, fs,
		qwx.WithCache(testutil.NewRecordingCache()),
		qwx.WithTTLPolicy(qwx.TTLMap{"weather/v1/current": time.Minute}))

	req := qwx.Request{
		Path:     "/weather/v1/current/39.92/116.41",
		CacheKey: "k",
		Endpoint: "weather/v1/current",
		TTL:      10 * time.Minute,
	}
	// 故意传 nil context：应返回错误，不能 panic。
	if err := c.Do(nil, req, nil); err == nil {
		t.Fatal("expected error for nil context")
	}
}

func TestEmptyEndpointSkipsPolicy(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	cache := testutil.NewRecordingCache()
	c := testutil.NewClient(t, fs, qwx.WithCache(cache),
		qwx.WithTTLPolicy(qwx.TTLPolicyFunc(func(qwx.Endpoint, time.Duration) time.Duration {
			return 0 // 对所有端点都关闭缓存
		})))

	req := qwx.Request{
		Path:     "/weather/v1/current/39.92/116.41",
		CacheKey: "k",
		TTL:      10 * time.Minute,
	}
	if err := c.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	got, ok := cache.TTL("k")
	if !ok {
		t.Fatal("cache key not written")
	}
	if got != 10*time.Minute {
		t.Errorf("ttl = %v, want 10m", got)
	}
}
