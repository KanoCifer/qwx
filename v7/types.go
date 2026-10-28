package v7

import (
	"time"

	"github.com/KanoCifer/qwx"
)

// 旧一代端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// EndpointWarnings 是旧一代预警端点。
	EndpointWarnings = qwx.Endpoint("v7/warning/now")
	// EndpointTide 是潮汐端点。
	EndpointTide = qwx.Endpoint("v7/ocean/tide")
	// EndpointIndices 是生活指数端点。
	EndpointIndices = qwx.Endpoint("v7/indices")
	// EndpointMinutely 是分钟级降水端点。
	EndpointMinutely = qwx.Endpoint("v7/minutely/5m")
)

// 旧一代接口的默认缓存时长。
//
// 预警与分钟级降水对齐上游 5 分钟的更新节奏，潮汐与生活指数 1 小时。
const (
	alertCacheTTL    = 5 * time.Minute
	minutelyCacheTTL = 5 * time.Minute
	forecastCacheTTL = time.Hour
)

// IndicesType 生活指数类型代码。
//
// 取值 1~16，具体含义见官方「天气指数」文档。
const (
	// IndicesTypeAll 取全部生活指数（type=0）。
	IndicesTypeAll = 0
	// IndicesTypeFishing 钓鱼指数（type=4）。
	IndicesTypeFishing = 4
	// IndicesMaxType 最大的指数类型代码。
	IndicesMaxType = 16
)

// Warning 是一条旧一代预警条目（/v7/warning/now）。
type Warning struct {
	// ID 是预警 ID。
	ID string `json:"id"`
	// Sender 是发布机构。
	Sender string `json:"sender"`
	// PubTime 是发布时间。
	PubTime string `json:"pubTime"`
	// Title 是预警标题。
	Title string `json:"title"`
	// StartTime 是开始时间。
	StartTime string `json:"startTime"`
	// EndTime 是结束时间。
	EndTime string `json:"endTime"`
	// Status 是状态，如 "active"。
	Status string `json:"status"`
	// Level 是预警级别。可能为空字符串。
	Level string `json:"level"`
	// Severity 是严重程度，如 "Minor"。
	Severity string `json:"severity"`
	// SeverityColor 是严重程度颜色，如 "Blue"。
	SeverityColor string `json:"severityColor"`
	// Type 是预警类型代码，如 "1006"。
	Type string `json:"type"`
	// TypeName 是预警类型名称，如 "大风"。
	TypeName string `json:"typeName"`
	// Urgency 是紧急程度。可能为空字符串。
	Urgency string `json:"urgency"`
	// Certainty 是确定性。可能为空字符串。
	Certainty string `json:"certainty"`
	// Text 是预警正文。
	Text string `json:"text"`
	// Related 是关联预警。可能为空字符串。
	Related string `json:"related"`
}

// WarningsResult 是旧一代预警查询的结果。
type WarningsResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// UpdateTime 是数据更新时间。
	UpdateTime string `json:"updateTime"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// Warnings 是预警列表。
	Warnings []Warning `json:"warning"`
}

// TideResult 是潮汐数据。
type TideResult struct {
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

// IndicesResult 是生活指数查询的结果。
type IndicesResult struct {
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

// MinutelyResult 是分钟级降水预报的结果。
type MinutelyResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// UpdateTime 是数据更新时间。
	UpdateTime string `json:"updateTime"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// Summary 是一句话摘要，如 "95分钟后雨就停了"。
	Summary string `json:"summary"`
	// Minutely 是逐 5 分钟的降水预报。
	Minutely []Minutely `json:"minutely"`
}

// Minutely 是一条分钟级降水预报。
type Minutely struct {
	// FXTime 是预报时刻。
	FXTime string `json:"fxTime"`
	// Precip 是降水量。
	//
	// 上游给的是固定两位小数的字符串。这里保持字符串：
	// 空串表示"无此数据"，与 "0.00"（真的没下雨）不是一回事，
	// 转成数值会把两者压成同一个 0。
	Precip string `json:"precip"`
	// Type 是降水类型："rain" / "snow"。
	Type string `json:"type"`
}
