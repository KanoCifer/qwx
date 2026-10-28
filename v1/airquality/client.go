// Package airquality 提供新一代空气质量接口（/airquality/v1/*）。
//
// 这一代只认坐标：位置必须是经纬度路径参数，LocationID 会被上游拒绝。
package airquality

import (
	"context"
	"fmt"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Client 是空气质量接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造空气质量接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("airquality: nil base client")
	}
	return &Client{base: base}
}

// Current 查询实时空气质量。
//
// 调 GET /airquality/v1/current/{latitude}/{longitude}。
//
// opts.Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
//
// 缓存键：`qwx:aq:v1:current:<lat>,<lon>` 加语言后缀。
//
// 默认 TTL 30 分钟，可用 [qwx.WithTTLPolicy] 按 [EndpointCurrent] 覆盖。
func (c *Client) Current(ctx context.Context, loc qwx.Location, opts Options) (*Current, error) {
	if err := opts.Lang.Validate(); err != nil {
		return nil, err
	}
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	p := opts.params()
	var out Current
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/airquality/v1/current/" + lat + "/" + lon,
		Query:    p.Query(),
		Endpoint: EndpointCurrent,
		CacheKey: currentCacheKey(loc, p),
		TTL:      currentCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// currentCacheKey 实时空气质量的缓存键。
func currentCacheKey(loc qwx.Location, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyAirQuality).
		Path("current").
		PathCoord(loc.Lat, loc.Lon).
		Params(p).
		String()
}

// Daily 查询逐日空气质量预报。
//
// 调 GET /airquality/v1/daily/{latitude}/{longitude}。
//
// opts.Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
//
// 缓存键：`qwx:aq:v1:daily:<lat>,<lon>` 加语言后缀。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointDaily] 覆盖。
func (c *Client) Daily(ctx context.Context, loc qwx.Location, opts Options) (*DailyForecast, error) {
	if err := opts.Lang.Validate(); err != nil {
		return nil, err
	}
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	p := opts.params()
	var out DailyForecast
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/airquality/v1/daily/" + lat + "/" + lon,
		Query:    p.Query(),
		Endpoint: EndpointDaily,
		CacheKey: dailyCacheKey(loc, p),
		TTL:      forecastCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// dailyCacheKey 逐日空气质量预报的缓存键。
func dailyCacheKey(loc qwx.Location, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyAirQuality).
		Path("daily").
		PathCoord(loc.Lat, loc.Lon).
		Params(p).
		String()
}

// Hourly 查询逐小时空气质量预报。
//
// 调 GET /airquality/v1/hourly/{latitude}/{longitude}。
//
// opts.Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
//
// 缓存键：`qwx:aq:v1:hourly:<lat>,<lon>` 加语言后缀。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointHourly] 覆盖。
func (c *Client) Hourly(ctx context.Context, loc qwx.Location, opts Options) (*HourlyForecast, error) {
	if err := opts.Lang.Validate(); err != nil {
		return nil, err
	}
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	p := opts.params()
	var out HourlyForecast
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/airquality/v1/hourly/" + lat + "/" + lon,
		Query:    p.Query(),
		Endpoint: EndpointHourly,
		CacheKey: hourlyCacheKey(loc, p),
		TTL:      forecastCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// hourlyCacheKey 逐小时空气质量预报的缓存键。
func hourlyCacheKey(loc qwx.Location, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyAirQuality).
		Path("hourly").
		PathCoord(loc.Lat, loc.Lon).
		Params(p).
		String()
}

// Station 查询单个监测站的污染物数据。
//
// 调 GET /airquality/v1/stations/{locationId}。
//
// stationID 是监测站 ID（如 "P51762"），来自 [Current] 的 Stations 列表。
//
// opts.Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
//
// 官方文档没有这个端点的独立页面，lang 的支持无从核实 —— 上游与其他
// 空气质量端点同源，接受 lang。
//
// 缓存键：`qwx:aq:v1:station:<id>` 加语言后缀。
//
// 默认 TTL 30 分钟，可用 [qwx.WithTTLPolicy] 按 [EndpointStation] 覆盖。
func (c *Client) Station(ctx context.Context, stationID string, opts Options) (*StationData, error) {
	if stationID == "" {
		return nil, fmt.Errorf("%w: empty station ID", qwx.ErrInvalidParameter)
	}
	if err := opts.Lang.Validate(); err != nil {
		return nil, err
	}
	p := opts.params()
	var out StationData
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/airquality/v1/stations/" + stationID,
		Query:    p.Query(),
		Endpoint: EndpointStation,
		CacheKey: stationCacheKey(stationID, p),
		TTL:      currentCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// stationCacheKey 单个监测站的缓存键。
func stationCacheKey(stationID string, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyAirQuality).
		Path("station", stationID).
		Params(p).
		String()
}
