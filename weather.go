package qwx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// 新一代预报接口的参数上限（官方文档口径）。
const (
	// MaxHourlyHours 逐小时预报最多 240 小时。
	MaxHourlyHours = 240
	// MaxDailyDays 每日预报最多 10 天。
	MaxDailyDays = 10
)

// 天气接口的缓存时长。
//
// 按上游的数据更新节奏取值：实况 10 分钟、逐时/逐日 1 小时。
const (
	currentCacheTTL  = 10 * time.Minute
	forecastCacheTTL = time.Hour
)

// WeatherOptions 是天气接口的可选参数。
type WeatherOptions struct {
	// Lang 是语言代码，如 "zh"、"en"。留空用上游默认。
	Lang string
	// LocalTime 是否返回查询地点的当地时间。
	//
	// 新一代接口默认给 UTC（"2026-10-21T05:00Z"），不传 localTime
	// 时间会整整差 8 小时。除非明确要 UTC，否则应设为 true。
	LocalTime bool
}

// params 转换成查询参数。
func (o WeatherOptions) params() url.Values {
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
func (o WeatherOptions) cacheSuffix() string {
	suffix := ""
	if o.Lang != "" {
		suffix += ":lang=" + o.Lang
	}
	if o.LocalTime {
		suffix += ":localTime=1"
	}
	return suffix
}

// CurrentWeather 是实时天气。
type CurrentWeather struct {
	// Metadata 是元数据。
	Metadata Metadata `json:"metadata"`
	// Condition 是天气现象。
	Condition Condition `json:"condition"`
	// Temperature 是气温。
	Temperature Value `json:"temperature"`
	// FeelsLike 是体感温度。
	FeelsLike Value `json:"feelsLike"`
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

// Current 查询实时天气。
//
// 调 GET /weather/v1/current/{latitude}/{longitude}。
//
// loc 必须带坐标：新一代接口只认经纬度路径参数，LocationID 塞进路径
// 上游直接 404（body 为空）。用 LocationID 时先用 [Client.LookupCity]
// 换成坐标。
//
// 缓存键：`qwx:v1:current:<lat>,<lon>` 加查询参数后缀。
func (c *Client) Current(ctx context.Context, loc Location, opts WeatherOptions) (*CurrentWeather, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out CurrentWeather
	err = c.do(ctx, request{
		method: "GET",
		path:   "/weather/v1/current/" + lat + "/" + lon,
		query:  opts.params(),
	}, c.currentCacheKey(loc, opts), currentCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// currentCacheKey 实时天气的缓存键。
func (c *Client) currentCacheKey(loc Location, opts WeatherOptions) string {
	return "qwx:v1:current:" + formatCoord(loc.Lat) + "," + formatCoord(loc.Lon) + opts.cacheSuffix()
}

// HourlyForecast 是逐小时预报。
type HourlyForecast struct {
	// Metadata 是元数据。
	Metadata Metadata `json:"metadata"`
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
	Temperature Value `json:"temperature"`
	// FeelsLike 是体感温度。
	FeelsLike Value `json:"feelsLike"`
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

// Hourly 查询逐小时预报。
//
// 调 GET /weather/v1/hourly/{latitude}/{longitude}?hours=N。
//
// hours 取值 1~240，超出范围返回 [ErrInvalidParameter]，不打上游。
//
// 缓存键：`qwx:v1:hourly:<lat>,<lon>:h<hours>` 加查询参数后缀。
func (c *Client) Hourly(ctx context.Context, loc Location, hours int, opts WeatherOptions) (*HourlyForecast, error) {
	if hours < 1 || hours > MaxHourlyHours {
		return nil, fmt.Errorf("%w: hours %d out of range [1,%d]", ErrInvalidParameter, hours, MaxHourlyHours)
	}
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	q := opts.params()
	q.Set("hours", strconv.Itoa(hours))

	var out HourlyForecast
	err = c.do(ctx, request{
		method: "GET",
		path:   "/weather/v1/hourly/" + lat + "/" + lon,
		query:  q,
	}, c.hourlyCacheKey(loc, hours, opts), forecastCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// hourlyCacheKey 逐小时预报的缓存键。
//
// 预报小时数进键：hours=24 与 hours=48 是两份不同的数据，共键会串。
func (c *Client) hourlyCacheKey(loc Location, hours int, opts WeatherOptions) string {
	return "qwx:v1:hourly:" + formatCoord(loc.Lat) + "," + formatCoord(loc.Lon) +
		":h" + strconv.Itoa(hours) + opts.cacheSuffix()
}

// DailyForecast 是逐日预报。
type DailyForecast struct {
	// Metadata 是元数据。
	Metadata Metadata `json:"metadata"`
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
	TemperatureMax Value `json:"temperatureMax"`
	// TemperatureMin 是最低气温。
	TemperatureMin Value `json:"temperatureMin"`
	// TemperatureAvg 是平均气温。
	TemperatureAvg Value `json:"temperatureAvg"`
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
	TemperatureMax Value `json:"temperatureMax"`
	// TemperatureMin 是最低气温。
	TemperatureMin Value `json:"temperatureMin"`
	// Wind 是风。
	Wind Wind `json:"wind"`
	// WindGust 是阵风。
	WindGust WindGust `json:"windGust"`
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

// Daily 查询逐日预报。
//
// 调 GET /weather/v1/daily/{latitude}/{longitude}?days=N。
//
// days 取值 1~10，超出范围返回 [ErrInvalidParameter]，不打上游。
//
// 缓存键：`qwx:v1:daily:<lat>,<lon>:d<days>` 加查询参数后缀。
func (c *Client) Daily(ctx context.Context, loc Location, days int, opts WeatherOptions) (*DailyForecast, error) {
	if days < 1 || days > MaxDailyDays {
		return nil, fmt.Errorf("%w: days %d out of range [1,%d]", ErrInvalidParameter, days, MaxDailyDays)
	}
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	q := opts.params()
	q.Set("days", strconv.Itoa(days))

	var out DailyForecast
	err = c.do(ctx, request{
		method: "GET",
		path:   "/weather/v1/daily/" + lat + "/" + lon,
		query:  q,
	}, c.dailyCacheKey(loc, days, opts), forecastCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// dailyCacheKey 逐日预报的缓存键。
func (c *Client) dailyCacheKey(loc Location, days int, opts WeatherOptions) string {
	return "qwx:v1:daily:" + formatCoord(loc.Lat) + "," + formatCoord(loc.Lon) +
		":d" + strconv.Itoa(days) + opts.cacheSuffix()
}
