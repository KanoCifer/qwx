package console

import (
	"context"
	"fmt"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Client 是控制台 API 的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造控制台 API 的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("console: nil base client")
	}
	return &Client{base: base}
}

// Finance 查询财务与计费的汇总信息。
//
// 调 GET /finance/v1/summary。
//
// 凭据默认没有权限访问这个端点：帐号所有者要在控制台的
// 「项目管理 - 凭据 - 控制台权限」里勾选「允许访问财务汇总数据」，
// 否则上游拒绝。
//
// 返回的数据截止至上一个小时或更早，与控制台网页上显示的可能有 1 小时
// 或更长的延迟，以 [FinanceResult.AsOf] 为准。
//
// 缓存键：`qwx:console:finance`。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointFinance] 覆盖。
func (c *Client) Finance(ctx context.Context) (*FinanceResult, error) {
	var out FinanceResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/finance/v1/summary",
		Endpoint: EndpointFinance,
		CacheKey: cachekey.New(cachekey.FamilyConsole).Path("finance").String(),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Stats 查询最近 24 小时的 API 请求量统计。
//
// 调 GET /metrics/v1/stats。
//
// opts.Project 与 opts.Credential 互斥，都留空时统计帐号下所有项目与凭据。
// 两个都填会在客户端被挡下，不打上游。
//
// 凭据默认没有权限访问这个端点：帐号所有者要在控制台的
// 「项目管理 - 凭据 - 控制台权限」里勾选对应的权限，否则上游拒绝。
//
// 返回的数据截止至上一个小时或更早，以 [StatsResult.AsOf] 为准。
//
// 缓存键：`qwx:console:stats` 加查询参数后缀（项目或凭据 ID）。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointStats] 覆盖。
func (c *Client) Stats(ctx context.Context, opts StatsOptions) (*StatsResult, error) {
	if opts.Project != "" && opts.Credential != "" {
		return nil, fmt.Errorf("%w: project and credential are mutually exclusive", qwx.ErrInvalidParameter)
	}
	p := opts.params()
	var out StatsResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/metrics/v1/stats",
		Query:    p.Query(),
		Endpoint: EndpointStats,
		CacheKey: cachekey.New(cachekey.FamilyConsole).Path("stats").Params(p).String(),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
