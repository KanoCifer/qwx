package qwx

import (
	"context"
	"net/url"
	"time"
)

// Request 描述一次上游请求。
//
// 端点子包用它构造请求：[Client.Do] 负责缓存读写、发送与错误归类。
type Request struct {
	// Method 是 HTTP 方法。留空按 GET 处理。
	Method string
	// Path 是接口路径，如 "/weather/v1/current/39.92/116.41"。
	Path string
	// Query 是查询参数。
	Query url.Values
	// CacheKey 是缓存键。留空表示不缓存。
	CacheKey string
	// TTL 是缓存时长。小于等于 0 表示不缓存。
	TTL time.Duration
}

// Doer 发起上游请求。
//
// [Client] 实现该接口。端点子包只依赖这个接口，因此可以注入假实现做测试。
type Doer interface {
	// Do 发送请求并把响应解析到 out。out 为 nil 时只取原始响应体。
	Do(ctx context.Context, req Request, out any) error
}

// 编译期断言。
var _ Doer = (*Client)(nil)
