package qwx_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
)

func TestDoDecodesResponse(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200","locations":[]}`)
	c := testutil.NewClient(t, fs)

	var out struct {
		Code      string `json:"code"`
		Locations []any  `json:"locations"`
	}
	err := c.Do(context.Background(), qwx.Request{Path: "/geo/v2/city/lookup"}, &out)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if out.Code != "200" {
		t.Errorf("code = %q, want 200", out.Code)
	}
	fs.AssertPath(t, "/geo/v2/city/lookup")
}

func TestDoCachesResponse(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	c := testutil.NewClient(t, fs, qwx.WithCache(qwx.NewMemoryCache()))
	req := qwx.Request{Path: "/geo/v2/city/top", CacheKey: "qwx:geo:topcity", TTL: time.Minute}

	var out struct {
		Code string `json:"code"`
	}
	for i := 0; i < 2; i++ {
		if err := c.Do(context.Background(), req, &out); err != nil {
			t.Fatalf("Do: %v", err)
		}
	}
	if fs.Hits != 1 {
		t.Errorf("hits = %d, want 1 (second call must hit cache)", fs.Hits)
	}
	// 命中缓存时也要能把内容解析出来。
	if out.Code != "200" {
		t.Errorf("code = %q, want 200", out.Code)
	}
}

func TestDoCacheKeyIsolatesResults(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	c := testutil.NewClient(t, fs, qwx.WithCache(qwx.NewMemoryCache()))

	// 键不同是两份数据，不能互相命中。
	if err := c.Do(context.Background(), qwx.Request{Path: "/x", CacheKey: "k1", TTL: time.Minute}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if err := c.Do(context.Background(), qwx.Request{Path: "/x", CacheKey: "k2", TTL: time.Minute}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if fs.Hits != 2 {
		t.Errorf("hits = %d, want 2", fs.Hits)
	}
}

func TestDoCacheFailureDegradesGracefully(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	c := testutil.NewClient(t, fs, qwx.WithCache(failingCache{}))

	// 缓存故障不应让请求失败。
	if err := c.Do(context.Background(), qwx.Request{Path: "/x", CacheKey: "k", TTL: time.Minute}, nil); err != nil {
		t.Fatalf("Do with broken cache: %v", err)
	}
	if fs.Hits != 1 {
		t.Errorf("hits = %d, want 1", fs.Hits)
	}
}

func TestDoSurfacesUpstreamError(t *testing.T) {
	// Do 要把上游的错误原样包成 *UpstreamError，并按状态码归类。
	// 逐个状态码走完整条链路：HTTP 响应 → UpstreamError → sentinel。
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"bad request", 400, `{"error":{"status":400,"title":"Invalid Parameters","invalidParams":["location"]}}`, qwx.ErrUpstreamRejected},
		{"unauthorized", 401, `{"error":{"status":401,"title":"Unauthorized"}}`, qwx.ErrUnauthorized},
		{"forbidden", 403, `{"error":{"status":403,"title":"Forbidden"}}`, qwx.ErrUnauthorized},
		{"quota", 429, `{"error":{"status":429,"title":"Too Many Requests"}}`, qwx.ErrQuotaExceeded},
		{"server error", 500, `{"error":{"status":500,"title":"Internal Server Error"}}`, qwx.ErrUpstreamUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := testutil.NewFakeServer(t, tc.body)
			fs.Handler = func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}
			c := testutil.NewClient(t, fs)

			err := c.Do(context.Background(), qwx.Request{Path: "/x"}, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("error %v does not match %v", err, tc.want)
			}

			var ue *qwx.UpstreamError
			if !errors.As(err, &ue) {
				t.Fatalf("error is not *UpstreamError: %T", err)
			}
			if ue.StatusCode != tc.status {
				t.Errorf("status = %d, want %d", ue.StatusCode, tc.status)
			}
			if ue.Title == "" {
				t.Error("title should be parsed from upstream body")
			}
		})
	}
}

func TestDoSurfacesUpstreamInvalidParams(t *testing.T) {
	// 新一代会指出非法参数名，调用方靠它定位是经纬倒置还是别的。
	body := `{"error":{"status":400,"title":"Invalid Parameters","invalidParams":["longitude","latitude"]}}`
	fs := testutil.NewFakeServer(t, body)
	fs.Handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(body))
	}
	c := testutil.NewClient(t, fs)

	err := c.Do(context.Background(), qwx.Request{Path: "/x"}, nil)
	var ue *qwx.UpstreamError
	if !errors.As(err, &ue) {
		t.Fatalf("error is not *UpstreamError: %T", err)
	}
	if len(ue.InvalidParams) != 2 {
		t.Errorf("invalidParams = %v, want 2 entries", ue.InvalidParams)
	}
}

func TestDoRejectsInvalidJSON(t *testing.T) {
	fs := testutil.NewFakeServer(t, `not json`)
	c := testutil.NewClient(t, fs)

	var out map[string]any
	err := c.Do(context.Background(), qwx.Request{Path: "/x"}, &out)
	if !errors.Is(err, qwx.ErrInvalidResponse) {
		t.Errorf("error = %v, want ErrInvalidResponse", err)
	}
}

func TestDoReturnsErrorWhenContextCanceledDuringRetry(t *testing.T) {
	// 上游持续 5xx：SDK 进入重试退避。此时调用方取消 ctx，SDK 要立刻返回
	// 错误，不能继续等退避，也不能返回空响应让调用方 panic。
	body := `{"error":{"status":500,"title":"Internal Server Error"}}`
	fs := testutil.NewFakeServer(t, body)
	fs.Handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(body))
	}
	c := testutil.NewClient(t, fs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(200*time.Millisecond, cancel)

	err := c.Do(ctx, qwx.Request{Path: "/x"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestDoKeepsTransportErrorWhenContextCanceled(t *testing.T) {
	// ctx 已取消时不能只返回 context.Canceled：传输错误里的 URL 带着请求
	// 路径，丢了就没法定位是哪个请求断的，日志里也就只剩一句"context
	// canceled"。
	auth, err := qwx.NewAPIKeyAuth("k")
	if err != nil {
		t.Fatalf("NewAPIKeyAuth: %v", err)
	}
	c, err := qwx.NewClient(auth, "127.0.0.1:1")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = c.Do(ctx, qwx.Request{Path: "/geo/v2/city/lookup"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	var ue *url.Error
	if !errors.As(err, &ue) {
		t.Fatalf("transport error lost: %T", err)
	}
	if !strings.Contains(ue.URL, "/geo/v2/city/lookup") {
		t.Errorf("url = %q, want the request path", ue.URL)
	}
}

func TestNewClientRequiresAuth(t *testing.T) {
	if _, err := qwx.NewClient(nil, "abcxyz.qweatherapi.com"); err == nil {
		t.Error("expected error")
	} else if !errors.Is(err, qwx.ErrInvalidConfig) {
		t.Errorf("error = %v, want ErrInvalidConfig", err)
	}
}

// failingCache 是一个总是失败的缓存实现。
type failingCache struct{}

func (failingCache) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, errCacheBroken
}

func (failingCache) Set(context.Context, string, []byte, time.Duration) error {
	return errCacheBroken
}

var errCacheBroken = errors.New("cache broken")
