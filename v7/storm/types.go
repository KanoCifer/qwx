package storm

import (
	"time"

	"github.com/KanoCifer/qwx"
)

// 台风数据端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// EndpointList 是台风列表端点。
	EndpointList = qwx.Endpoint("v7/tropical/storm-list")
	// EndpointForecast 是台风预报端点。
	EndpointForecast = qwx.Endpoint("v7/tropical/storm-forecast")
	// EndpointTrack 是台风实况和路径端点。
	EndpointTrack = qwx.Endpoint("v7/tropical/storm-track")
)

// 各端点的默认缓存时长。
const (
	// 台风列表一天内基本不变。
	listCacheTTL = 24 * time.Hour
	// 台风预报上游几小时更新一次。
	forecastCacheTTL = time.Hour
	// 台风路径上游几小时更新一次。
	trackCacheTTL = time.Hour
)

// BasinNP 是西北太平洋流域代码（basin=NP）。
//
// 上游目前只支持这一个流域（"目前仅支持中国沿海地区，即 basin=NP"）。
// 别的流域代码上游不认，客户端直接拒绝。
const BasinNP = "NP"

// ListResult 是台风列表查询的结果。
type ListResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// UpdateTime 是数据更新时间。
	UpdateTime string `json:"updateTime"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// Storms 是台风列表。
	//
	// 上游的 "storm" 是数组，不是对象。长度不固定：一个年份里可能有几十个
	// 台风，也可能一个都没有。
	Storms []StormSummary `json:"storm"`
	// Refer 是数据来源与许可信息。
	//
	// 上游在结果里带 refer 对象，展示这些数据时需要一并展示。
	Refer qwx.Refer `json:"refer"`
}

// StormSummary 是一个台风的概要。
type StormSummary struct {
	// ID 是台风 ID，如 "NP2018"。
	ID string `json:"id"`
	// Name 是台风名称，如 "莫拉菲"。
	Name string `json:"name"`
	// Basin 是台风所处流域，如 "NP"。
	Basin string `json:"basin"`
	// Year 是台风所处年份。上游给的是字符串，这里保持字符串。
	Year string `json:"year"`
	// IsActive 是活跃标记："1" 活跃台风，"0" 停编。
	//
	// 上游给的是字符串，这里保持字符串，不转成 bool。
	IsActive string `json:"isActive"`
}

// ForecastResult 是台风预报查询的结果。
type ForecastResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// UpdateTime 是数据更新时间。
	UpdateTime string `json:"updateTime"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// Forecast 是台风预报数据列表。
	//
	// 台风已结束时上游返回空列表。
	Forecast []Forecast `json:"forecast"`
	// Refer 是数据来源与许可信息。
	//
	// 上游在结果里带 refer 对象，展示这些数据时需要一并展示。
	Refer qwx.Refer `json:"refer"`
}

// TrackResult 是台风实况和路径查询的结果。
type TrackResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// UpdateTime 是数据更新时间。
	UpdateTime string `json:"updateTime"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// IsActive 是活跃标记："1" 活跃台风，"0" 停编。
	//
	// 上游给的是字符串，这里保持字符串，不转成 bool。
	IsActive string `json:"isActive"`
	// Now 是当前台风实况。
	//
	// 台风已结束时上游可能给 null，这时是 nil。
	Now *TrackPoint `json:"now"`
	// Track 是台风轨迹点列表。长度不固定。
	Track []TrackPoint `json:"track"`
	// Refer 是数据来源与许可信息。
	//
	// 上游在结果里带 refer 对象，展示这些数据时需要一并展示。
	Refer qwx.Refer `json:"refer"`
}

// StormState 是一个时刻的台风状态。
//
// 上游把台风属性平铺在预报元素与轨迹元素里，这两组字段完全一样，所以
// 收成一个类型内嵌进 [Forecast] 与 [TrackPoint] —— JSON 形状不变，字段
// 访问也不变。
type StormState struct {
	// Lat 是台风所处纬度。
	//
	// 上游把数值包成字符串，这里保持字符串 —— 小数位数是上游的表达精度，
	// 转成 float64 会丢掉它。
	Lat string `json:"lat"`
	// Lon 是台风所处经度。上游给的是字符串，这里保持字符串。
	Lon string `json:"lon"`
	// Type 是台风类型，如 "TS"、"TD"、"STS"。
	Type string `json:"type"`
	// Pressure 是台风中心气压。上游给的是字符串，这里保持字符串。
	Pressure string `json:"pressure"`
	// WindSpeed 是台风附近最大风速。上游给的是字符串，这里保持字符串。
	WindSpeed string `json:"windSpeed"`
	// MoveSpeed 是台风移动速度。上游给的是字符串，这里保持字符串。
	MoveSpeed string `json:"moveSpeed"`
	// MoveDir 是台风移动方位，如 "NE"。
	MoveDir string `json:"moveDir"`
	// Move360 是台风移动方位 360 度方向。上游可能是空串。
	Move360 string `json:"move360"`
	// WindRadius30 是 7 级风圈半径。台风等级不够时上游不给，这时是 nil。
	WindRadius30 *WindRadius `json:"windRadius30,omitempty"`
	// WindRadius50 是 10 级风圈半径。台风等级不够时上游不给，这时是 nil。
	WindRadius50 *WindRadius `json:"windRadius50,omitempty"`
	// WindRadius64 是 12 级风圈半径。台风等级不够时上游不给，这时是 nil。
	WindRadius64 *WindRadius `json:"windRadius64,omitempty"`
}

// Forecast 是一个预报时刻的台风状态。
//
// 上游把台风属性平铺在预报元素里，这里按上游 JSON 一一对应，不做嵌套。
type Forecast struct {
	// FXTime 是预报时刻。
	FXTime string `json:"fxTime"`
	// StormState 是该时刻的台风状态。
	StormState
}

// TrackPoint 是一个轨迹点的台风状态。
//
// 上游把台风属性平铺在轨迹元素里，这里按上游 JSON 一一对应，不做嵌套。
type TrackPoint struct {
	// PubTime 是实况发布时间。只有 "now" 元素有这个字段。
	PubTime string `json:"pubTime"`
	// Time 是轨迹点时刻。只有 "track" 数组元素有这个字段。
	Time string `json:"time"`
	// StormState 是该时刻的台风状态。
	StormState
}

// WindRadius 是一个风圈的四个方向半径。
type WindRadius struct {
	// NERadius 是东北半径。上游可能是空串。
	NERadius string `json:"neRadius"`
	// SERadius 是东南半径。上游可能是空串。
	SERadius string `json:"seRadius"`
	// SWRadius 是西南半径。上游可能是空串。
	SWRadius string `json:"swRadius"`
	// NWRadius 是西北半径。上游可能是空串。
	NWRadius string `json:"nwRadius"`
}
