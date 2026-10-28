// Package airquality 提供新一代空气质量接口（/airquality/v1/*）。
//
// 这一代只认坐标：位置必须是经纬度路径参数，LocationID 会被上游拒绝。
package airquality

import (
	"context"
	"fmt"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
	"github.com/KanoCifer/qwx/internal/coord"
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
// 缓存键：`qwx:aq:v1:current:<lat>,<lon>` 加语言后缀。
//
// 默认 TTL 30 分钟，可用 [qwx.WithTTLPolicy] 按 [EndpointCurrent] 覆盖。
func (c *Client) Current(ctx context.Context, loc qwx.Location, opts Options) (*Current, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out Current
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/airquality/v1/current/" + lat + "/" + lon,
		Query:    opts.params(),
		Endpoint: EndpointCurrent,
		CacheKey: "qwx:aq:v1:current:" + coord.Format(loc.Lat) + "," + coord.Format(loc.Lon) + cachekey.Lang(opts.Lang),
		TTL:      currentCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Daily 查询逐日空气质量预报。
//
// 调 GET /airquality/v1/daily/{latitude}/{longitude}。
//
// 缓存键：`qwx:aq:v1:daily:<lat>,<lon>` 加语言后缀。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointDaily] 覆盖。
func (c *Client) Daily(ctx context.Context, loc qwx.Location, opts Options) (*DailyForecast, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out DailyForecast
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/airquality/v1/daily/" + lat + "/" + lon,
		Query:    opts.params(),
		Endpoint: EndpointDaily,
		CacheKey: "qwx:aq:v1:daily:" + coord.Format(loc.Lat) + "," + coord.Format(loc.Lon) + cachekey.Lang(opts.Lang),
		TTL:      forecastCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Hourly 查询逐小时空气质量预报。
//
// 调 GET /airquality/v1/hourly/{latitude}/{longitude}。
//
// 缓存键：`qwx:aq:v1:hourly:<lat>,<lon>` 加语言后缀。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointHourly] 覆盖。
func (c *Client) Hourly(ctx context.Context, loc qwx.Location, opts Options) (*HourlyForecast, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out HourlyForecast
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/airquality/v1/hourly/" + lat + "/" + lon,
		Query:    opts.params(),
		Endpoint: EndpointHourly,
		CacheKey: "qwx:aq:v1:hourly:" + coord.Format(loc.Lat) + "," + coord.Format(loc.Lon) + cachekey.Lang(opts.Lang),
		TTL:      forecastCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Station 查询单个监测站的污染物数据。
//
// 调 GET /airquality/v1/stations/{locationId}。
//
// stationID 是监测站 ID（如 "P51762"），来自 [Current] 的 Stations 列表。
//
// 缓存键：`qwx:aq:v1:station:<id>` 加语言后缀。
//
// 默认 TTL 30 分钟，可用 [qwx.WithTTLPolicy] 按 [EndpointStation] 覆盖。
func (c *Client) Station(ctx context.Context, stationID string, opts Options) (*StationData, error) {
	if stationID == "" {
		return nil, fmt.Errorf("%w: empty station ID", qwx.ErrInvalidParameter)
	}
	var out StationData
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/airquality/v1/stations/" + stationID,
		Query:    opts.params(),
		Endpoint: EndpointStation,
		CacheKey: "qwx:aq:v1:station:" + stationID + cachekey.Lang(opts.Lang),
		TTL:      currentCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
