// Package testutil 提供跨包测试的公共辅助。
//
// 内部包：假上游与测试客户端，被各端点包的测试文件共用。
package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
)

// FakeServer 是记录请求的假上游。
type FakeServer struct {
	*httptest.Server
	// LastPath 是最后收到的请求路径。
	LastPath string
	// LastQuery 是最后收到的查询参数。
	LastQuery url.Values
	// Hits 记录收到的请求次数。
	Hits int
	// Handler 可覆盖默认行为，用于构造错误响应等场景。
	Handler func(w http.ResponseWriter, r *http.Request)
}

// NewFakeServer 构造一个返回固定 JSON 的假上游。
func NewFakeServer(t *testing.T, body string) *FakeServer {
	t.Helper()
	fs := &FakeServer{}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.Hits++
		fs.LastPath = r.URL.Path
		fs.LastQuery = r.URL.Query()
		if fs.Handler != nil {
			fs.Handler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(fs.Close)
	return fs
}

// AssertPath 断言最后收到的请求路径。
func (fs *FakeServer) AssertPath(t *testing.T, want string) {
	t.Helper()
	if fs.LastPath != want {
		t.Errorf("path = %q, want %q", fs.LastPath, want)
	}
}

// AssertQuery 断言最后收到的查询参数。
func (fs *FakeServer) AssertQuery(t *testing.T, key, want string) {
	t.Helper()
	if got := fs.LastQuery.Get(key); got != want {
		t.Errorf("query %q = %q, want %q", key, got, want)
	}
}

// NewClient 构造一个指向假上游的基础客户端。
func NewClient(t *testing.T, fs *FakeServer, opts ...qwx.Option) *qwx.Client {
	t.Helper()
	auth, err := qwx.NewAPIKeyAuth("test-key")
	if err != nil {
		t.Fatalf("NewAPIKeyAuth: %v", err)
	}
	c, err := qwx.NewClient(auth, fs.URL, opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// RecordingDoer 记录收到的请求，用于断言端点包传出的 [qwx.Request]。
//
// 端点包的测试要断言端点标识与缓存时长，用假上游拿不到这些信息。
type RecordingDoer struct {
	// Requests 按顺序记录收到的请求。
	Requests []qwx.Request
}

// Do 记录请求，不发网络请求。
func (d *RecordingDoer) Do(_ context.Context, req qwx.Request, _ any) error {
	d.Requests = append(d.Requests, req)
	return nil
}

// Last 返回最后收到的请求。
func (d *RecordingDoer) Last(t *testing.T) qwx.Request {
	t.Helper()
	if len(d.Requests) == 0 {
		t.Fatal("no request recorded")
	}
	return d.Requests[len(d.Requests)-1]
}

// RecordingCache 记录写入缓存的数据与时长。
type RecordingCache struct {
	mu   sync.Mutex
	data map[string][]byte
	ttls map[string]time.Duration
}

// NewRecordingCache 构造一个记录写入内容的缓存。
func NewRecordingCache() *RecordingCache {
	return &RecordingCache{
		data: make(map[string][]byte),
		ttls: make(map[string]time.Duration),
	}
}

// Get 读取缓存。
func (c *RecordingCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	d, ok := c.data[key]
	return d, ok, nil
}

// Set 写入缓存并记录时长。
func (c *RecordingCache) Set(_ context.Context, key string, data []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = data
	c.ttls[key] = ttl
	return nil
}

// TTL 返回某个键写入时用的时长。
func (c *RecordingCache) TTL(key string) (time.Duration, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	d, ok := c.ttls[key]
	return d, ok
}

// 编译期断言。
var (
	_ qwx.Doer  = (*RecordingDoer)(nil)
	_ qwx.Cache = (*RecordingCache)(nil)
)
