package weather

import (
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
	v1 "github.com/KanoCifer/qwx/v1"
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
//
// 它是 [github.com/KanoCifer/qwx/v1.Options] 的别名：天气与预警两个端点族
// 接受同一组可选参数，类型放在代际共享包里。
type Options = v1.Options

// params 是影响结果的查询参数。
//
// 同一份参数既生成上游查询串，也生成缓存键后缀 —— 新增参数时不会漏进键。
func params(o Options) cachekey.Params {
	return cachekey.Params{Lang: o.Lang, LocalTime: o.LocalTime}
}

// Condition 是天气现象。
type Condition struct {
	// Text 是天气现象的文字描述，随 lang 变化。
	Text string `json:"text"`
	// Code 是天气现象代码，如 "100"（晴）、"300"（小雨）。
	//
	// 官方会新增代码，不要依赖枚举的完备性。
	Code string `json:"code"`
}

// WindDirection 是风向。
type WindDirection struct {
	// Degree 是风向角度（0~360，0 为北）。
	Degree float64 `json:"degree"`
	// Compass 是 16 方位英文代码（如 "sw"、"wnw"），不是中文。
	Compass string `json:"compass"`
}

// Wind 是风速与风向。
type Wind struct {
	// Direction 是风向。
	Direction WindDirection `json:"direction"`
	// Speed 是风速。
	Speed v1.Value `json:"speed"`
	// Scale 是风力等级（蒲福风级）。
	Scale float64 `json:"scale"`
}

// WindGust 是阵风。
type WindGust struct {
	// Value 是阵风风速。
	Value float64 `json:"value"`
	// Unit 是单位。
	Unit string `json:"unit"`
}

// Precipitation 是降水情况。
type Precipitation struct {
	// Amount 是降水量。
	Amount v1.Value `json:"amount"`
	// Intensity 是降水强度。
	//
	// 实况与逐时预报有这个字段，逐日预报的白天/夜间时段没有 ——
	// 用指针区分「没有」与「强度为 0」。
	Intensity *v1.Value `json:"intensity,omitempty"`
	// Probability 是降水概率（0~1）。
	//
	// 实时天气接口没有这个字段，逐小时预报与逐日预报的白天/夜间时段
	// 才有。用指针区分「没有」与「概率为 0」。
	Probability *float64 `json:"probability,omitempty"`
	// Type 是降水类型代码，如 "rain"、"snow"、"none"。
	//
	// 上游会新增类型，不要依赖枚举的完备性。
	Type string `json:"type"`
}

// Pressure 是气压。
type Pressure struct {
	// Value 是气压值。
	Value float64 `json:"value"`
	// Unit 是单位（hPa）。
	Unit string `json:"unit"`
}

// Visibility 是能见度。
type Visibility struct {
	// Value 是能见度值。
	Value float64 `json:"value"`
	// Unit 是单位（m）。
	Unit string `json:"unit"`
}

// DewPoint 是露点温度。
type DewPoint struct {
	// Value 是露点温度。
	Value float64 `json:"value"`
	// Unit 是单位（°C）。
	Unit string `json:"unit"`
}

// CurrentWeather 是实时天气。
type CurrentWeather struct {
	// Metadata 是元数据。
	Metadata v1.Metadata `json:"metadata"`
	// Condition 是天气现象。
	Condition Condition `json:"condition"`
	// Temperature 是气温。
	Temperature v1.Value `json:"temperature"`
	// FeelsLike 是体感温度。
	FeelsLike v1.Value `json:"feelsLike"`
	// Humidity 是相对湿度，**0~1 的比例**（不是百分数）。
	Humidity float64 `json:"humidity"`
	// Wind 是风。
	Wind Wind `json:"wind"`
	// WindGust 是阵风。
	WindGust WindGust `json:"windGust"`
	// Precipitation 是降水情况。
	Precipitation Precipitation `json:"precipitation"`
	// Pressure 是气压。
	Pressure Pressure `json:"pressure"`
	// Visibility 是能见度。
	Visibility Visibility `json:"visibility"`
	// DewPoint 是露点。
	DewPoint DewPoint `json:"dewPoint"`
	// CloudCover 是云量，**0~1 的比例**。
	CloudCover float64 `json:"cloudCover"`
	// UVIndex 是紫外线指数。
	UVIndex float64 `json:"uvIndex"`
}

// HourlyForecast 是逐小时预报。
type HourlyForecast struct {
	// Metadata 是元数据。
	Metadata v1.Metadata `json:"metadata"`
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
	Condition Condition `json:"condition"`
	// Temperature 是气温。
	Temperature v1.Value `json:"temperature"`
	// FeelsLike 是体感温度。
	FeelsLike v1.Value `json:"feelsLike"`
	// Humidity 是相对湿度，0~1 的比例。
	Humidity float64 `json:"humidity"`
	// Wind 是风。
	Wind Wind `json:"wind"`
	// WindGust 是阵风。
	WindGust WindGust `json:"windGust"`
	// Precipitation 是降水情况，含概率。
	Precipitation Precipitation `json:"precipitation"`
	// Pressure 是气压。
	Pressure Pressure `json:"pressure"`
	// Visibility 是能见度。
	Visibility Visibility `json:"visibility"`
	// DewPoint 是露点。
	DewPoint DewPoint `json:"dewPoint"`
	// CloudCover 是云量，0~1 的比例。
	CloudCover float64 `json:"cloudCover"`
	// UVIndex 是紫外线指数。
	UVIndex float64 `json:"uvIndex"`
}

// DailyForecast 是逐日预报。
type DailyForecast struct {
	// Metadata 是元数据。
	Metadata v1.Metadata `json:"metadata"`
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
	TemperatureMax v1.Value `json:"temperatureMax"`
	// TemperatureMin 是最低气温。
	TemperatureMin v1.Value `json:"temperatureMin"`
	// TemperatureAvg 是平均气温。
	TemperatureAvg v1.Value `json:"temperatureAvg"`
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
	Condition Condition `json:"condition"`
	// TemperatureMax 是最高气温。
	TemperatureMax v1.Value `json:"temperatureMax"`
	// TemperatureMin 是最低气温。
	TemperatureMin v1.Value `json:"temperatureMin"`
	// Wind 是风。
	Wind Wind `json:"wind"`
	// WindGustMax 是时段内的最大阵风。
	//
	// 上游 daily 端点的白天/夜间时段用 windGustMax，与 current、hourly
	// 端点的 windGust 不是同一个键。
	WindGustMax WindGust `json:"windGustMax"`
	// Precipitation 是降水情况。
	Precipitation Precipitation `json:"precipitation"`
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
