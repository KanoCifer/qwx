package airquality

import (
	"net/url"

	"github.com/KanoCifer/qwx"
)

// Options 是空气质量接口的可选参数。
type Options struct {
	// Lang 是语言代码。留空用上游默认。
	Lang string
}

// params 转换成查询参数。
func (o Options) params() url.Values {
	q := url.Values{}
	if o.Lang != "" {
		q.Set("lang", o.Lang)
	}
	return q
}

// Index 是一个 AQI 指数条目。
type Index struct {
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
	Color qwx.Color `json:"color"`
	// PrimaryPollutant 是首要污染物。可能是 nil（空气很好时无首要污染物）。
	PrimaryPollutant *PollutantBrief `json:"primaryPollutant,omitempty"`
	// Health 是健康建议。
	Health *Health `json:"health,omitempty"`
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

// Health 是健康影响与建议。
type Health struct {
	// Effect 是健康影响说明。
	Effect string `json:"effect"`
	// Advice 是分人群的活动建议。
	Advice Advice `json:"advice"`
}

// Advice 是分人群建议。
type Advice struct {
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
	Concentration qwx.Value `json:"concentration"`
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

// Station 是一个监测站。
type Station struct {
	// ID 是站点 ID，如 "P51762"。
	ID string `json:"id"`
	// Name 是站点名称。
	Name string `json:"name"`
}

// Current 是实时空气质量。
type Current struct {
	// Metadata 是元数据。
	Metadata qwx.Metadata `json:"metadata"`
	// Indexes 是各标准的 AQI 指数。
	Indexes []Index `json:"indexes"`
	// Pollutants 是各污染物的实测数据。
	Pollutants []Pollutant `json:"pollutants"`
	// Stations 是附近的监测站。
	Stations []Station `json:"stations"`
}

// DailyForecast 是逐日空气质量预报。
type DailyForecast struct {
	// Metadata 是元数据。
	Metadata qwx.Metadata `json:"metadata"`
	// Days 是逐日预报。
	Days []Day `json:"days"`
}

// Day 是一天的空气质量预报。
type Day struct {
	// ForecastStartTime 是预报起始时刻。
	ForecastStartTime string `json:"forecastStartTime"`
	// ForecastEndTime 是预报结束时刻。
	ForecastEndTime string `json:"forecastEndTime"`
	// Indexes 是各标准的 AQI 指数。
	Indexes []Index `json:"indexes"`
	// Pollutants 是各污染物的预报浓度。
	Pollutants []Pollutant `json:"pollutants"`
}

// HourlyForecast 是逐小时空气质量预报。
type HourlyForecast struct {
	// Metadata 是元数据。
	Metadata qwx.Metadata `json:"metadata"`
	// Hours 是逐小时预报。
	Hours []Hour `json:"hours"`
}

// Hour 是一小时的空气质量预报。
type Hour struct {
	// ForecastTime 是预报时刻。
	ForecastTime string `json:"forecastTime"`
	// Indexes 是各标准的 AQI 指数。
	Indexes []Index `json:"indexes"`
	// Pollutants 是各污染物的预报浓度。
	Pollutants []Pollutant `json:"pollutants"`
}

// StationData 是单个监测站的污染物数据。
type StationData struct {
	// Metadata 是元数据。
	Metadata qwx.Metadata `json:"metadata"`
	// Pollutants 是各污染物的实测数据。
	Pollutants []Pollutant `json:"pollutants"`
}
