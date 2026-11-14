package quick

import (
	"fmt"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/weather"
)

// Part 标记报告里的一块数据。
//
// 用它跳过不关心的部分：一次快速查询默认取齐五块，账号没开通某项权限时
// 也可以直接跳过。
type Part string

// 报告包含的数据块。
const (
	// PartCurrent 是实时天气。
	PartCurrent Part = "current"
	// PartHourly 是逐小时预报。
	PartHourly Part = "hourly"
	// PartDaily 是逐日预报。
	PartDaily Part = "daily"
	// PartAirQuality 是实时空气质量。
	PartAirQuality Part = "airquality"
	// PartAlerts 是预警。
	PartAlerts Part = "alerts"
)

// 快速查询的默认预报范围。
const (
	// DefaultHours 是逐小时预报的默认小时数。
	DefaultHours = 24
	// DefaultDays 是逐日预报的默认天数。
	DefaultDays = 3
)

// Options 是快速查询的可选参数。
//
// 它把各端点各自的参数收在一处：语言对五块数据都生效，当地时间只对
// 天气与预警生效（上游的空气质量没有当地时间参数），逐时逐日的条数只对
// 预报生效，行政区划与搜索范围只对地名解析生效。
type Options struct {
	// Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
	Lang qwx.Lang
	// LocalTime 是否返回查询地点的当地时间。
	//
	// 新一代接口默认给 UTC，不传本地时间会整整差 8 小时。
	//
	// 只对天气与预警生效：上游的空气质量接口没有当地时间参数，
	// [Report.AirQuality] 的时间始终是 UTC。
	LocalTime bool
	// Hours 是逐小时预报的小时数。0 用 [DefaultHours]。
	//
	// 取值 1~240，超出范围返回 [qwx.ErrInvalidParameter]，不打上游。
	Hours int
	// Days 是逐日预报的天数。0 用 [DefaultDays]。
	//
	// 取值 1~10，超出范围返回 [qwx.ErrInvalidParameter]，不打上游。
	Days int
	// Number 是城市检索返回条数上限。0 用上游默认（10）。
	//
	// 只在位置是地名或 LocationID 时起作用。
	Number int
	// Adm 是上级行政区划的 LocationID，如 "101010100"。
	//
	// 用于排除重名城市。只在位置是地名或 LocationID 时起作用。
	Adm string
	// Range 是城市检索的搜索范围，用 ISO 3166 国家代码，如 "cn"、"world"。
	//
	// 留空时上游在所有城市里搜，"world" 才能查到境外城市。
	// 只在位置是地名或 LocationID 时起作用。
	Range string
	// Skip 是要跳过的数据块。留空表示五块全取。
	//
	// 写未知的数据块返回 [qwx.ErrInvalidParameter]，不打上游。
	Skip []Part
	// Strict 是否遇到任何一块失败就整体失败。
	//
	// 默认 false：失败的那块记进 [Report.Errors]，其余照常返回 —— 账号没
	// 开通空气质量权限时，预警和天气仍然拿得到。开了之后第一块失败就返回
	// 错误，报告拿不到。
	Strict bool
}

// parts 是本次要取的数据块。
func (o Options) parts() ([]Part, error) {
	skip := make(map[Part]bool, len(o.Skip))
	for _, p := range o.Skip {
		switch p {
		case PartCurrent, PartHourly, PartDaily, PartAirQuality, PartAlerts:
			skip[p] = true
		default:
			return nil, fmt.Errorf("%w: unknown part %q", qwx.ErrInvalidParameter, p)
		}
	}
	all := []Part{PartCurrent, PartHourly, PartDaily, PartAirQuality, PartAlerts}
	out := make([]Part, 0, len(all))
	for _, p := range all {
		if !skip[p] {
			out = append(out, p)
		}
	}
	return out, nil
}

// counts 是逐时逐日的条数，0 用默认值。
func (o Options) counts() (hours, days int, err error) {
	hours = o.Hours
	if hours == 0 {
		hours = DefaultHours
	}
	if hours < 1 || hours > weather.MaxHourlyHours {
		return 0, 0, fmt.Errorf("%w: hours %d out of range [1,%d]", qwx.ErrInvalidParameter, hours, weather.MaxHourlyHours)
	}
	days = o.Days
	if days == 0 {
		days = DefaultDays
	}
	if days < 1 || days > weather.MaxDailyDays {
		return 0, 0, fmt.Errorf("%w: days %d out of range [1,%d]", qwx.ErrInvalidParameter, days, weather.MaxDailyDays)
	}
	return hours, days, nil
}

// Report 是一次快速查询的结果。
//
// 取到的每一块各自独立：某块失败不影响其他块，失败原因在 [Report.Errors]。
// 没取到的块对应字段为 nil。
type Report struct {
	// Location 是最终查询用的坐标位置。
	//
	// 传坐标时就是它本身；传地名或 LocationID 时是解析出来的坐标。
	Location qwx.Location
	// Place 是地名解析的结果。传坐标时为 nil。
	Place *Place
	// Current 是实时天气。没取到时为 nil。
	Current *CurrentWeather
	// Hourly 是逐小时预报。没取到时为 nil。
	Hourly *HourlyForecast
	// Daily 是逐日预报。没取到时为 nil。
	Daily *DailyForecast
	// AirQuality 是实时空气质量。没取到时为 nil。
	AirQuality *AirQuality
	// Alerts 是预警。没取到时为 nil。
	//
	// 查得到但没有预警时是空列表，不是 nil。
	Alerts *AlertResult
	// Errors 是失败的数据块。
	//
	// 严格模式（[Options.Strict]）下它恒为空 —— 失败会直接返回错误。
	Errors []EndpointError
}

// EndpointError 是某一块数据的失败原因。
type EndpointError struct {
	// Part 是失败的数据块。
	Part Part
	// Endpoint 是失败的上游端点标识，如 [qwx.Endpoint]。
	Endpoint qwx.Endpoint
	// Err 是原始错误。
	//
	// 用 errors.Is 判断类别，如 [qwx.ErrUnauthorized]。
	Err error
}

// Error 实现 error 接口。
func (e EndpointError) Error() string {
	return fmt.Sprintf("quick: %s endpoint %q failed: %v", e.Part, e.Endpoint, e.Err)
}

// Unwrap 让 errors.Is 能识别原始错误的类别。
func (e EndpointError) Unwrap() error { return e.Err }
