// Package alert 提供预警接口（/weatheralert/v1/*）。
//
// 这一代只认坐标：位置必须是经纬度路径参数，LocationID 会被上游拒绝。
// 要按 LocationID 或地名查预警，先用 geo 包的 LookupCity 换坐标。
package alert

import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Client 是预警接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造预警接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("alert: nil base client")
	}
	return &Client{base: base}
}

// Current 查询坐标点上的预警。
//
// 调 GET /weatheralert/v1/current/{latitude}/{longitude}。
//
// opts.Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
//
// 缓存键：`qwx:alert:v1:<lat>,<lon>` 加查询参数后缀。
//
// 默认 TTL 5 分钟，可用 [qwx.WithTTLPolicy] 按 [Endpoint] 覆盖。
func (c *Client) Current(ctx context.Context, loc qwx.Location, opts Options) (*Result, error) {
	if err := opts.Lang.Validate(); err != nil {
		return nil, err
	}
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	p := params(opts)
	var out Result
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/weatheralert/v1/current/" + lat + "/" + lon,
		Query:    p.Query(),
		Endpoint: Endpoint,
		CacheKey: currentCacheKey(loc, p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// currentCacheKey 预警实况的缓存键。
func currentCacheKey(loc qwx.Location, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyAlert).
		PathCoord(loc.Lat, loc.Lon).
		Params(p).
		String()
}
