package qwx

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// MethodGet 是 GET 方法。
//
// 全库只读，没有端点用到别的方法。端点包统一用它构造请求：方法字符串
// 只有一处，外部 [Doer] 实现读到的也是同一个值。
const MethodGet = "GET"

// Request 描述一次上游请求。
//
// 端点子包用它构造请求：[Client.Do] 负责缓存读写、发送与错误归类。
type Request struct {
	// Method 是 HTTP 方法。留空按 GET 处理。
	//
	// 端点包填 [MethodGet]。实际发出去的方法由 [Request.HTTPMethod] 决定，
	// 外部实现 [Doer] 时读它就行。
	Method string
	// Path 是接口路径，如 "/weather/v1/current/39.92/116.41"。
	Path string
	// Query 是查询参数。
	Query url.Values
	// CacheKey 是缓存键。留空表示不缓存。
	CacheKey string
	// Endpoint 是该请求所属的端点，由端点包填写。
	//
	// 用于 [TTLPolicy] 按端点匹配。留空表示不参与策略匹配：
	// 此时策略不会被调用，缓存时长只由 [Request.TTL] 决定。
	Endpoint Endpoint
	// TTL 是端点包内置的默认缓存时长。小于等于 0 表示不缓存。
	//
	// 最终时长由 [Client] 解析：单次覆盖 > TTLPolicy > 本字段。
	TTL time.Duration
}

// HTTPMethod 返回实际发出的 HTTP 方法。
//
// 留空按 GET 处理 —— 约定只在这一处：端点包不写死方法，日志与外部
// [Doer] 实现读同一个值。
func (r Request) HTTPMethod() string {
	if r.Method == "" {
		return http.MethodGet
	}
	return r.Method
}

// Doer 发起上游请求。
//
// [Client] 实现该接口。端点子包只依赖这个接口，因此可以注入假实现做测试。
type Doer interface {
	// Do 发送请求并把响应解析到 out。out 为 nil 时只取原始响应体。
	Do(ctx context.Context, req Request, out any) error
}

var _ Doer = (*Client)(nil)
