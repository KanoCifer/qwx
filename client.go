package qwx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultTimeout 是 HTTP 请求的默认超时。
const (
	DefaultTimeout = 30 * time.Second
	maxRetry       = 3
)

// Client 是和风天气 API 的基础客户端。
//
// 负责认证、请求发送、缓存读写与错误归类，不含具体接口。无状态，可并发
// 共享。零值不可用，请用 [NewClient] 构造。
//
// 接口族在各自的子包里，用同一个 Client 构造服务客户端。
type Client struct {
	apiHost   string
	http      *http.Client
	auth      Auth
	cache     Cache
	ttlPolicy TTLPolicy
	logger    *slog.Logger
	now       func() time.Time
}

// Option 配置 [Client]。
type Option func(*Client)

// WithHTTPClient 注入自定义的 [http.Client]。
//
// 传入的 client 不会被修改超时；如需超时，请自行设置 [http.Client.Timeout]。
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.http = hc
		}
	}
}

// WithAuth 设置认证方式。
func WithAuth(a Auth) Option {
	return func(c *Client) {
		if a != nil {
			c.auth = a
		}
	}
}

// WithCache 注入缓存实现。
//
// 传入 nil 等同于不缓存。缓存的键由端点包生成（见各端点方法的 CacheKey 说明），
// TTL 默认由端点语义决定，需要调整时用 [WithTTLPolicy]。
//
// 缓存键不含 API Host：不同 Host 对应不同开发者账号，一个 Cache 实例
// 不应跨账号共用，否则会串味。
func WithCache(cache Cache) Option {
	return func(c *Client) { c.cache = cache }
}

// WithTTLPolicy 设置按端点区分的缓存时长策略。
//
// 策略优先于端点包内置的默认时长；返回 0 表示该端点不缓存。
// 不设置时行为与默认一致，各端点用自己内置的 TTL。
//
// 策略在每次请求时求值，可以按时间或负载动态调整，实现方要保证并发安全。
//
// 缓存键不受策略影响：改 TTL 不会让已写入的旧条目提前失效。
func WithTTLPolicy(policy TTLPolicy) Option {
	return func(c *Client) {
		if policy != nil {
			c.ttlPolicy = policy
		}
	}
}

// WithClock 注入时钟，供测试断言 JWT 的 iat/exp。
func WithClock(now func() time.Time) Option {
	return func(c *Client) {
		if now != nil {
			c.now = now
		}
	}
}

// NewClient 构造 [Client]。
//
// auth，apiHost 为必填
// https://dev.qweather.com/docs/start/ QWeather官方文档
//
// cache 可为 nil。
func NewClient(auth Auth, apiHost string, opts ...Option) (*Client, error) {
	if auth == nil {
		return nil, fmt.Errorf("%w: auth is required", ErrInvalidConfig)
	}
	host, err := normalizeAPIHost(apiHost)
	if err != nil {
		return nil, err
	}

	c := &Client{
		apiHost: host,
		http:    &http.Client{Timeout: DefaultTimeout},
		auth:    auth,
		logger:  discardLogger(),
		now:     time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// normalizeAPIHost 规整控制台分配的 API Host。
//
// 只给 host 时补 "https://"，带 scheme 时原样保留 scheme，去掉尾部斜杠。
// Host 本身不校验是不是 qweatherapi.com —— 网关、自建代理、测试用假
// 上游都要能用。
func normalizeAPIHost(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("%w: api host is required (控制台设置页的专属 API Host)", ErrInvalidConfig)
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("%w: invalid api host %q: %v", ErrInvalidConfig, raw, err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("%w: invalid api host %q: missing host", ErrInvalidConfig, raw)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	// ForceQuery 与 RawQuery 是两个字段："host?" 解析后 RawQuery 为空但
	// ForceQuery 为真，不清它会保留尾部问号，拼路径时整个路径会被吞进查询串。
	u.ForceQuery = false
	return u.String(), nil
}

// Do 发送一次上游请求并把响应解析到 out。
//
// 命中缓存时直接返回缓存内容，不打上游。上游返回非 2xx 时，按状态码归类
// 到具体的 sentinel error，并尽可能解析出上游的错误细节。
//
// out 为 nil 时只取原始响应体。
func (c *Client) Do(ctx context.Context, req Request, out any) error {
	ttl := c.resolveTTL(ctx, req)
	useCache := c.cache != nil && req.CacheKey != "" && ttl > 0

	if useCache {
		hit, ok, err := c.cache.Get(ctx, req.CacheKey)
		switch {
		case err != nil:
			// 缓存读失败不该让业务请求失败，降级为直接请求上游。
			c.logCacheFailure(ctx, req, "cache read failed", err)
		case ok:
			c.logCacheHit(ctx, req)
			if err := decode(hit, out); err != nil {
				c.logDecodeFailure(ctx, req, err)
				return err
			}
			return nil
		default:
			c.logCacheMiss(ctx, req)
		}
	}

	payload, err := c.fetch(ctx, req)
	if err != nil {
		return err
	}

	if useCache {
		// 缓存写失败同样不该让业务请求失败，响应已经拿到了。
		if err := c.cache.Set(ctx, req.CacheKey, payload, ttl); err != nil {
			c.logCacheFailure(ctx, req, "cache write failed", err)
		}
	}

	if err := decode(payload, out); err != nil {
		c.logDecodeFailure(ctx, req, err)
		return err
	}
	return nil
}

// resolveTTL 决定本次请求的缓存时长。
//
// 优先级：单次上下文覆盖 > TTLPolicy > 端点包内置的默认时长。
// 返回值 <= 0 表示本次不缓存。
//
// 端点为空表示该请求不参与策略匹配，策略不会被调用，
// 缓存时长只由 Request.TTL 决定。
func (c *Client) resolveTTL(ctx context.Context, req Request) time.Duration {
	if d, ok := ttlOverrideFromContext(ctx); ok {
		return d
	}
	if c.ttlPolicy == nil || req.Endpoint == "" {
		return req.TTL
	}
	return c.ttlPolicy.TTL(req.Endpoint, req.TTL)
}

// decode 把响应体解析到 out。
func decode(payload []byte, out any) error {
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("%w: decode response: %v", ErrInvalidResponse, err)
	}
	return nil
}

func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	// 格式一：秒数
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	// 格式二：HTTP 日期（http.ParseTime 支持 RFC1123/RFC850/ANSI C 三种）
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0 // 时间已过，可以立即重试
		}
		return d, true
	}
	return 0, false
}

// 带退避，重试的HTTP请求
//
// 重试是上游 5xx 或网络错误触发的，每次重试前记一条日志。
func (c *Client) doWithRetry(ctx context.Context, req Request, httpReq *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	var retryDelay time.Duration = 0

	for i := 0; i <= maxRetry; i++ {

		if i > 0 && httpReq.GetBody != nil {
			body, err := httpReq.GetBody()
			if err != nil {
				return nil, fmt.Errorf("retry: rebuild body: %w", err)
			}
			httpReq.Body = body
		}
		resp, err = c.http.Do(httpReq)
		if err == nil && resp.StatusCode < 500 && resp.StatusCode != 429 {
			return resp, nil
		}

		retryDelay = time.Duration(i+1) * time.Second

		if resp != nil && resp.StatusCode == 429 {
			retryAfter := resp.Header.Get("Retry-After")
			if delay, isGet := parseRetryAfter(retryAfter, time.Now()); isGet {
				retryDelay = delay
			}
		}

		// 最后一轮或调用方已断开：不再重试
		if i == maxRetry || ctx.Err() != nil {
			if ctx.Err() != nil && resp != nil {
				resp.Body.Close()
			}
			return nil, err
		}

		if resp != nil {
			resp.Body.Close()
		}

		c.logRetry(ctx, req, i+1, responseStatus(resp), err)

		select {
		case <-time.After(retryDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return resp, err
}

// responseStatus 取响应状态码，响应为空时返回 0。
func responseStatus(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

// fetch 发起一次真实的上游请求，返回原始响应体。
func (c *Client) fetch(ctx context.Context, req Request) ([]byte, error) {
	u := c.apiHost + req.Path
	if len(req.Query) > 0 {
		u += "?" + req.Query.Encode()
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		c.logLocalFailure(ctx, req, "build request failed", err)
		return nil, fmt.Errorf("%w: build request: %v", ErrInvalidConfig, err)
	}
	if err := c.auth.Apply(httpReq); err != nil {
		c.logLocalFailure(ctx, req, "sign request failed", err)
		return nil, err
	}

	start := time.Now()
	resp, err := c.doWithRetry(ctx, req, httpReq)
	if err != nil {
		c.logUpstreamFailure(ctx, req, err, time.Since(start))
		return nil, fmt.Errorf("%w: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logUpstreamFailure(ctx, req, err, time.Since(start))
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		ue := newUpstreamError(resp.StatusCode, body)
		c.logUpstreamRejected(ctx, req, ue, time.Since(start))
		return nil, ue
	}
	c.logUpstream(ctx, req, resp.StatusCode, time.Since(start))
	return body, nil
}
