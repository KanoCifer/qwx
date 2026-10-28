package weather

import (
	"net/url"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// 新一代预报接口的参数上限（官方文档口径）。
const (
	// MaxHourlyHours 逐小时预报最多 240 小时。
	MaxHourlyHours = 240
	// MaxDailyDays 每日预报最多 10 天。
	MaxDailyDays = 10
)

// 天气端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// EndpointCurrent 是实时天气端点。
	EndpointCurrent = qwx.Endpoint("weather/v1/current")
	// EndpointHourly 是逐小时预报端点。
	EndpointHourly = qwx.Endpoint("weather/v1/hourly")
	// EndpointDaily 是逐日预报端点。
	EndpointDaily = qwx.Endpoint("weather/v1/daily")
)

// 天气接口的默认缓存时长。
//
// 按上游的数据更新节奏取值：实况 10 分钟、逐时/逐日 1 小时。
const (
	currentCacheTTL  = 10 * time.Minute
	forecastCacheTTL = time.Hour
)

// Options 是天气接口的可选参数。
type Options struct {
	// Lang 是语言代码，如 "zh"、"en"。留空用上游默认。
	Lang string
	// LocalTime 是否返回查询地点的当地时间。
	//
	// 新一代接口默认给 UTC（"2026-10-21T05:00Z"），不传 localTime
	// 时间会整整差 8 小时。除非明确要 UTC，否则应设为 true。
	LocalTime bool
}

// params 转换成查询参数。
func (o Options) params() url.Values {
	q := url.Values{}
	if o.Lang != "" {
		q.Set("lang", o.Lang)
	}
	if o.LocalTime {
		q.Set("localTime", "true")
	}
	return q
}

// cacheSuffix 是查询参数对缓存键的影响部分。
func (o Options) cacheSuffix() string {
	suffix := cachekey.Lang(o.Lang)
	if o.LocalTime {
		suffix += ":localTime=1"
	}
	return suffix
}

// CurrentWeather 是实时天气。
type CurrentWeather struct {
	// Metadata 是元数据。
	Metadata qwx.Metadata `json:"metadata"`
	// Condition 是天气现象。
	Condition qwx.Condition `json:"condition"`
	// Temperature 是气温。
	Temperature qwx.Value `json:"temperature"`
	// FeelsLike 是体感温度。
	FeelsLike qwx.Value `json:"feelsLike"`
	// Humidity 是相对湿度，**0~1 的比例**（不是百分数）。
	Humidity float64 `json:"humidity"`
	// Wind 是风。
	Wind qwx.Wind `json:"wind"`
	// WindGust 是阵风。
	WindGust qwx.WindGust `json:"windGust"`
	// Precipitation 是降水情况。
	Precipitation qwx.Precipitation `json:"precipitation"`
	// Pressure 是气压。
	Pressure qwx.Pressure `json:"pressure"`
	// Visibility 是能见度。
	Visibility qwx.Visibility `json:"visibility"`
	// DewPoint 是露点。
	DewPoint qwx.DewPoint `json:"dewPoint"`
	// CloudCover 是云量，**0~1 的比例**。
	CloudCover float64 `json:"cloudCover"`
	// UVIndex 是紫外线指数。
	UVIndex float64 `json:"uvIndex"`
}

// HourlyForecast 是逐小时预报。
type HourlyForecast struct {
	// Metadata 是元数据。
	Metadata qwx.Metadata `json:"metadata"`
	// Hours 是逐小时预报列表。
	//
	// 条数由请求的 hours 决定，但上游可能给得更少 —— 不要假设长度。
	Hours []Hourly `json:"hours"`
}

// Hourly 是一个小时的预报。
type Hourly struct {
	// ForecastTime 是预报时刻。
	ForecastTime string `json:"forecastTime"`
	// Condition 是天气现象。
	Condition qwx.Condition `json:"condition"`
	// Temperature 是气温。
	Temperature qwx.Value `json:"temperature"`
	// FeelsLike 是体感温度。
	FeelsLike qwx.Value `json:"feelsLike"`
	// Humidity 是相对湿度，0~1 的比例。
	Humidity float64 `json:"humidity"`
	// Wind 是风。
	Wind qwx.Wind `json:"wind"`
	// WindGust 是阵风。
	WindGust qwx.WindGust `json:"windGust"`
	// Precipitation 是降水情况，含概率。
	Precipitation qwx.Precipitation `json:"precipitation"`
	// Pressure 是气压。
	Pressure qwx.Pressure `json:"pressure"`
	// Visibility 是能见度。
	Visibility qwx.Visibility `json:"visibility"`
	// DewPoint 是露点。
	DewPoint qwx.DewPoint `json:"dewPoint"`
	// CloudCover 是云量，0~1 的比例。
	CloudCover float64 `json:"cloudCover"`
	// UVIndex 是紫外线指数。
	UVIndex float64 `json:"uvIndex"`
}

// DailyForecast 是逐日预报。
type DailyForecast struct {
	// Metadata 是元数据。
	Metadata qwx.Metadata `json:"metadata"`
	// Days 是逐日预报列表。
	Days []Daily `json:"days"`
}

// Daily 是一天的预报。
type Daily struct {
	// ForecastStartTime 是预报起始时刻。
	ForecastStartTime string `json:"forecastStartTime"`
	// ForecastEndTime 是预报结束时刻。
	ForecastEndTime string `json:"forecastEndTime"`
	// Astro 是天文信息（日出日落、月相等）。
	//
	// 高纬度地区可能没有日出日落 —— 该字段可能缺失。
	Astro *Astro `json:"astro,omitempty"`
	// TemperatureMax 是最高气温。
	TemperatureMax qwx.Value `json:"temperatureMax"`
	// TemperatureMin 是最低气温。
	TemperatureMin qwx.Value `json:"temperatureMin"`
	// TemperatureAvg 是平均气温。
	TemperatureAvg qwx.Value `json:"temperatureAvg"`
	// UVIndexMax 是最大紫外线指数。
	UVIndexMax float64 `json:"uvIndexMax"`
	// Daytime 是白天时段（v7 时代是一天一条，新一代拆成白天/夜间两段）。
	Daytime DayNight `json:"daytime"`
	// Nighttime 是夜间时段。
	Nighttime DayNight `json:"nighttime"`
}

// DayNight 是白天或夜间时段的天气。
type DayNight struct {
	// ForecastStartTime 是时段起始时刻。
	ForecastStartTime string `json:"forecastStartTime"`
	// ForecastEndTime 是时段结束时刻。
	ForecastEndTime string `json:"forecastEndTime"`
	// Condition 是天气现象。
	Condition qwx.Condition `json:"condition"`
	// TemperatureMax 是最高气温。
	TemperatureMax qwx.Value `json:"temperatureMax"`
	// TemperatureMin 是最低气温。
	TemperatureMin qwx.Value `json:"temperatureMin"`
	// Wind 是风。
	Wind qwx.Wind `json:"wind"`
	// WindGust 是阵风。
	WindGust qwx.WindGust `json:"windGust"`
	// Precipitation 是降水情况。
	Precipitation qwx.Precipitation `json:"precipitation"`
	// Humidity 是相对湿度，0~1 的比例。
	Humidity float64 `json:"humidity"`
	// CloudCover 是云量，0~1 的比例。
	CloudCover float64 `json:"cloudCover"`
}

// Astro 是天文信息。
type Astro struct {
	// Sunrise 是日出时刻。
	Sunrise string `json:"sunrise"`
	// Sunset 是日落时刻。
	Sunset string `json:"sunset"`
	// AstronomicalDawn 是天文曙光开始时刻。
	AstronomicalDawn string `json:"astronomicalDawn"`
	// NauticalDawn 是航海曙光开始时刻。
	NauticalDawn string `json:"nauticalDawn"`
	// CivilDawn 是民用曙光开始时刻。
	CivilDawn string `json:"civilDawn"`
	// AstronomicalDusk 是天文暮光结束时刻。
	AstronomicalDusk string `json:"astronomicalDusk"`
	// NauticalDusk 是航海暮光结束时刻。
	NauticalDusk string `json:"nauticalDusk"`
	// CivilDusk 是民用暮光结束时刻。
	CivilDusk string `json:"civilDusk"`
	// SolarNoon 是正午时刻。
	SolarNoon string `json:"solarNoon"`
	// SolarMidnight 是子夜时刻。
	SolarMidnight string `json:"solarMidnight"`
	// Moonrise 是月出时刻。
	Moonrise string `json:"moonrise"`
	// Moonset 是月落时刻。
	Moonset string `json:"moonset"`
	// MoonTransit 是月过中天时刻。
	MoonTransit string `json:"moonTransit"`
	// MoonUnderfoot 是月球位于天顶的时刻。
	MoonUnderfoot string `json:"moonUnderfoot"`
	// MoonPhase 是月相代码，如 "waxing-gibbous"。
	MoonPhase string `json:"moonPhase"`
}
