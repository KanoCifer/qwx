package indices

import (
	"time"

	"github.com/KanoCifer/qwx"
)

// 生活指数端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// Endpoint 是生活指数端点。
	Endpoint = qwx.Endpoint("v7/indices")
)

// 生活指数接口的默认缓存时长：上游一天更新几次，缓存一小时。
const cacheTTL = time.Hour

// 生活指数类型代码。
//
// 取值 1~16，具体含义见官方「天气指数」文档。
const (
	// TypeAll 取全部生活指数（type=0）。
	TypeAll = 0
	// TypeFishing 钓鱼指数（type=4）。
	TypeFishing = 4
	// MaxType 最大的指数类型代码。
	MaxType = 16
)

// Result 是生活指数查询的结果。
type Result struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// UpdateTime 是数据更新时间。
	UpdateTime string `json:"updateTime"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// Daily 是逐日指数。
	Daily []DailyIndex `json:"daily"`
}

// DailyIndex 是一天的生活指数。
type DailyIndex struct {
	// Date 是日期，"YYYY-MM-DD"。
	Date string `json:"date"`
	// Type 是指数类型代码。
	Type string `json:"type"`
	// Name 是指数名称，如 "运动指数"。
	Name string `json:"name"`
	// Level 是等级。上游给的是字符串，这里保持字符串。
	Level string `json:"level"`
	// Category 是等级描述，如 "较不宜"。
	Category string `json:"category"`
	// Text 是详细说明。
	Text string `json:"text"`
}
