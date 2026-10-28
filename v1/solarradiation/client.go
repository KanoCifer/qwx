// Package solarradiation 提供太阳辐射预报接口（/solarradiation/v1/*）。
//
// 这一代只认坐标：位置必须是经纬度路径参数，LocationID 会被上游拒绝。
// 这一族没有 lang 参数 —— 上游的太阳辐射预报不返回文字描述。
package solarradiation

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Client 是太阳辐射预报接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造太阳辐射预报接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("solarradiation: nil base client")
	}
	return &Client{base: base}
}

// Forecast 查询坐标点上的太阳辐射预报。
//
// 调 GET /solarradiation/v1/forecast/{latitude}/{longitude}?hours=N。
//
// loc 必须带坐标：这一代只认经纬度路径参数，**纬度在前**。LocationID
// 塞进路径上游直接 400。用 LocationID 时先用 geo 包的 LookupCity 换坐标。
//
// hours 取值 1~72，越界返回 [qwx.ErrInvalidParameter]，不打上游。
// 上游的默认是 24 小时，但本方法要求显式给 hours —— 0 同样按越界处理，
// 不替调用方套用上游默认值。
//
// 上限以官方文档页为准：页面写「最多支持未来72小时预报」，参数表写
// 「可选1-72」。OpenAPI 的 schema 写 maximum: 60，是整份 spec 里唯一的
// maximum: 60，与同一条参数自己的 description 也矛盾。
//
// opts 的取值范围：
//
//   - Interval 只能是 15、30、60；0 表示用上游默认（60 分钟）。
//   - Tilt 0~90、Azimuth 0~359，都是指针：nil 表示不传，0 是合法值。
//   - Extra 只能是 "weather"、"poa" 或两者的组合（逗号分隔）。
//   - Extra 含 "poa" 时上游要求 tilt 与 azimuth 都传，缺任一返回
//     [qwx.ErrInvalidParameter]。
//
// 这一族没有 lang 参数：上游不返回文字描述，多传上游会 400。
//
// 时间默认 UTC，要当地时间设 [Options.LocalTime]。
//
// 缓存键：`qwx:solarradiation:v1:forecast:<lat>,<lon>:h<hours>` 加查询参数后缀。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [Endpoint] 覆盖。
func (c *Client) Forecast(ctx context.Context, loc qwx.Location, hours int, opts Options) (*Result, error) {
	if err := validateForecast(hours, opts); err != nil {
		return nil, err
	}
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	p := opts.params()
	q := p.Query()
	q.Set("hours", strconv.Itoa(hours))

	var out Result
	err = c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/solarradiation/v1/forecast/" + lat + "/" + lon,
		Query:    q,
		Endpoint: Endpoint,
		CacheKey: forecastCacheKey(loc, hours, p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// forecastCacheKey 太阳辐射预报的缓存键。
//
// 预报小时数进键：hours=24 与 hours=48 是两份不同的数据，共键会串。
func forecastCacheKey(loc qwx.Location, hours int, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilySolarRadiation).
		Path("forecast").
		PathCoord(loc.Lat, loc.Lon).
		Count("h", hours).
		Params(p).
		String()
}

// validateForecast 校验端点参数，越界返回 [qwx.ErrInvalidParameter]。
func validateForecast(hours int, o Options) error {
	if hours < minForecastHours || hours > MaxForecastHours {
		return fmt.Errorf("%w: hours %d out of range [%d,%d]", qwx.ErrInvalidParameter, hours, minForecastHours, MaxForecastHours)
	}
	if o.Interval != 0 && o.Interval != 15 && o.Interval != 30 && o.Interval != 60 {
		return fmt.Errorf("%w: interval %d not in {15,30,60}", qwx.ErrInvalidParameter, o.Interval)
	}
	if o.Tilt != nil && (*o.Tilt < 0 || *o.Tilt > 90) {
		return fmt.Errorf("%w: tilt %d out of range [0,90]", qwx.ErrInvalidParameter, *o.Tilt)
	}
	if o.Azimuth != nil && (*o.Azimuth < 0 || *o.Azimuth > 359) {
		return fmt.Errorf("%w: azimuth %d out of range [0,359]", qwx.ErrInvalidParameter, *o.Azimuth)
	}
	extras, err := parseExtra(o.Extra)
	if err != nil {
		return err
	}
	if extras["poa"] && (o.Tilt == nil || o.Azimuth == nil) {
		return fmt.Errorf("%w: extra=poa requires both tilt and azimuth", qwx.ErrInvalidParameter)
	}
	return nil
}

// parseExtra 解析 extra 参数，只认 "weather" 与 "poa"。
func parseExtra(raw string) (map[string]bool, error) {
	set := map[string]bool{}
	if raw == "" {
		return set, nil
	}
	for _, part := range strings.Split(raw, ",") {
		if part != "weather" && part != "poa" {
			return nil, fmt.Errorf("%w: extra %q contains unknown value %q", qwx.ErrInvalidParameter, raw, part)
		}
		if set[part] {
			return nil, fmt.Errorf("%w: extra %q contains duplicate value %q", qwx.ErrInvalidParameter, raw, part)
		}
		set[part] = true
	}
	return set, nil
}
