package qwx

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

// 空气质量接口的缓存时长。
//
// 实况 30 分钟（上游逐小时更新，留一半余量），预报 1 小时。
const (
	airCurrentCacheTTL  = 30 * time.Minute
	airForecastCacheTTL = time.Hour
)

// AirQualityOptions 是空气质量接口的可选参数。
type AirQualityOptions struct {
	// Lang 是语言代码。留空用上游默认。
	Lang string
}

// params 转换成查询参数。
func (o AirQualityOptions) params() url.Values {
	q := url.Values{}
	if o.Lang != "" {
		q.Set("lang", o.Lang)
	}
	return q
}

// AQIIndex 是一个 AQI 指数条目。
type AQIIndex struct {
	// Code 是标准代码，如 "us-epa"、"qaqi"。
	Code string `json:"code"`
	// Name 是标准名称，如 "AQI (US)"。
	Name string `json:"name"`
	// AQI 是指数值。
	//
	// 不同标准的量纲不同（QAQI 是 0~1，US EPA 是 0~500），不要跨标准比较。
	AQI float64 `json:"aqi"`
	// AQIDisplay 是上游给的可读形式。
	AQIDisplay string `json:"aqiDisplay"`
	// Level 是等级（"1" ~ "6"）。
	Level string `json:"level"`
	// Category 是等级描述，如 "Good"、"Excellent"。
	Category string `json:"category"`
	// Color 是等级对应的颜色。
	Color Color `json:"color"`
	// PrimaryPollutant 是首要污染物。可能是 nil（空气很好时无首要污染物）。
	PrimaryPollutant *PollutantBrief `json:"primaryPollutant,omitempty"`
	// Health 是健康建议。
	Health *AQHealth `json:"health,omitempty"`
}

// PollutantBrief 是污染物的简要信息。
type PollutantBrief struct {
	// Code 是污染物代码，如 "pm2p5"。
	Code string `json:"code"`
	// Name 是名称，如 "PM 2.5"。
	Name string `json:"name"`
	// FullName 是全称。
	FullName string `json:"fullName"`
}

// AQHealth 是健康影响与建议。
type AQHealth struct {
	// Effect 是健康影响说明。
	Effect string `json:"effect"`
	// Advice 是分人群的活动建议。
	Advice AQAdvice `json:"advice"`
}

// AQAdvice 是分人群建议。
type AQAdvice struct {
	// GeneralPopulation 是普通人群的建议。
	GeneralPopulation string `json:"generalPopulation"`
	// SensitivePopulation 是敏感人群的建议。
	SensitivePopulation string `json:"sensitivePopulation"`
}

// Pollutant 是一个污染物的实测数据。
type Pollutant struct {
	// Code 是污染物代码。
	Code string `json:"code"`
	// Name 是名称。
	Name string `json:"name"`
	// FullName 是全称。
	FullName string `json:"fullName"`
	// Concentration 是浓度值。
	Concentration Value `json:"concentration"`
	// SubIndexes 是该污染物在各标准下的分指数。
	SubIndexes []SubIndex `json:"subIndexes"`
}

// SubIndex 是一个污染物在某标准下的分指数。
type SubIndex struct {
	// Code 是标准代码。
	Code string `json:"code"`
	// AQI 是分指数值。
	AQI float64 `json:"aqi"`
	// AQIDisplay 是可读形式。
	AQIDisplay string `json:"aqiDisplay"`
}

// AirStation 是一个监测站。
type AirStation struct {
	// ID 是站点 ID，如 "P51762"。
	ID string `json:"id"`
	// Name 是站点名称。
	Name string `json:"name"`
}

// AirQualityCurrent 是实时空气质量。
type AirQualityCurrent struct {
	// Metadata 是元数据。
	Metadata Metadata `json:"metadata"`
	// Indexes 是各标准的 AQI 指数。
	Indexes []AQIIndex `json:"indexes"`
	// Pollutants 是各污染物的实测数据。
	Pollutants []Pollutant `json:"pollutants"`
	// Stations 是附近的监测站。
	Stations []AirStation `json:"stations"`
}

// AirQuality 查询实时空气质量。
//
// 调 GET /airquality/v1/current/{latitude}/{longitude}。
//
// 缓存键：`qwx:aq:v1:current:<lat>,<lon>` 加语言后缀。
func (c *Client) AirQuality(ctx context.Context, loc Location, opts AirQualityOptions) (*AirQualityCurrent, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out AirQualityCurrent
	err = c.do(ctx, request{
		method: "GET",
		path:   "/airquality/v1/current/" + lat + "/" + lon,
		query:  opts.params(),
	}, "qwx:aq:v1:current:"+formatCoord(loc.Lat)+","+formatCoord(loc.Lon)+langSuffix(opts.Lang), airCurrentCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AirQualityDaily 是逐日空气质量预报。
type AirQualityDaily struct {
	// Metadata 是元数据。
	Metadata Metadata `json:"metadata"`
	// Days 是逐日预报。
	Days []AirQualityDay `json:"days"`
}

// AirQualityDay 是一天的空气质量预报。
type AirQualityDay struct {
	// ForecastStartTime 是预报起始时刻。
	ForecastStartTime string `json:"forecastStartTime"`
	// ForecastEndTime 是预报结束时刻。
	ForecastEndTime string `json:"forecastEndTime"`
	// Indexes 是各标准的 AQI 指数。
	Indexes []AQIIndex `json:"indexes"`
	// Pollutants 是各污染物的预报浓度。
	Pollutants []Pollutant `json:"pollutants"`
}

// AirQualityDailyForecast 查询逐日空气质量预报。
//
// 调 GET /airquality/v1/daily/{latitude}/{longitude}。
//
// 缓存键：`qwx:aq:v1:daily:<lat>,<lon>` 加语言后缀。
func (c *Client) AirQualityDailyForecast(ctx context.Context, loc Location, opts AirQualityOptions) (*AirQualityDaily, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out AirQualityDaily
	err = c.do(ctx, request{
		method: "GET",
		path:   "/airquality/v1/daily/" + lat + "/" + lon,
		query:  opts.params(),
	}, "qwx:aq:v1:daily:"+formatCoord(loc.Lat)+","+formatCoord(loc.Lon)+langSuffix(opts.Lang), airForecastCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AirQualityHourly 是逐小时空气质量预报。
type AirQualityHourly struct {
	// Metadata 是元数据。
	Metadata Metadata `json:"metadata"`
	// Hours 是逐小时预报。
	Hours []AirQualityHour `json:"hours"`
}

// AirQualityHour 是一小时的空气质量预报。
type AirQualityHour struct {
	// ForecastTime 是预报时刻。
	ForecastTime string `json:"forecastTime"`
	// Indexes 是各标准的 AQI 指数。
	Indexes []AQIIndex `json:"indexes"`
	// Pollutants 是各污染物的预报浓度。
	Pollutants []Pollutant `json:"pollutants"`
}

// AirQualityHourlyForecast 查询逐小时空气质量预报。
//
// 调 GET /airquality/v1/hourly/{latitude}/{longitude}。
//
// 缓存键：`qwx:aq:v1:hourly:<lat>,<lon>` 加语言后缀。
func (c *Client) AirQualityHourlyForecast(ctx context.Context, loc Location, opts AirQualityOptions) (*AirQualityHourly, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out AirQualityHourly
	err = c.do(ctx, request{
		method: "GET",
		path:   "/airquality/v1/hourly/" + lat + "/" + lon,
		query:  opts.params(),
	}, "qwx:aq:v1:hourly:"+formatCoord(loc.Lat)+","+formatCoord(loc.Lon)+langSuffix(opts.Lang), airForecastCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AirStationData 是单个监测站的污染物数据。
type AirStationData struct {
	// Metadata 是元数据。
	Metadata Metadata `json:"metadata"`
	// Pollutants 是各污染物的实测数据。
	Pollutants []Pollutant `json:"pollutants"`
}

// AirStation 查询单个监测站的污染物数据。
//
// 调 GET /airquality/v1/stations/{locationId}。
//
// stationID 是监测站 ID（如 "P51762"），来自 [AirQualityCurrent] 的
// Stations 列表。
//
// 缓存键：`qwx:aq:v1:station:<id>` 加语言后缀。
func (c *Client) AirStation(ctx context.Context, stationID string, opts AirQualityOptions) (*AirStationData, error) {
	if stationID == "" {
		return nil, fmt.Errorf("%w: empty station ID", ErrInvalidParameter)
	}
	var out AirStationData
	err := c.do(ctx, request{
		method: "GET",
		path:   "/airquality/v1/stations/" + stationID,
		query:  opts.params(),
	}, "qwx:aq:v1:station:"+stationID+langSuffix(opts.Lang), airCurrentCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// langSuffix 是语言参数对缓存键的影响部分。
func langSuffix(lang string) string {
	if lang == "" {
		return ""
	}
	return ":lang=" + lang
}
