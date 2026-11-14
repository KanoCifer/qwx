package qwx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fakeServer 是记录请求的假上游。
//
// 根包的测试不共用 internal/testutil：那个包要 import 根包，根包的测试
// 再 import 它就成环。
type fakeServer struct {
	*httptest.Server
	lastPath  string
	lastQuery url.Values
	hits      int
	handler   func(w http.ResponseWriter, r *http.Request)
}

// newFakeServer 构造一个返回固定 JSON 的假上游。
func newFakeServer(t *testing.T, body string) *fakeServer {
	t.Helper()
	fs := &fakeServer{}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.hits++
		fs.lastPath = r.URL.Path
		fs.lastQuery = r.URL.Query()
		if fs.handler != nil {
			fs.handler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(fs.Close)
	return fs
}

// client 构造一个指向假上游的基础客户端。
func (fs *fakeServer) client(t *testing.T, opts ...Option) *Client {
	t.Helper()
	auth, err := NewAPIKeyAuth("test-key")
	if err != nil {
		t.Fatalf("NewAPIKeyAuth: %v", err)
	}
	c, err := NewClient(auth, fs.URL, opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// assertPath 断言最后收到的请求路径。
func (fs *fakeServer) assertPath(t *testing.T, want string) {
	t.Helper()
	if fs.lastPath != want {
		t.Errorf("path = %q, want %q", fs.lastPath, want)
	}
}

func TestDoDecodesResponse(t *testing.T) {
	fs := newFakeServer(t, `{"code":"200","locations":[]}`)
	c := fs.client(t)

	var out map[string]any
	err := c.Do(context.Background(), Request{Path: "/geo/v2/city/lookup"}, &out)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if out["code"] != "200" {
		t.Errorf("code = %v, want 200", out["code"])
	}
	fs.assertPath(t, "/geo/v2/city/lookup")
}

func TestDoCachesResponse(t *testing.T) {
	fs := newFakeServer(t, `{"code":"200"}`)
	c := fs.client(t, WithCache(NewMemoryCache()))
	req := Request{Path: "/geo/v2/city/top", CacheKey: "qwx:geo:topcity", TTL: time.Minute}

	var out map[string]any
	for i := 0; i < 2; i++ {
		if err := c.Do(context.Background(), req, &out); err != nil {
			t.Fatalf("Do: %v", err)
		}
	}
	if fs.hits != 1 {
		t.Errorf("hits = %d, want 1 (second call must hit cache)", fs.hits)
	}
	// 命中缓存时也要能把内容解析出来。
	if out["code"] != "200" {
		t.Errorf("code = %v, want 200", out["code"])
	}
}

func TestDoCacheKeyIsolatesResults(t *testing.T) {
	fs := newFakeServer(t, `{"code":"200"}`)
	c := fs.client(t, WithCache(NewMemoryCache()))

	// 键不同是两份数据，不能互相命中。
	if err := c.Do(context.Background(), Request{Path: "/x", CacheKey: "k1", TTL: time.Minute}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if err := c.Do(context.Background(), Request{Path: "/x", CacheKey: "k2", TTL: time.Minute}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if fs.hits != 2 {
		t.Errorf("hits = %d, want 2", fs.hits)
	}
}

func TestDoCacheFailureDegradesGracefully(t *testing.T) {
	fs := newFakeServer(t, `{"code":"200"}`)
	c := fs.client(t, WithCache(failingCache{}))

	// 缓存故障不应让请求失败。
	if err := c.Do(context.Background(), Request{Path: "/x", CacheKey: "k", TTL: time.Minute}, nil); err != nil {
		t.Fatalf("Do with broken cache: %v", err)
	}
	if fs.hits != 1 {
		t.Errorf("hits = %d, want 1", fs.hits)
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
		{"bad request", 400, `{"error":{"status":400,"title":"Invalid Parameters","invalidParams":["location"]}}`, ErrUpstreamRejected},
		{"unauthorized", 401, `{"error":{"status":401,"title":"Unauthorized"}}`, ErrUnauthorized},
		{"forbidden", 403, `{"error":{"status":403,"title":"Forbidden"}}`, ErrUnauthorized},
		{"quota", 429, `{"error":{"status":429,"title":"Too Many Requests"}}`, ErrQuotaExceeded},
		{"server error", 500, `{"error":{"status":500,"title":"Internal Server Error"}}`, ErrUpstreamUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := newFakeServer(t, tc.body)
			fs.handler = func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}
			c := fs.client(t)

			err := c.Do(context.Background(), Request{Path: "/x"}, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("error %v does not match %v", err, tc.want)
			}

			var ue *UpstreamError
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
	fs := newFakeServer(t, body)
	fs.handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(body))
	}
	c := fs.client(t)

	err := c.Do(context.Background(), Request{Path: "/x"}, nil)
	var ue *UpstreamError
	if !errors.As(err, &ue) {
		t.Fatalf("error is not *UpstreamError: %T", err)
	}
	if len(ue.InvalidParams) != 2 {
		t.Errorf("invalidParams = %v, want 2 entries", ue.InvalidParams)
	}
}

func TestDoRejectsInvalidJSON(t *testing.T) {
	fs := newFakeServer(t, "not json")
	c := fs.client(t)

	var out map[string]any
	err := c.Do(context.Background(), Request{Path: "/x"}, &out)
	if !errors.Is(err, ErrInvalidResponse) {
		t.Errorf("error = %v, want ErrInvalidResponse", err)
	}
}

func TestDoReturnsErrorWhenContextCanceledDuringRetry(t *testing.T) {
	// 上游持续 5xx：SDK 进入重试退避。此时调用方取消 ctx，SDK 要立刻返回
	// 错误，不能继续等退避，也不能返回空响应让调用方 panic。
	body := `{"error":{"status":500,"title":"Internal Server Error"}}`
	fs := newFakeServer(t, body)
	fs.handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(body))
	}
	c := fs.client(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(200*time.Millisecond, cancel)

	err := c.Do(ctx, Request{Path: "/x"}, nil)
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
	auth, err := NewAPIKeyAuth("k")
	if err != nil {
		t.Fatalf("NewAPIKeyAuth: %v", err)
	}
	c, err := NewClient(auth, "127.0.0.1:1")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = c.Do(ctx, Request{Path: "/geo/v2/city/lookup"}, nil)
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
	if _, err := NewClient(nil, "abcxyz.qweatherapi.com"); err == nil {
		t.Error("expected error")
	} else if !errors.Is(err, ErrInvalidConfig) {
		t.Errorf("error = %v, want ErrInvalidConfig", err)
	}
}

func TestNewClientRequiresAPIHost(t *testing.T) {
	for _, host := range []string{"", "   "} {
		_, err := NewClient(&APIKeyAuth{key: "k"}, host)
		if err == nil {
			t.Fatalf("NewClient(host=%q) should fail", host)
		}
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("error = %v, want ErrInvalidConfig", err)
		}
	}
}

func TestNewClientNormalizesAPIHost(t *testing.T) {
	cases := []struct {
		name string
		host string
		want string
	}{
		{"host only", "abcxyz.qweatherapi.com", "https://abcxyz.qweatherapi.com"},
		{"with scheme", "https://abcxyz.qweatherapi.com", "https://abcxyz.qweatherapi.com"},
		{"trailing slash", "https://abcxyz.qweatherapi.com/", "https://abcxyz.qweatherapi.com"},
		{"trailing slashes", "abcxyz.qweatherapi.com///", "https://abcxyz.qweatherapi.com"},
		{"surrounding spaces", "  abcxyz.qweatherapi.com  ", "https://abcxyz.qweatherapi.com"},
		{"plain http kept", "http://127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"path kept", "https://gateway.example.com/qwx", "https://gateway.example.com/qwx"},
		// 尾随问号必须去掉：留着会让整个请求路径被吞进查询串。
		{"trailing question mark", "abcxyz.qweatherapi.com?", "https://abcxyz.qweatherapi.com"},
		{"query dropped", "abcxyz.qweatherapi.com?x=1", "https://abcxyz.qweatherapi.com"},
		{"fragment dropped", "abcxyz.qweatherapi.com#frag", "https://abcxyz.qweatherapi.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewClient(&APIKeyAuth{key: "k"}, tc.host)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if c.apiHost != tc.want {
				t.Errorf("apiHost = %q, want %q", c.apiHost, tc.want)
			}
		})
	}
}

func TestNewClientRejectsMalformedAPIHost(t *testing.T) {
	for _, host := range []string{"https://", "://abc", "https://a b.com"} {
		if _, err := NewClient(&APIKeyAuth{key: "k"}, host); err == nil {
			t.Errorf("NewClient(host=%q) should fail", host)
		} else if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("error = %v, want ErrInvalidConfig", err)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{"seconds", "120", 2 * time.Minute, true},
		{"http date", now.Add(90 * time.Second).Format(http.TimeFormat), 90 * time.Second, true},
		{"past date", now.Add(-time.Minute).Format(http.TimeFormat), 0, true},
		{"empty", "", 0, false},
		{"garbage", "later", 0, false},
		{"negative", "-5", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseRetryAfter(tc.value, now)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("delay = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNextRetryDelay(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name       string
		attempt    int
		retryAfter string
		want       time.Duration
	}{
		{"backoff", 1, "", time.Second},
		{"backoff grows", 3, "", 3 * time.Second},
		{"retry-after raises the floor", 1, "5", 5 * time.Second},
		{"retry-after does not shorten the backoff", 3, "1", 3 * time.Second},
		{"retry-after is capped", 1, "3600", maxRetryDelay},
		{"invalid retry-after ignored", 2, "soon", 2 * time.Second},
		{"negative retry-after ignored", 2, "-5", 2 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextRetryDelay(tc.attempt, tc.retryAfter, now); got != tc.want {
				t.Errorf("nextRetryDelay = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResponseStatusHandlesNilResponse(t *testing.T) {
	if got := responseStatus(nil); got != 0 {
		t.Errorf("responseStatus = %d, want 0", got)
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
