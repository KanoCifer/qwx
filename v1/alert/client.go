// Package alert 提供新一代预警接口（/weatheralert/v1/*）。
//
// 这一代只认坐标：位置必须是经纬度路径参数，LocationID 会被上游拒绝。
// 旧一代预警在 qwx/v7 包。
package alert

import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/coord"
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
// 缓存键：`qwx:alert:v1:<lat>,<lon>` 加查询参数后缀。
//
// 默认 TTL 5 分钟，可用 [qwx.WithTTLPolicy] 按 [EndpointCurrent] 覆盖。
func (c *Client) Current(ctx context.Context, loc qwx.Location, opts Options) (*Result, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out Result
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/weatheralert/v1/current/" + lat + "/" + lon,
		Query:    opts.params(),
		Endpoint: EndpointCurrent,
		CacheKey: "qwx:alert:v1:" + coord.Format(loc.Lat) + "," + coord.Format(loc.Lon) + opts.cacheSuffix(),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
