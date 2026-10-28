package astronomy

import (
	"time"

	"github.com/KanoCifer/qwx"
)

// 天文端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
//
// 常量的值发布后不再变更：策略按值匹配，改值会让用户的 TTL 配置静默失效。
const (
	// EndpointSun 是日出日落端点。
	EndpointSun = qwx.Endpoint("v7/astronomy/sun")
	// EndpointMoon 是月升月落和月相端点。
	EndpointMoon = qwx.Endpoint("v7/astronomy/moon")
	// EndpointAngle 是太阳高度角端点。
	EndpointAngle = qwx.Endpoint("v7/astronomy/solar-elevation-angle")
)

// 天文接口的默认缓存时长：这些数据按日期固定，一天之内不变。
const cacheTTL = 24 * time.Hour

// Options 是月相端点的可选参数。
//
// 只有月相端点认语言参数 —— 日出日落与太阳高度角不带语言。
type Options struct {
	// Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
	Lang qwx.Lang
}

// SunResult 是日出日落数据。
type SunResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// UpdateTime 是数据更新时间。
	UpdateTime string `json:"updateTime"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// Sunrise 是日出时间。
	//
	// 上游在高纬度地区可能给 null，用指针区分"没有日出"与"空串"。
	Sunrise *string `json:"sunrise"`
	// Sunset 是日落时间。
	//
	// 同 [SunResult.Sunrise]：极昼极夜地区可能为 null。
	Sunset *string `json:"sunset"`
	// Refer 是数据来源与许可信息。
	//
	// 上游在结果里带 refer 对象，展示这些数据时需要一并展示。
	Refer qwx.Refer `json:"refer"`
}

// MoonResult 是月升月落和月相数据。
type MoonResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// UpdateTime 是数据更新时间。
	UpdateTime string `json:"updateTime"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// Moonrise 是当天月升时间。
	//
	// 上游说可能为空，用指针区分"没有月升"与"空串"。
	Moonrise *string `json:"moonrise"`
	// Moonset 是当天月落时间。
	//
	// 同 [MoonResult.Moonrise]：可能为空。
	Moonset *string `json:"moonset"`
	// MoonPhase 是逐小时月相。
	//
	// 列表长度不固定，不假设。
	MoonPhase []MoonPhase `json:"moonPhase"`
	// Refer 是数据来源与许可信息。
	//
	// 上游在结果里带 refer 对象，展示这些数据时需要一并展示。
	Refer qwx.Refer `json:"refer"`
}

// MoonPhase 是一小时的月相。
type MoonPhase struct {
	// FXTime 是月相预报时间。
	FXTime string `json:"fxTime"`
	// Value 是月相数值。
	Value string `json:"value"`
	// Name 是月相名称，如 "新月"、"满月"。
	//
	// 名称随语言变，上游的取值也不止一种，这里只透传不写完备列表。
	Name string `json:"name"`
	// Illumination 是月亮照明度，百分比数值。
	//
	// 上游给的是字符串，这里保持字符串 —— 转成 float64 会丢掉它的表达精度。
	Illumination string `json:"illumination"`
	// Icon 是月相图标代码。
	Icon string `json:"icon"`
}

// AngleResult 是太阳高度角数据。
//
// 上游这个端点的响应里没有 updateTime 与 fxLink，本类型也没有这两个字段。
type AngleResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// SolarElevationAngle 是太阳高度角（度）。
	//
	// 上游给的是字符串，这里保持字符串 —— 转成 float64 会丢掉它的表达精度。
	SolarElevationAngle string `json:"solarElevationAngle"`
	// SolarAzimuthAngle 是太阳方位角（度），0 度为正北。
	//
	// 上游给的是字符串，这里保持字符串。
	SolarAzimuthAngle string `json:"solarAzimuthAngle"`
	// SolarHour 是太阳时，"HHmm" 格式。
	SolarHour string `json:"solarHour"`
	// HourAngle 是时角。
	HourAngle string `json:"hourAngle"`
	// Refer 是数据来源与许可信息。
	//
	// 上游在结果里带 refer 对象，展示这些数据时需要一并展示。
	Refer qwx.Refer `json:"refer"`
}
