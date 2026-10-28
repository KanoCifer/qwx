// Package qwx 是和风天气（QWeather）开发服务的 Go 客户端。
//
// 它覆盖官方文档中的核心接口族：GeoAPI 城市/POI 检索、天气预报（新一代
// /weather/v1/*）、空气质量、预警、海洋潮汐、生活指数与分钟级降水预报。
//
// # 认证
//
// 官方支持两种认证方式，本包都支持，但**不要同时使用**：
//
//   - JWT（Ed25519 签名，推荐）：见 [NewJWTAuth]。需要控制台生成的
//     Ed25519 私钥，以及凭据的 Key ID、Project ID、Developer ID。
//   - API KEY：见 [NewAPIKeyAuth]。官方已公告从 2027 年 1 月 1 日起
//     限制用 API KEY 认证的每日请求量，新项目建议直接用 JWT。
//
// # 缓存
//
// 本包不内置缓存实现，而是定义 [Cache] 接口由调用方注入（Redis、内存均可）。
// 未注入时每次请求都打到上游。
//
// # 并发
//
// [Client] 无状态，可并发共享。所有方法都接受 [context.Context]。
package qwx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// Client 是和风天气 API 的客户端。
//
// 无状态，可并发共享。零值不可用，请用 [NewClient] 构造。
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
// 传入 nil 等同于不缓存。缓存的键由本包生成（见各端点方法的 CacheKey 说明），
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

// request 是一次上游请求的描述。
type request struct {
	method string
	path   string
	query  url.Values
	body   io.Reader
}

// do 发送请求并解析响应。
//
// 命中缓存时直接返回缓存内容，不打上游。upstream 返回非 2xx 时，
// 按状态码归类到具体的 sentinel error，并尽可能解析出上游的错误细节。
func (c *Client) do(ctx context.Context, req request, cacheKey string, ttl time.Duration, out any) error {
	if c.cache != nil && cacheKey != "" && ttl > 0 {
		hit, ok, err := c.cache.Get(ctx, cacheKey)
		if err == nil && ok {
			return json.Unmarshal(hit, out)
		}
		// 缓存读失败不该让业务请求失败，降级为直接请求上游。
	}

	payload, err := c.fetch(ctx, req)
	if err != nil {
		return err
	}

	if c.cache != nil && cacheKey != "" && ttl > 0 {
		// 缓存写失败同样不该让业务请求失败，响应已经拿到了。
		_ = c.cache.Set(ctx, cacheKey, payload, ttl)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("%w: decode response: %v", ErrInvalidResponse, err)
	}
	return nil
}

// fetch 发起一次真实的上游请求，返回原始响应体。
func (c *Client) fetch(ctx context.Context, req request) ([]byte, error) {
	u := c.baseURL + req.path
	if len(req.query) > 0 {
		u += "?" + req.query.Encode()
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method, u, req.body)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %v", ErrInvalidConfig, err)
	}
	for k, vs := range req.query {
		for _, v := range vs {
			httpReq.Header.Add(k, v)
		}
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
