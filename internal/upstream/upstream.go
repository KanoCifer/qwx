// Package upstream 声明上游查询参数名。
//
// 内部包：上游查询参数名的唯一真源。端点包用它拼查询参数，internal/cachekey
// 用它生成查询串与缓存键后缀。零依赖 —— 不 import 根包，根包将来要用它
// 也不会成环。
package upstream

// 上游查询参数名。
//
// 值随上游文档走：上游改名时只改这里。
const (
	// Lang 是语言代码，如 "zh"、"en"。
	Lang = "lang"
	// LocalTime 是否请求当地时间。
	LocalTime = "localTime"
	// Number 是返回条数上限。
	Number = "number"
	// Radius 是搜索半径（公里）。
	Radius = "radius"
	// Adm 是上级行政区划的 LocationID。
	Adm = "adm"
	// Range 是搜索范围，如 "cn"、"world"。
	Range = "range"
	// City 是限定城市。
	City = "city"
	// Interval 是时间间隔分钟数。
	Interval = "interval"
	// Tilt 是光伏系统的倾斜角。
	Tilt = "tilt"
	// Azimuth 是光伏系统的方位角。
	Azimuth = "azimuth"
	// Extra 是额外信息，如 "weather"、"poa"。
	Extra = "extra"
	// Project 是项目 ID。
	Project = "project"
	// Credential 是凭据 ID。
	Credential = "credential"
	// Location 是位置，取值如 "116.41,39.92"、"101010100"、"P2352"。
	Location = "location"
	// Date 是日期，格式 yyyyMMdd。
	Date = "date"
	// Type 是类型，取值随端点族而异。
	Type = "type"
	// Hours 是预报小时数。
	Hours = "hours"
	// Days 是预报天数。
	Days = "days"
	// Time 是时刻，格式 HHmm。
	Time = "time"
	// Timezone 是时区偏移，格式 HHmm。
	Timezone = "tz"
	// Altitude 是海拔高度（米）。
	Altitude = "alt"
	// Basin 是台风流域代码。
	Basin = "basin"
	// Year 是年份。
	Year = "year"
	// StormID 是台风 ID。
	StormID = "stormid"
)
