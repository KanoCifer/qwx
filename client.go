package qwx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL 是官方默认的 API Host。
//
// 控制台可以为每个项目分配专属 Host（如 abcxyz.qweatherapi.com），
// 专属 Host 通常更快也更稳定，优先用它。
const DefaultBaseURL = "https://devapi.qweather.com"

// DefaultTimeout 是 HTTP 请求的默认超时。
const DefaultTimeout = 30 * time.Second

// Client 是和风天气 API 的基础客户端。
//
// 负责认证、请求发送、缓存读写与错误归类，不含具体接口。无状态，可并发
// 共享。零值不可用，请用 [NewClient] 构造。
//
// 接口族在各自的子包里，用同一个 Client 构造服务客户端。
type Client struct {
	baseURL string
	http    *http.Client
	auth    Auth
	cache   Cache
	now     func() time.Time
}

// Option 配置 [Client]。
type Option func(*Client)

// WithBaseURL 指定 API Host。
//
// 可以带 scheme（"https://abcxyz.qweatherapi.com"），也可以只给 host
// （"abcxyz.qweatherapi.com"），后者会自动补上 https。
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

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
// TTL 由端点语义决定，调用方无需关心。
func WithCache(cache Cache) Option {
	return func(c *Client) { c.cache = cache }
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
// auth 为必填 —— 没有认证方式所有请求都会被上游拒绝。cache 可为 nil。
func NewClient(auth Auth, opts ...Option) (*Client, error) {
	if auth == nil {
		return nil, fmt.Errorf("%w: auth is required", ErrInvalidConfig)
	}
	c := &Client{
		baseURL: DefaultBaseURL,
		http:    &http.Client{Timeout: DefaultTimeout},
		auth:    auth,
		now:     time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.baseURL == "" {
		return nil, fmt.Errorf("%w: base URL is required", ErrInvalidConfig)
	}
	if !strings.Contains(c.baseURL, "://") {
		c.baseURL = "https://" + c.baseURL
	}
	return c, nil
}

// Do 发送一次上游请求并把响应解析到 out。
//
// 命中缓存时直接返回缓存内容，不打上游。上游返回非 2xx 时，按状态码归类
// 到具体的 sentinel error，并尽可能解析出上游的错误细节。
//
// out 为 nil 时只取原始响应体。
func (c *Client) Do(ctx context.Context, req Request, out any) error {
	if c.cache != nil && req.CacheKey != "" && req.TTL > 0 {
		hit, ok, err := c.cache.Get(ctx, req.CacheKey)
		if err == nil && ok {
			return decode(hit, out)
		}
		// 缓存读失败不该让业务请求失败，降级为直接请求上游。
	}

	payload, err := c.fetch(ctx, req)
	if err != nil {
		return err
	}

	if c.cache != nil && req.CacheKey != "" && req.TTL > 0 {
		// 缓存写失败同样不该让业务请求失败，响应已经拿到了。
		_ = c.cache.Set(ctx, req.CacheKey, payload, req.TTL)
	}

	return decode(payload, out)
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

// fetch 发起一次真实的上游请求，返回原始响应体。
func (c *Client) fetch(ctx context.Context, req Request) ([]byte, error) {
	u := c.baseURL + req.Path
	if len(req.Query) > 0 {
		u += "?" + req.Query.Encode()
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", ErrInvalidConfig, err)
	}
	if err := c.auth.Apply(httpReq); err != nil {
		return nil, err
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", ErrUpstreamUnavailable, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newUpstreamError(resp.StatusCode, body)
	}
	return body, nil
}
