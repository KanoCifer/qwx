package solarradiation

import (
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
	"github.com/KanoCifer/qwx/v1"
)

// 太阳辐射预报的参数上限（官方文档口径）。
const (
	// minForecastHours 预报小时数下限。
	minForecastHours = 1
	// MaxForecastHours 预报小时数上限。
	//
	// 以官方文档页为准：页面写「最多支持未来72小时预报」，参数表也写
	// 「可选1-72」。同一份 OpenAPI 的 schema 写 maximum: 60 —— 那是整份
	// spec 里唯一的 maximum: 60，与同一条参数自己的 description 也自相矛盾。
	MaxForecastHours = 72
)

// Endpoint 是太阳辐射预报端点，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const Endpoint = qwx.Endpoint("solarradiation/v1/forecast")

// 太阳辐射预报接口的默认缓存时长：上游逐 15 分钟更新一次预报。
const cacheTTL = time.Hour

// Options 是太阳辐射预报接口的可选参数。
type Options struct {
	// LocalTime 是否返回查询地点的当地时间。
	//
	// 新一代接口默认给 UTC，不传 localTime 时间会整整差 8 小时。
	LocalTime bool
	// Interval 是预报数据的时间间隔分钟数，只能是 15、30、60。
	//
	// 0 表示不传，用上游默认（60 分钟）。
	Interval int
	// Tilt 是光伏系统的倾斜角，0~90。
	//
	// nil 表示不传。用指针区分「没有这个参数」与「倾斜角为 0」——
	// 平铺的光伏板倾斜角就是 0，0 是合法取值。
	Tilt *int
	// Azimuth 是光伏系统的方位角，0~359，0 表示正北。
	//
	// nil 表示不传，0 是合法取值。
	Azimuth *int
	// Extra 是额外信息，如 "weather"、"poa"，多项用逗号分隔。
	//
	// Extra 含 "poa" 时上游要求 tilt 与 azimuth 都传，缺一个会被拒绝。
	// 取值原样进缓存键，所以 "poa,weather" 与 "weather,poa" 是两个键。
	Extra string
}

// params 是影响结果的查询参数。
//
// 同一份参数既生成上游查询串，也生成缓存键后缀 —— 新增参数时不会漏进键。
func (o Options) params() cachekey.Params {
	return cachekey.Params{
		LocalTime: o.LocalTime,
		Interval:  o.Interval,
		Tilt:      o.Tilt,
		Azimuth:   o.Azimuth,
		Extra:     o.Extra,
	}
}

// Result 是太阳辐射预报查询的结果。
type Result struct {
	// Metadata 是元数据。
	Metadata v1.Metadata `json:"metadata"`
	// Forecasts 是太阳辐照预报数据列表。
	//
	// 条数由请求的 hours 与 interval 决定，但上游可能给得更少 ——
	// 不要假设长度。
	Forecasts []Forecast `json:"forecasts"`
}

// Forecast 是一个时刻的太阳辐射预报。
type Forecast struct {
	// ForecastTime 是预报时刻。
	ForecastTime string `json:"forecastTime"`
	// SolarAngle 是太阳角度。
	SolarAngle SolarAngle `json:"solarAngle"`
	// DNI 是法向直接辐照。
	DNI v1.Value `json:"dni"`
	// DHI 是散射水平面辐照。
	DHI v1.Value `json:"dhi"`
	// GHI 是总水平面辐照。
	GHI v1.Value `json:"ghi"`
	// Weather 是基本天气数据。
	//
	// 上游只在请求了 extra=weather 时返回这个对象，其他情况为 nil。
	Weather *Weather `json:"weather,omitempty"`
	// POA 是阵列平面辐照度。
	//
	// 上游只在请求了 extra=poa 时返回这个对象，其他情况为 nil。
	POA *POA `json:"poa,omitempty"`
}

// SolarAngle 是太阳角度。
type SolarAngle struct {
	// Azimuth 是太阳方位角，0~359，正北为 0 度。
	Azimuth float64 `json:"azimuth"`
	// Elevation 是太阳高度角。
	Elevation float64 `json:"elevation"`
}

// Weather 是太阳辐射预报附带的基本天气数据。
type Weather struct {
	// Temperature 是温度。
	Temperature v1.Value `json:"temperature"`
	// WindSpeed 是风速。
	WindSpeed v1.Value `json:"windSpeed"`
	// Humidity 是相对湿度，单位是**百分数**（0~100）。
	//
	// 与新一代天气端点不同：那边给的是 0~1 的比例，这边给的是百分数。
	Humidity float64 `json:"humidity"`
}

// POA 是阵列平面辐照度（Plane of Array）。
type POA struct {
	// Global 是阵列平面总辐照度。
	Global v1.Value `json:"global"`
	// Direct 是阵列平面直接辐照度。
	Direct v1.Value `json:"direct"`
	// Diffuse 是阵列平面散射辐照度。
	Diffuse v1.Value `json:"diffuse"`
	// Reflected 是阵列平面反射辐照度。
	Reflected v1.Value `json:"reflected"`
}
