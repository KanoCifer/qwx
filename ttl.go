package qwx

import (
	"context"
	"time"
)

// Endpoint 标识一个 API 端点。
//
// 各端点包导出自己的端点常量（如 [github.com/KanoCifer/qwx/v1/weather].EndpointCurrent），
// 用户用它们作为 [TTLPolicy] 的键。取值发布后不再变更。
//
// 值形如 "weather/v1/current"，与上游路径一致但不含前导斜杠。
type Endpoint string

// TTLPolicy 按端点决定缓存时长。
//
// 返回值 <= 0 表示该端点本次不缓存；正数是缓存时长。
// fallback 是端点包内置的默认时长，策略对该端点没有意见时原样返回即可。
//
// 实现必须并发安全：同一个客户端会被多个 goroutine 共享，策略会被并发调用。
type TTLPolicy interface {
	// TTL 返回该端点本次请求使用的缓存时长。
	TTL(endpoint Endpoint, fallback time.Duration) time.Duration
}

// TTLPolicyFunc 把普通函数适配成 [TTLPolicy]。
type TTLPolicyFunc func(endpoint Endpoint, fallback time.Duration) time.Duration

// TTL 实现 [TTLPolicy]。
func (f TTLPolicyFunc) TTL(endpoint Endpoint, fallback time.Duration) time.Duration {
	return f(endpoint, fallback)
}

// TTLMap 是 map 形态的 [TTLPolicy]：键是端点，值是缓存时长。
//
// 键不存在时沿用端点包内置的默认时长；值为 0 表示该端点不缓存。
type TTLMap map[Endpoint]time.Duration

// TTL 实现 [TTLPolicy]。
func (m TTLMap) TTL(endpoint Endpoint, fallback time.Duration) time.Duration {
	if d, ok := m[endpoint]; ok {
		return d
	}
	return fallback
}

// ttlOverrideKey 是单次 TTL 覆盖在 context 里的键。
type ttlOverrideKey struct{}

// WithTTLOverride 返回一个带单次缓存时长覆盖的 ctx。
//
// 覆盖优先级高于 [TTLPolicy] 与端点包内置的默认时长。d <= 0 表示这一次
// 不读也不写缓存。
//
// 覆盖作用于同一个 ctx 发出的所有请求，不要把这个 ctx 存下来复用。
// ctx 不能为 nil。
func WithTTLOverride(ctx context.Context, d time.Duration) context.Context {
	return context.WithValue(ctx, ttlOverrideKey{}, d)
}

// ttlOverrideFromContext 取出 ctx 里的单次 TTL 覆盖。
//
// ctx 为 nil 时视为没有覆盖：调用方多半是忘了传 context，
// 请求会照常走到上游，由 http 层报错，不要在这里 panic。
func ttlOverrideFromContext(ctx context.Context) (time.Duration, bool) {
	if ctx == nil {
		return 0, false
	}
	d, ok := ctx.Value(ttlOverrideKey{}).(time.Duration)
	return d, ok
}
