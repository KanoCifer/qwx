// Package quick 提供组合式的快速天气查询。
//
//	report, err := quick.New(base).Query(ctx, "北京", quick.Options{
//		Lang:      qwx.LangZH,
//		LocalTime: true,
//	})
//
// 位置可以是经纬度字符串（"116.41,39.92"，经度在前）、地名（"北京"）、
// LocationID，也可以直接给 [qwx.Location]。
package quick

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/geo"
	"github.com/KanoCifer/qwx/v1/airquality"
	"github.com/KanoCifer/qwx/v1/alert"
	"github.com/KanoCifer/qwx/v1/weather"
)

// Place 是地名解析出来的地点。
//
// 它是 [geo.Place] 的别名。
type Place = geo.Place

// CurrentWeather 是实时天气。
//
// 它是 [weather.CurrentWeather] 的别名。
type CurrentWeather = weather.CurrentWeather

// HourlyForecast 是逐小时预报。
//
// 它是 [weather.HourlyForecast] 的别名。
type HourlyForecast = weather.HourlyForecast

// DailyForecast 是逐日预报。
//
// 它是 [weather.DailyForecast] 的别名。
type DailyForecast = weather.DailyForecast

// AirQuality 是实时空气质量。
//
// 它是 [airquality.Current] 的别名。
type AirQuality = airquality.Current

// AlertResult 是预警查询的结果。
//
// 它是 [alert.Result] 的别名。
type AlertResult = alert.Result

// Client 是快速天气查询的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	geo        *geo.Client
	weather    *weather.Client
	airquality *airquality.Client
	alert      *alert.Client
}

// New 用基础客户端构造快速查询的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("quick: nil base client")
	}
	return &Client{
		geo:        geo.New(base),
		weather:    weather.New(base),
		airquality: airquality.New(base),
		alert:      alert.New(base),
	}
}

// Query 按位置查一份完整的天气信息。
//
// loc 是位置，支持三种写法：
//
//   - 字符串：经纬度对（"116.41,39.92"，**经度在前**）、地名（"北京"）、
//     LocationID（"101010100"）。经纬度与 LocationID 用 [qwx.ParseLocation]
//     识别，识别不了的一律当地名处理。
//   - [qwx.Location]：带坐标直接用，带 ID 或 Name 先查 GeoAPI 换坐标。
//   - *[qwx.Location]：同 [qwx.Location]。
//
// 位置是地名或 LocationID 时，先调 GeoAPI 的城市检索换坐标 —— 新一代接口
// 只认坐标。解析出的地点取第一条，重名城市用 [Options.Adm] 或
// [Options.Range] 限定。
//
// 五块数据并发取，缓存与 TTL 各自按端点常量走。用 [Options.Skip] 跳过
// 不需要的块。
//
// 默认宽松：某块失败记进 [Report.Errors]，其余照常返回；全部失败才返回
// 错误。开了 [Options.Strict] 则任一块失败即返回错误。
func (c *Client) Query(ctx context.Context, loc any, opts Options) (*Report, error) {
	parts, err := opts.parts()
	if err != nil {
		return nil, err
	}
	if err := opts.Lang.Validate(); err != nil {
		return nil, err
	}
	hours, days, err := opts.counts()
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: no part requested", qwx.ErrInvalidParameter)
	}

	resolved, place, err := c.resolve(ctx, loc, opts)
	if err != nil {
		return nil, err
	}

	results := c.fetchAll(ctx, parts, resolved, hours, days, opts)

	report := &Report{Location: resolved, Place: place}
	for _, r := range results {
		if r.err != nil {
			report.Errors = append(report.Errors, EndpointError{Part: r.part, Endpoint: r.endpoint, Err: r.err})
			continue
		}
		switch r.part {
		case PartCurrent:
			report.Current, _ = r.value.(*CurrentWeather)
		case PartHourly:
			report.Hourly, _ = r.value.(*HourlyForecast)
		case PartDaily:
			report.Daily, _ = r.value.(*DailyForecast)
		case PartAirQuality:
			report.AirQuality, _ = r.value.(*AirQuality)
		case PartAlerts:
			report.Alerts, _ = r.value.(*AlertResult)
		}
	}

	if len(report.Errors) == len(parts) {
		// 一块都没拿到，报告没有意义，直接返回错误。
		return nil, report.Errors[0]
	}
	if opts.Strict && len(report.Errors) > 0 {
		return nil, report.Errors[0]
	}
	return report, nil
}

// partResult 是某一块数据的取回结果。
type partResult struct {
	part     Part
	endpoint qwx.Endpoint
	value    any
	err      error
}

// fetchAll 并发取回各块数据。
//
// 上游请求彼此独立，并发能把总耗时压到最慢的那一个。基础客户端无状态，
// 可以并发共享。
func (c *Client) fetchAll(ctx context.Context, parts []Part, loc qwx.Location, hours, days int, opts Options) []partResult {
	results := make([]partResult, len(parts))
	var wg sync.WaitGroup
	for i, p := range parts {
		wg.Add(1)
		go func(i int, p Part) {
			defer wg.Done()
			v, ep, err := c.fetchPart(ctx, p, loc, hours, days, opts)
			results[i] = partResult{part: p, endpoint: ep, value: v, err: err}
		}(i, p)
	}
	wg.Wait()
	return results
}

// fetchPart 取回一块数据。
func (c *Client) fetchPart(ctx context.Context, p Part, loc qwx.Location, hours, days int, opts Options) (any, qwx.Endpoint, error) {
	weatherOpts := weather.Options{Lang: opts.Lang, LocalTime: opts.LocalTime}
	switch p {
	case PartCurrent:
		v, err := c.weather.Current(ctx, loc, weatherOpts)
		if err != nil {
			return nil, weather.EndpointCurrent, err
		}
		return v, weather.EndpointCurrent, nil
	case PartHourly:
		v, err := c.weather.Hourly(ctx, loc, hours, weatherOpts)
		if err != nil {
			return nil, weather.EndpointHourly, err
		}
		return v, weather.EndpointHourly, nil
	case PartDaily:
		v, err := c.weather.Daily(ctx, loc, days, weatherOpts)
		if err != nil {
			return nil, weather.EndpointDaily, err
		}
		return v, weather.EndpointDaily, nil
	case PartAirQuality:
		v, err := c.airquality.Current(ctx, loc, airquality.Options{Lang: opts.Lang})
		if err != nil {
			return nil, airquality.EndpointCurrent, err
		}
		return v, airquality.EndpointCurrent, nil
	case PartAlerts:
		v, err := c.alert.Current(ctx, loc, alert.Options{Lang: opts.Lang, LocalTime: opts.LocalTime})
		if err != nil {
			return nil, alert.Endpoint, err
		}
		return v, alert.Endpoint, nil
	default:
		return nil, "", fmt.Errorf("%w: unknown part %q", qwx.ErrInvalidParameter, p)
	}
}

// resolve 把位置解析成坐标。
//
// 返回解析后的坐标与地名解析结果（直接传坐标时后者为 nil）。
func (c *Client) resolve(ctx context.Context, loc any, opts Options) (qwx.Location, *Place, error) {
	switch v := loc.(type) {
	case string:
		return c.resolveString(ctx, v, opts)
	case qwx.Location:
		return c.resolveLocation(ctx, v, opts)
	case *qwx.Location:
		if v == nil {
			return qwx.Location{}, nil, fmt.Errorf("%w: nil location", qwx.ErrInvalidLocation)
		}
		return c.resolveLocation(ctx, *v, opts)
	case nil:
		return qwx.Location{}, nil, fmt.Errorf("%w: nil location", qwx.ErrInvalidLocation)
	default:
		return qwx.Location{}, nil, fmt.Errorf("%w: unsupported location type %T", qwx.ErrInvalidLocation, loc)
	}
}

// resolveString 解析字符串位置。
//
// 先按坐标对与 LocationID 试，识别不了的一律当地名。
func (c *Client) resolveString(ctx context.Context, raw string, opts Options) (qwx.Location, *Place, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return qwx.Location{}, nil, fmt.Errorf("%w: empty location", qwx.ErrInvalidLocation)
	}
	parsed, err := qwx.ParseLocation(s)
	if err == nil {
		if parsed.HasCoordinates() {
			return parsed, nil, nil
		}
		// LocationID：新一代只认坐标，先反查。
		return c.lookup(ctx, qwx.Location{ID: parsed.ID}, opts)
	}
	// 识别不了坐标对与 LocationID，就是地名。
	return c.lookup(ctx, qwx.Location{Name: s}, opts)
}

// resolveLocation 解析 [qwx.Location]。
func (c *Client) resolveLocation(ctx context.Context, loc qwx.Location, opts Options) (qwx.Location, *Place, error) {
	if loc.HasCoordinates() {
		return loc, nil, nil
	}
	switch {
	case loc.ID != "":
		return c.lookup(ctx, qwx.Location{ID: loc.ID}, opts)
	case loc.Name != "":
		return c.lookup(ctx, qwx.Location{Name: loc.Name}, opts)
	default:
		return qwx.Location{}, nil, fmt.Errorf("%w: location has no coordinates, ID or name", qwx.ErrInvalidLocation)
	}
}

// lookup 用 GeoAPI 的城市检索把地名或 LocationID 换成坐标。
//
// 取第一条结果。地名有重名时用 [Options.Adm] 或 [Options.Range] 限定。
func (c *Client) lookup(ctx context.Context, query qwx.Location, opts Options) (qwx.Location, *Place, error) {
	res, err := c.geo.LookupCity(ctx, query, geo.Options{
		Lang:   opts.Lang,
		Number: opts.Number,
		Adm:    opts.Adm,
		Range:  opts.Range,
	})
	if err != nil {
		return qwx.Location{}, nil, err
	}
	if len(res.Locations) == 0 {
		return qwx.Location{}, nil, fmt.Errorf("%w: no place matched %q", qwx.ErrInvalidLocation, query.Query())
	}
	place := res.Locations[0]
	lat, lon, err := place.Coordinates()
	if err != nil {
		return qwx.Location{}, nil, err
	}
	return qwx.Location{Lat: lat, Lon: lon}, &place, nil
}
