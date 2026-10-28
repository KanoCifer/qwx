// Package storm 提供旧一代台风数据接口（/v7/tropical/*）。
//
// 台风数据至今只有 v7 版本，没有新一代。
//
// 三个端点都不收位置参数：台风按流域与台风 ID 检索，与查询地点无关。
//
// 旧一代与新一代在字段单位、时间基准、错误体与缓存键上都不同，
// 本包不与 v1 下的包混用任何逻辑。
package storm

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Client 是台风数据接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造台风数据接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("storm: nil base client")
	}
	return &Client{base: base}
}

// List 查询台风列表。
//
// 调 GET /v7/tropical/storm-list?basin=NP&year=2026。
//
// 本端点不收位置参数：列表按流域与年份检索，与查询地点无关。
//
// basin 只接受 "NP"（西北太平洋）—— 上游明确说"目前仅支持中国沿海地区，
// 即 basin=NP"，别的流域代码上游不认，客户端在打上游之前就返回
// [qwx.ErrInvalidParameter]。year 必须大于 0，上游只保证支持本年度与
// 上一年度，更早的年份能不能查到由上游判断。
//
// 缓存键：`qwx:storm:list:NP:2026`。
//
// 默认 TTL 24 小时（台风列表一天内基本不变），可用 [qwx.WithTTLPolicy]
// 按 [EndpointList] 覆盖。
func (c *Client) List(ctx context.Context, basin string, year int) (*ListResult, error) {
	if basin != BasinNP {
		return nil, fmt.Errorf("%w: storm basin %q is not supported (only %q)", qwx.ErrInvalidParameter, basin, BasinNP)
	}
	if year <= 0 {
		return nil, fmt.Errorf("%w: storm year %d must be positive", qwx.ErrInvalidParameter, year)
	}
	q := url.Values{}
	q.Set("basin", basin)
	q.Set("year", strconv.Itoa(year))

	var out ListResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/tropical/storm-list",
		Query:    q,
		Endpoint: EndpointList,
		CacheKey: listCacheKey(basin, year),
		TTL:      listCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Forecast 查询台风预报。
//
// 调 GET /v7/tropical/storm-forecast?stormid=NP2018。
//
// 本端点不收位置参数：预报按台风 ID 检索，与查询地点无关。
//
// stormid 必填且不能为空串，空串返回 [qwx.ErrInvalidParameter]。
// ID 从 [Client.List] 的结果里取。
//
// 缓存键：`qwx:storm:forecast:NP2018`。
//
// 默认 TTL 1 小时（上游几小时更新一次），可用 [qwx.WithTTLPolicy]
// 按 [EndpointForecast] 覆盖。
//
// 上游怪癖：查询的台风已经结束时，返回的 forecast 列表是空的 —— 不是错误。
// 要先调 [Client.List] 拿台风状态，再决定要不要查预报。
func (c *Client) Forecast(ctx context.Context, stormID string) (*ForecastResult, error) {
	if stormID == "" {
		return nil, fmt.Errorf("%w: storm forecast requires a storm id", qwx.ErrInvalidParameter)
	}
	q := url.Values{}
	q.Set("stormid", stormID)

	var out ForecastResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/tropical/storm-forecast",
		Query:    q,
		Endpoint: EndpointForecast,
		CacheKey: forecastCacheKey(stormID),
		TTL:      forecastCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Track 查询台风实况和路径。
//
// 调 GET /v7/tropical/storm-track?stormid=NP2018。
//
// 本端点不收位置参数：实况与路径按台风 ID 检索，与查询地点无关。
//
// stormid 必填且不能为空串，空串返回 [qwx.ErrInvalidParameter]。
// ID 从 [Client.List] 的结果里取。
//
// 缓存键：`qwx:storm:track:NP2018`。
//
// 默认 TTL 1 小时（上游几小时更新一次），可用 [qwx.WithTTLPolicy]
// 按 [EndpointTrack] 覆盖。
//
// 上游怪癖：now 字段是当前实况，台风已结束时上游可能给 null，这时
// [TrackResult.Now] 是 nil。台风是否活跃看 isActive（"1" 活跃，"0" 停编）。
func (c *Client) Track(ctx context.Context, stormID string) (*TrackResult, error) {
	if stormID == "" {
		return nil, fmt.Errorf("%w: storm track requires a storm id", qwx.ErrInvalidParameter)
	}
	q := url.Values{}
	q.Set("stormid", stormID)

	var out TrackResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/tropical/storm-track",
		Query:    q,
		Endpoint: EndpointTrack,
		CacheKey: trackCacheKey(stormID),
		TTL:      trackCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// listCacheKey 台风列表的缓存键。
//
// 流域与年份都进键：不同流域或年份是不同数据，共键就是串味。
func listCacheKey(basin string, year int) string {
	return cachekey.New(cachekey.FamilyStorm).
		Path("list", basin).
		Path(strconv.Itoa(year)).
		String()
}

// forecastCacheKey 台风预报的缓存键。
//
// 台风 ID 进键：不同台风的预报是不同数据。
func forecastCacheKey(stormID string) string {
	return cachekey.New(cachekey.FamilyStorm).
		Path("forecast", stormID).
		String()
}

// trackCacheKey 台风实况和路径的缓存键。
//
// 台风 ID 进键：不同台风的实况是不同数据。
func trackCacheKey(stormID string) string {
	return cachekey.New(cachekey.FamilyStorm).
		Path("track", stormID).
		String()
}
