// Package historical 提供旧一代历史天气接口（/v7/historical/weather）。
//
// 历史天气（上游叫「天气时光机」）至今只有 v7 版本。
//
// 旧一代与新一代在位置形态、字段单位、时间基准、错误体与缓存键上都不同，
// 本包不与 v1 下的包混用任何逻辑。
package historical

import (
	"context"
	"fmt"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/baseclient"
	"github.com/KanoCifer/qwx/internal/cachekey"
	"github.com/KanoCifer/qwx/internal/param"
	"github.com/KanoCifer/qwx/internal/upstream"
)

// Client 是历史天气接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造历史天气接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	baseclient.Require(base, "historical")
	return &Client{base: base}
}

// Weather 查询某一天的历史天气。
//
// 调 GET /v7/historical/weather?location=...&date=yyyyMMdd&lang=...
//
// loc **只认 LocationID** —— 上游这个参数只接受 LocationID，坐标对与地名
// 都会被拒。LocationID 用 geo.LookupCity 查。
//
// date 必填，格式 "yyyyMMdd"。客户端只校验格式与日历合法性，日期范围交给
// 上游判断：上游只给最近 10 天、不含今天的数据，超范围回 400。
//
// opts.Lang 是语言，如 [qwx.LangZH]。留空用上游默认。月相名称与逐小时天气
// 描述随语言变，所以 lang 进缓存键。
//
// 上游还有 unit 参数（"m" 公制 / "i" 英制），本包不暴露 —— 全库都用公制。
//
// 缓存键：`qwx:historical:weather:<locationID>:<date>` 加语言后缀。
//
// 默认 TTL 24 小时 —— 某一天的数据过去之后就不再变，可用 [qwx.WithTTLPolicy]
// 按 [Endpoint] 覆盖。
func (c *Client) Weather(ctx context.Context, loc qwx.Location, date string, opts Options) (*Result, error) {
	if loc.ID == "" {
		return nil, fmt.Errorf("%w: historical weather requires a LocationID", qwx.ErrInvalidLocation)
	}
	if err := param.ValidateDate(date); err != nil {
		return nil, err
	}
	if err := opts.Lang.Validate(); err != nil {
		return nil, err
	}
	p := cachekey.Params{Lang: opts.Lang.API}
	q := p.Query()
	q.Set(upstream.Location, loc.ID)
	q.Set(upstream.Date, date)

	var out Result
	err := c.base.Do(ctx, qwx.Request{
		Method:   qwx.MethodGet,
		Path:     "/v7/historical/weather",
		Query:    q,
		Endpoint: Endpoint,
		CacheKey: weatherCacheKey(loc.ID, date, p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// weatherCacheKey 历史天气的缓存键。
//
// 地点与日期都进键：换一个地点或换一个日期是另一份数据。
func weatherCacheKey(id, date string, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyHistorical).
		Path("weather", id, date).
		Params(p).
		String()
}
