// Package indices 提供旧一代生活指数接口（/v7/indices/*）。
//
// 生活指数至今只有 v7 版本 —— 实测 /weather/v1/indices/* 与 /v1/indices/* 全部 404。
//
// 旧一代与新一代在位置形态、字段单位、时间基准、错误体与缓存键上都不同，
// 本包不与 v1 下的包混用任何逻辑。
package indices

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Client 是生活指数接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造生活指数接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("indices: nil base client")
	}
	return &Client{base: base}
}

// Indices 查询生活指数。
//
// 调 GET /v7/indices/{days}d?type={type}&location=...。
//
// 上游至今只有 v7 版本，没有新一代（实测 /weather/v1/indices/* 与
// /v1/indices/* 均 404）。
//
// days 只支持 1 与 3 —— 实测 2d / 7d / 10d / 15d / 30d 全部 404。
// type 取 0~16，0 表示全部。
//
// loc 必须是坐标对：上游对裸经度（如 "121.47"）一律回 400。
//
// 缓存键：`qwx:indices:<location>:t<type>:d<days>` 加语言后缀。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [Endpoint] 覆盖。
func (c *Client) Indices(ctx context.Context, loc qwx.Location, typ, days int, lang string) (*Result, error) {
	if typ < TypeAll || typ > MaxType {
		return nil, fmt.Errorf("%w: indices type %d out of range [0,%d]", qwx.ErrInvalidParameter, typ, MaxType)
	}
	if days != 1 && days != 3 {
		return nil, fmt.Errorf("%w: indices days %d is not supported (only 1 and 3)", qwx.ErrInvalidParameter, days)
	}
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: indices requires coordinates", qwx.ErrInvalidLocation)
	}
	q := url.Values{}
	q.Set("location", loc.Query())
	q.Set("type", strconv.Itoa(typ))
	if lang != "" {
		q.Set("lang", lang)
	}

	var out Result
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/indices/" + strconv.Itoa(days) + "d",
		Query:    q,
		Endpoint: Endpoint,
		CacheKey: "qwx:indices:" + loc.Query() + ":t" + strconv.Itoa(typ) + ":d" + strconv.Itoa(days) + cachekey.Lang(lang),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
