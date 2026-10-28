package ocean

import (
	"time"

	"github.com/KanoCifer/qwx"
)

// 潮汐端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// Endpoint 是潮汐端点。
	Endpoint = qwx.Endpoint("v7/ocean/tide")
)

// 潮汐接口的默认缓存时长：上游一天更新几次，缓存一小时。
const cacheTTL = time.Hour

// Result 是潮汐数据。
type Result struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// UpdateTime 是数据更新时间。
	UpdateTime string `json:"updateTime"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// TideTable 是潮汐表（高低潮）。
	TideTable []Tide `json:"tideTable"`
	// TideHourly 是逐小时潮位。
	TideHourly []TideHourly `json:"tideHourly"`
	// Refer 是数据来源与许可信息。
	//
	// 上游在结果里带 refer 对象，展示这些数据时需要一并展示。
	Refer qwx.Refer `json:"refer"`
}

// Tide 是一个潮汐时刻（高低潮）。
type Tide struct {
	// FXTime 是潮汐时刻。
	FXTime string `json:"fxTime"`
	// Height 是潮位高度（米）。
	//
	// 上游给的是字符串，这里保持字符串 —— 小数位数是上游的表达精度，
	// 转成 float64 会丢掉它。
	Height string `json:"height"`
	// Type 是潮汐类型："H"（高潮）/ "L"（低潮）。
	Type string `json:"type"`
}

// TideHourly 是一个小时的潮位。
type TideHourly struct {
	// FXTime 是时刻。
	FXTime string `json:"fxTime"`
	// Height 是潮位高度（米）。
	Height string `json:"height"`
}
