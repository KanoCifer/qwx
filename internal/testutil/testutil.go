// Package testutil 提供跨包测试的公共辅助。
//
// 内部包：假上游与测试客户端，被各端点包的测试文件共用。
package testutil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
	base := []qwx.Option{qwx.WithBaseURL(fs.URL)}
	c, err := qwx.NewClient(auth, append(base, opts...)...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}
