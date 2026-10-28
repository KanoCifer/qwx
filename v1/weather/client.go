// Package weather 提供新一代天气预报接口（/weather/v1/*）。
//
// 这一代只认坐标：位置必须是经纬度路径参数，LocationID 会被上游拒绝。
package weather

import (
	"context"
	"fmt"
	"strconv"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/coord"
)

// Client 是天气接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造天气接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("weather: nil base client")
	}
	return &Client{base: base}
}

// Current 查询实时天气。
//
// 调 GET /weather/v1/current/{latitude}/{longitude}。
//
// loc 必须带坐标：新一代接口只认经纬度路径参数，LocationID 塞进路径
// 上游直接 404（body 为空）。用 LocationID 时先用 geo 包的 LookupCity
// 换成坐标。
//
// 缓存键：`qwx:v1:current:<lat>,<lon>` 加查询参数后缀。
//
// 默认 TTL 10 分钟，可用 [qwx.WithTTLPolicy] 按 [EndpointCurrent] 覆盖。
func (c *Client) Current(ctx context.Context, loc qwx.Location, opts Options) (*CurrentWeather, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out CurrentWeather
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/weather/v1/current/" + lat + "/" + lon,
		Query:    opts.params(),
		Endpoint: EndpointCurrent,
		CacheKey: c.currentCacheKey(loc, opts),
		TTL:      currentCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// currentCacheKey 实时天气的缓存键。
func (c *Client) currentCacheKey(loc qwx.Location, opts Options) string {
	return "qwx:v1:current:" + coord.Format(loc.Lat) + "," + coord.Format(loc.Lon) + opts.cacheSuffix()
}

// Hourly 查询逐小时预报。
//
// 调 GET /weather/v1/hourly/{latitude}/{longitude}?hours=N。
//
// hours 取值 1~240，超出范围返回 [qwx.ErrInvalidParameter]，不打上游。
//
// 缓存键：`qwx:v1:hourly:<lat>,<lon>:h<hours>` 加查询参数后缀。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointHourly] 覆盖。
func (c *Client) Hourly(ctx context.Context, loc qwx.Location, hours int, opts Options) (*HourlyForecast, error) {
	if hours < 1 || hours > MaxHourlyHours {
		return nil, fmt.Errorf("%w: hours %d out of range [1,%d]", qwx.ErrInvalidParameter, hours, MaxHourlyHours)
	}
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	q := opts.params()
	q.Set("hours", strconv.Itoa(hours))

	var out HourlyForecast
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/weather/v1/hourly/" + lat + "/" + lon,
		Query:    q,
		Endpoint: EndpointHourly,
		CacheKey: c.hourlyCacheKey(loc, hours, opts),
		TTL:      forecastCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// hourlyCacheKey 逐小时预报的缓存键。
//
// 预报小时数进键：hours=24 与 hours=48 是两份不同的数据，共键会串。
func (c *Client) hourlyCacheKey(loc qwx.Location, hours int, opts Options) string {
	return "qwx:v1:hourly:" + coord.Format(loc.Lat) + "," + coord.Format(loc.Lon) +
		":h" + strconv.Itoa(hours) + opts.cacheSuffix()
}

// Daily 查询逐日预报。
//
// 调 GET /weather/v1/daily/{latitude}/{longitude}?days=N。
//
// days 取值 1~10，超出范围返回 [qwx.ErrInvalidParameter]，不打上游。
//
// 缓存键：`qwx:v1:daily:<lat>,<lon>:d<days>` 加查询参数后缀。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointDaily] 覆盖。
func (c *Client) Daily(ctx context.Context, loc qwx.Location, days int, opts Options) (*DailyForecast, error) {
	if days < 1 || days > MaxDailyDays {
		return nil, fmt.Errorf("%w: days %d out of range [1,%d]", qwx.ErrInvalidParameter, days, MaxDailyDays)
	}
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	q := opts.params()
	q.Set("days", strconv.Itoa(days))

	var out DailyForecast
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/weather/v1/daily/" + lat + "/" + lon,
		Query:    q,
		Endpoint: EndpointDaily,
		CacheKey: c.dailyCacheKey(loc, days, opts),
		TTL:      forecastCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// dailyCacheKey 逐日预报的缓存键。
func (c *Client) dailyCacheKey(loc qwx.Location, days int, opts Options) string {
	return "qwx:v1:daily:" + coord.Format(loc.Lat) + "," + coord.Format(loc.Lon) +
		":d" + strconv.Itoa(days) + opts.cacheSuffix()
}
