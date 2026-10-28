package qwx_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
)

// captureHandler 收集日志记录，供断言。
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

// Enabled 放行所有级别。
func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

// Handle 收下记录。Record 的字段只在 Handle 期间有效，必须克隆。
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *captureHandler) WithGroup(string) slog.Handler { return h }

// find 返回消息匹配的全部记录。
func (h *captureHandler) find(msg string) []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []slog.Record
	for _, r := range h.records {
		if r.Message == msg {
			out = append(out, r)
		}
	}
	return out
}

// contains 判断是否有记录的文本里出现了 needle。
func (h *captureHandler) contains(needle string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if strings.Contains(r.Message, needle) {
			return true
		}
		found := false
		r.Attrs(func(a slog.Attr) bool {
			if strings.Contains(a.Value.String(), needle) {
				found = true
				return false
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

// recordAttr 取记录里的字段值。
func recordAttr(r slog.Record, key string) (slog.Value, bool) {
	var (
		v     slog.Value
		found bool
	)
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			v = a.Value
			found = true
			return false
		}
		return true
	})
	return v, found
}

func assertAttr(t *testing.T, r slog.Record, key, want string) {
	t.Helper()
	v, ok := recordAttr(r, key)
	if !ok {
		t.Errorf("attr %q missing", key)
		return
	}
	if v.String() != want {
		t.Errorf("attr %q = %q, want %q", key, v.String(), want)
	}
}

func assertAttrInt(t *testing.T, r slog.Record, key string, want int64) {
	t.Helper()
	v, ok := recordAttr(r, key)
	if !ok {
		t.Errorf("attr %q missing", key)
		return
	}
	if v.Int64() != want {
		t.Errorf("attr %q = %d, want %d", key, v.Int64(), want)
	}
}

func TestLoggerRecordsUpstreamRequest(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	h := &captureHandler{}
	c := testutil.NewClient(t, fs, qwx.WithLogger(slog.New(h)))

	req := qwx.Request{Path: "/geo/v2/city/lookup", Endpoint: "geo/v2/city/lookup"}
	if err := c.Do(context.Background(), req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}

	recs := h.find("upstream request")
	if len(recs) != 1 {
		t.Fatalf("upstream request records = %d, want 1", len(recs))
	}
	r := recs[0]
	if r.Level != slog.LevelInfo {
		t.Errorf("level = %v, want INFO", r.Level)
	}
	assertAttr(t, r, "endpoint", "geo/v2/city/lookup")
	assertAttr(t, r, "method", "GET")
	assertAttr(t, r, "path", "/geo/v2/city/lookup")
	assertAttrInt(t, r, "status", 200)
	if _, ok := recordAttr(r, "duration_ms"); !ok {
		t.Error("attr duration_ms missing")
	}
}

func TestLoggerRecordsCacheHitAndMiss(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	h := &captureHandler{}
	c := testutil.NewClient(t, fs,
		qwx.WithLogger(slog.New(h)),
		qwx.WithCache(qwx.NewMemoryCache()),
	)

	req := qwx.Request{Path: "/geo/v2/city/top", CacheKey: "qwx:geo:topcity", TTL: time.Minute}
	for i := 0; i < 2; i++ {
		if err := c.Do(context.Background(), req, nil); err != nil {
			t.Fatalf("Do: %v", err)
		}
	}

	if got := len(h.find("cache miss")); got != 1 {
		t.Errorf("cache miss records = %d, want 1", got)
	}
	hits := h.find("cache hit")
	if len(hits) != 1 {
		t.Fatalf("cache hit records = %d, want 1", len(hits))
	}
	if hits[0].Level != slog.LevelDebug {
		t.Errorf("level = %v, want DEBUG", hits[0].Level)
	}
	assertAttr(t, hits[0], "cache_key", "qwx:geo:topcity")

	// 命中缓存时不再打上游。
	if got := len(h.find("upstream request")); got != 1 {
		t.Errorf("upstream request records = %d, want 1", got)
	}
}

func TestLoggerRecordsCacheFailure(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	h := &captureHandler{}
	c := testutil.NewClient(t, fs,
		qwx.WithLogger(slog.New(h)),
		qwx.WithCache(failingCache{}),
	)

	// 缓存故障不该让请求失败。
	if err := c.Do(context.Background(), qwx.Request{
		Path:     "/x",
		CacheKey: "k",
		TTL:      time.Minute,
	}, nil); err != nil {
		t.Fatalf("Do with broken cache: %v", err)
	}

	recs := h.find("cache read failed")
	if len(recs) != 1 {
		t.Fatalf("cache read failed records = %d, want 1", len(recs))
	}
	if recs[0].Level != slog.LevelWarn {
		t.Errorf("level = %v, want WARN", recs[0].Level)
	}
	assertAttr(t, recs[0], "cache_key", "k")
}

func TestLoggerRecordsUpstreamRejected(t *testing.T) {
	body := `{"error":{"status":400,"title":"Invalid Parameters","invalidParams":["location"]}}`
	fs := testutil.NewFakeServer(t, body)
	fs.Handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(body))
	}
	h := &captureHandler{}
	c := testutil.NewClient(t, fs, qwx.WithLogger(slog.New(h)))

	if err := c.Do(context.Background(), qwx.Request{Path: "/x", Endpoint: "weather/v1/current"}, nil); err == nil {
		t.Fatal("expected error")
	}

	recs := h.find("upstream rejected")
	if len(recs) != 1 {
		t.Fatalf("upstream rejected records = %d, want 1", len(recs))
	}
	r := recs[0]
	// 4xx 是请求本身不合法，记 Warn。
	if r.Level != slog.LevelWarn {
		t.Errorf("level = %v, want WARN", r.Level)
	}
	assertAttrInt(t, r, "status", 400)
	assertAttr(t, r, "upstream_title", "Invalid Parameters")
	assertAttr(t, r, "endpoint", "weather/v1/current")
}

func TestLoggerRecordsUpstreamServerError(t *testing.T) {
	body := `{"error":{"status":500,"title":"Internal Server Error"}}`
	fs := testutil.NewFakeServer(t, body)
	fs.Handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(body))
	}
	h := &captureHandler{}
	c := testutil.NewClient(t, fs, qwx.WithLogger(slog.New(h)))

	if err := c.Do(context.Background(), qwx.Request{Path: "/x"}, nil); err == nil {
		t.Fatal("expected error")
	}

	// 5xx 是上游故障，记 Error。
	recs := h.find("upstream rejected")
	if len(recs) != 1 {
		t.Fatalf("upstream rejected records = %d, want 1", len(recs))
	}
	if recs[0].Level != slog.LevelError {
		t.Errorf("level = %v, want ERROR", recs[0].Level)
	}
	assertAttrInt(t, recs[0], "status", 500)

	// 重试也要留痕：最多 3 次，每次一条。
	retries := h.find("upstream retry")
	if len(retries) != 3 {
		t.Fatalf("upstream retry records = %d, want 3", len(retries))
	}
	for i, r := range retries {
		if r.Level != slog.LevelWarn {
			t.Errorf("retry %d level = %v, want WARN", i, r.Level)
		}
		assertAttrInt(t, r, "attempt", int64(i+1))
		assertAttrInt(t, r, "status", 500)
	}
}

func TestLoggerRecordsDecodeFailure(t *testing.T) {
	fs := testutil.NewFakeServer(t, `not json`)
	h := &captureHandler{}
	c := testutil.NewClient(t, fs, qwx.WithLogger(slog.New(h)))

	var out map[string]any
	if err := c.Do(context.Background(), qwx.Request{Path: "/x"}, &out); err == nil {
		t.Fatal("expected error")
	}

	recs := h.find("decode response failed")
	if len(recs) != 1 {
		t.Fatalf("decode response failed records = %d, want 1", len(recs))
	}
	if recs[0].Level != slog.LevelError {
		t.Errorf("level = %v, want ERROR", recs[0].Level)
	}
}

func TestLoggerDoesNotLeakCredentials(t *testing.T) {
	// 凭据在请求头上，不该出现在任何日志文本里。
	const secret = "super-secret-api-key"
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	h := &captureHandler{}

	auth, err := qwx.NewAPIKeyAuth(secret)
	if err != nil {
		t.Fatalf("NewAPIKeyAuth: %v", err)
	}
	c, err := qwx.NewClient(auth, fs.URL, qwx.WithLogger(slog.New(h)))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := c.Do(context.Background(), qwx.Request{Path: "/x", CacheKey: "k", TTL: time.Minute}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}

	if h.contains(secret) {
		t.Error("log contains the api key")
	}
}

func TestLoggerNilIsIgnored(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	c := testutil.NewClient(t, fs, qwx.WithLogger(nil))
	if err := c.Do(context.Background(), qwx.Request{Path: "/x"}, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestLoggerSanitizesErrorText(t *testing.T) {
	// 传输层错误是 *url.Error，文本里内嵌完整 URL。查询参数是用户输入，
	// 不该跟着进日志；路径要留着，否则错误没法定位。
	h := &captureHandler{}
	auth, err := qwx.NewAPIKeyAuth("k")
	if err != nil {
		t.Fatalf("NewAPIKeyAuth: %v", err)
	}
	c, err := qwx.NewClient(auth, "127.0.0.1:1", qwx.WithLogger(slog.New(h)))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	// ctx 已取消：请求不会真的发出去，但错误里照样带着完整 URL。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := qwx.Request{
		Path:  "/geo/v2/city/lookup",
		Query: url.Values{"location": {"杭州"}},
	}
	if err := c.Do(ctx, req, nil); err == nil {
		t.Fatal("expected error")
	}

	recs := h.find("upstream request failed")
	if len(recs) != 1 {
		t.Fatalf("upstream request failed records = %d, want 1", len(recs))
	}
	v, ok := recordAttr(recs[0], "error")
	if !ok {
		t.Fatal("attr error missing")
	}
	text := v.String()
	if strings.Contains(text, "location=") {
		t.Errorf("error text leaks the query: %q", text)
	}
	if strings.Contains(text, "杭州") {
		t.Errorf("error text leaks the location: %q", text)
	}
	if !strings.Contains(text, "/geo/v2/city/lookup") {
		t.Errorf("error text should keep the path: %q", text)
	}
}
