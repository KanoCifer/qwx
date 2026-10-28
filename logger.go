package qwx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// 日志字段名。集中定义，用户可以在自己的 handler 里按字段名过滤。
const (
	logFieldEndpoint      = "endpoint"
	logFieldMethod        = "method"
	logFieldPath          = "path"
	logFieldStatus        = "status"
	logFieldDuration      = "duration_ms"
	logFieldCacheKey      = "cache_key"
	logFieldAttempt       = "attempt"
	logFieldError         = "error"
	logFieldUpstreamCode  = "upstream_code"
	logFieldUpstreamTitle = "upstream_title"
)

// WithLogger 注入日志器。
//
// 不注入时不产生任何日志输出 —— SDK 不替用户决定日志去向。传入 nil
// 等同于不注入。
//
// 级别由 handler 决定，SDK 不再过滤：handler 放行 Debug，就会收到缓存
// 命中这类明细。
//
// 日志里没有认证凭据（凭据在请求头上）与响应体，查询参数也不记录；但有
// 接口路径与缓存键，两者可能带位置坐标。SDK 不对坐标脱敏，日志的可见
// 范围自己把握，要脱敏就在注入时包一层自己的 handler。
func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// discardLogger 返回丢弃全部记录的 logger。
//
// slog.DiscardHandler 不做格式化，比写 io.Discard 的 TextHandler 便宜。
func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// log 记录一条日志。
//
// 先问 handler 放不放行这个级别，不放行就不构造字段：默认静默时，
// 每次请求的额外开销只有一次 Enabled 调用。
func (c *Client) log(ctx context.Context, level slog.Level, msg string, req Request, attrs ...slog.Attr) {
	if c.logger == nil || !c.logger.Enabled(ctx, level) {
		return
	}
	base := make([]slog.Attr, 0, len(attrs)+3)
	if req.Endpoint != "" {
		base = append(base, slog.String(logFieldEndpoint, string(req.Endpoint)))
	}
	if req.Path != "" {
		base = append(base, slog.String(logFieldPath, req.Path))
	}
	base = append(base, attrs...)
	c.logger.LogAttrs(ctx, level, msg, base...)
}

// logCacheHit 记录缓存命中。
func (c *Client) logCacheHit(ctx context.Context, req Request) {
	c.log(ctx, slog.LevelDebug, "cache hit", req,
		slog.String(logFieldCacheKey, req.CacheKey),
	)
}

// logCacheMiss 记录缓存未命中，请求会打到上游。
func (c *Client) logCacheMiss(ctx context.Context, req Request) {
	c.log(ctx, slog.LevelDebug, "cache miss", req,
		slog.String(logFieldCacheKey, req.CacheKey),
	)
}

// logCacheFailure 记录缓存读写故障。
//
// 缓存故障不影响请求结果：SDK 降级为直连上游，这里只留一条告警。
func (c *Client) logCacheFailure(ctx context.Context, req Request, msg string, err error) {
	attrs := []slog.Attr{slog.String(logFieldCacheKey, req.CacheKey)}
	if err != nil {
		attrs = append(attrs, slog.String(logFieldError, sanitizeErrorText(err)))
	}
	c.log(ctx, slog.LevelWarn, msg, req, attrs...)
}

// logUpstream 记录一次成功的上游请求。
func (c *Client) logUpstream(ctx context.Context, req Request, status int, dur time.Duration) {
	c.log(ctx, slog.LevelInfo, "upstream request", req,
		slog.String(logFieldMethod, requestMethod(req)),
		slog.Int(logFieldStatus, status),
		slog.Int64(logFieldDuration, dur.Milliseconds()),
	)
}

// logUpstreamRejected 记录上游的非 2xx 响应。
//
// 4xx 是请求本身不合法，记 Warn；5xx 是上游故障，记 Error。
func (c *Client) logUpstreamRejected(ctx context.Context, req Request, ue *UpstreamError, dur time.Duration) {
	level := slog.LevelWarn
	if ue.StatusCode >= 500 {
		level = slog.LevelError
	}
	attrs := []slog.Attr{
		slog.String(logFieldMethod, requestMethod(req)),
		slog.Int(logFieldStatus, ue.StatusCode),
		slog.Int64(logFieldDuration, dur.Milliseconds()),
	}
	if ue.Code != "" {
		attrs = append(attrs, slog.String(logFieldUpstreamCode, ue.Code))
	}
	if ue.Title != "" {
		attrs = append(attrs, slog.String(logFieldUpstreamTitle, ue.Title))
	}
	c.log(ctx, level, "upstream rejected", req, attrs...)
}

// logUpstreamFailure 记录上游不可达：网络错误、超时、响应体读失败。
func (c *Client) logUpstreamFailure(ctx context.Context, req Request, err error, dur time.Duration) {
	attrs := []slog.Attr{
		slog.String(logFieldMethod, requestMethod(req)),
		slog.Int64(logFieldDuration, dur.Milliseconds()),
	}
	if err != nil {
		attrs = append(attrs, slog.String(logFieldError, sanitizeErrorText(err)))
	}
	c.log(ctx, slog.LevelError, "upstream request failed", req, attrs...)
}

// logRetry 记录一次重试。
func (c *Client) logRetry(ctx context.Context, req Request, attempt, status int, err error) {
	attrs := []slog.Attr{
		slog.Int(logFieldAttempt, attempt),
		slog.Int(logFieldStatus, status),
	}
	if err != nil {
		attrs = append(attrs, slog.String(logFieldError, sanitizeErrorText(err)))
	}
	c.log(ctx, slog.LevelWarn, "upstream retry", req, attrs...)
}

// logDecodeFailure 记录响应解析失败。
func (c *Client) logDecodeFailure(ctx context.Context, req Request, err error) {
	attrs := []slog.Attr{}
	if err != nil {
		attrs = append(attrs, slog.String(logFieldError, sanitizeErrorText(err)))
	}
	c.log(ctx, slog.LevelError, "decode response failed", req, attrs...)
}

// logLocalFailure 记录请求在本地准备阶段失败：构造请求、写入认证。
func (c *Client) logLocalFailure(ctx context.Context, req Request, msg string, err error) {
	attrs := []slog.Attr{}
	if err != nil {
		attrs = append(attrs, slog.String(logFieldError, sanitizeErrorText(err)))
	}
	c.log(ctx, slog.LevelError, msg, req, attrs...)
}

// sanitizeErrorText 去掉错误文本里的查询参数。
//
// 传输层的错误是 *url.Error，文本里内嵌了完整 URL，含查询参数。GeoAPI 的
// location=地名 这类用户输入会跟着漏进日志，这里只留 URL 的 path 部分，
// 错误原因照旧。
func sanitizeErrorText(err error) string {
	if err == nil {
		return ""
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		if u, perr := url.Parse(ue.URL); perr == nil {
			u.RawQuery = ""
			u.ForceQuery = false
			u.Fragment = ""
			return fmt.Sprintf("%s %q: %v", ue.Op, u.String(), ue.Err)
		}
	}
	return err.Error()
}

// requestMethod 取请求方法，留空按 GET 处理，与实际发出的请求一致。
func requestMethod(req Request) string {
	if req.Method == "" {
		return http.MethodGet
	}
	return req.Method
}
