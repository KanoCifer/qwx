package qwx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// 预警接口的缓存时长：上游 5 分钟更新一次预警。
const alertCacheTTL = 5 * time.Minute

// Alert 是新一代预警条目。
type Alert struct {
	// ID 是预警 ID。
	ID string `json:"id"`
	// SenderName 是发布机构，如 "临桂区气象台"。
	SenderName string `json:"senderName"`
	// IssuedTime 是发布时间。
	IssuedTime string `json:"issuedTime"`
	// MessageType 是消息类型。
	MessageType AlertMessageType `json:"messageType"`
	// EventType 是事件类型。
	EventType AlertEventType `json:"eventType"`
	// Urgency 是紧急程度。可能为 nil。
	Urgency *string `json:"urgency,omitempty"`
	// Severity 是严重程度。可能为 nil。
	Severity *string `json:"severity,omitempty"`
	// Certainty 是确定性。可能为 nil。
	Certainty *string `json:"certainty,omitempty"`
	// Icon 是预警图标代码。
	Icon string `json:"icon"`
	// Color 是预警颜色。
	Color AlertColor `json:"color"`
	// EffectiveTime 是生效时间。
	EffectiveTime string `json:"effectiveTime"`
	// OnsetTime 是开始时间。
	OnsetTime string `json:"onsetTime"`
	// ExpireTime 是失效时间。
	ExpireTime string `json:"expireTime"`
	// Headline 是预警标题。
	Headline string `json:"headline"`
	// Description 是预警描述全文。
	Description string `json:"description"`
	// Criteria 是预警的触发标准。
	Criteria string `json:"criteria"`
	// ResponseTypes 是建议的响应措施。
	ResponseTypes []string `json:"responseTypes"`
	// Instruction 是防御指南。
	Instruction string `json:"instruction"`
}

// AlertMessageType 是预警消息类型。
type AlertMessageType struct {
	// Code 是消息类型代码，如 "update"、"alert"。
	Code string `json:"code"`
	// Supersedes 是本条预警更新/替代的历史预警 ID。
	Supersedes []string `json:"supersedes"`
}

// AlertEventType 是预警事件类型。
type AlertEventType struct {
	// Name 是事件名称，如 "大风"。
	Name string `json:"name"`
	// Code 是事件代码，如 "1006"。
	//
	// 官方会新增代码，不要依赖枚举的完备性。
	Code string `json:"code"`
}

// AlertColor 是预警颜色。
type AlertColor struct {
	// Code 是颜色名，如 "blue"。
	Code string `json:"code"`
	// Red 是红通道。
	Red int `json:"red"`
	// Green 是绿通道。
	Green int `json:"green"`
	// Blue 是蓝通道。
	Blue int `json:"blue"`
	// Alpha 是透明度。
	Alpha int `json:"alpha"`
}

// AlertsResult 是预警查询的结果。
type AlertsResult struct {
	// Metadata 是元数据。
	Metadata Metadata `json:"metadata"`
	// Alerts 是预警列表。
	//
	// 没有预警时上游返回空列表（metadata.zeroResult 为 true）。
	Alerts []Alert `json:"alerts"`
}

// Alerts 查询坐标点上的预警。
//
// 调 GET /weatheralert/v1/current/{latitude}/{longitude}。
//
// 缓存键：`qwx:alert:v1:<lat>,<lon>` 加查询参数后缀。
func (c *Client) Alerts(ctx context.Context, loc Location, opts WeatherOptions) (*AlertsResult, error) {
	lat, lon, err := loc.Path()
	if err != nil {
		return nil, err
	}
	var out AlertsResult
	err = c.do(ctx, request{
		method: "GET",
		path:   "/weatheralert/v1/current/" + lat + "/" + lon,
		query:  opts.params(),
	}, "qwx:alert:v1:"+formatCoord(loc.Lat)+","+formatCoord(loc.Lon)+opts.cacheSuffix(), alertCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Warning 是旧一代预警条目（/v7/warning/now）。
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

// Warnings 查询 location 上的旧一代预警。
//
// 调 GET /v7/warning/now?location=...。
//
// 与 [Client.Alerts] 的区别：v7 版本认 LocationID 与地名，新一代只认坐标。
// 两者数据结构完全不同，按需选择。
//
// 缓存键：`qwx:warning:v7:<location>` 加语言后缀。
func (c *Client) Warnings(ctx context.Context, loc Location, lang string) (*WarningsResult, error) {
	q := url.Values{}
	if lang != "" {
		q.Set("lang", lang)
	}
	q.Set("location", loc.Query())

	var out WarningsResult
	err := c.do(ctx, request{
		method: "GET",
		path:   "/v7/warning/now",
		query:  q,
	}, "qwx:warning:v7:"+loc.Query()+langSuffix(lang), alertCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
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

// Tide 查询潮汐。
//
// 调 GET /v7/ocean/tide?location=...&date=...。
//
// location 是潮汐站 ID（如 "P2352"），不是坐标、不是 LocationID。
// 用 [Client.LookupPOI] 配 [POITSTA] 可以检索到本站点。
//
// date 是潮汐站当地日期，格式 "YYYYMMDD"。
//
// **日期必须是潮汐站当地的当天**：潮汐窗口的下界是当地当天，
// 用 UTC 日期会在北京时间 00:00–08:00 要到"昨天"，上游会回 400。
//
// 缓存键：`qwx:tide:<location>:<date>`。
func (c *Client) Tide(ctx context.Context, stationID, date string) (*TideResult, error) {
	if stationID == "" {
		return nil, fmt.Errorf("%w: empty tide station ID", ErrInvalidParameter)
	}
	if err := validateDate(date); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("location", stationID)
	q.Set("date", date)

	var out TideResult
	err := c.do(ctx, request{
		method: "GET",
		path:   "/v7/ocean/tide",
		query:  q,
	}, "qwx:tide:"+stationID+":"+date, forecastCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// validateDate 校验日期格式为 YYYYMMDD。
func validateDate(date string) error {
	if len(date) != 8 {
		return fmt.Errorf("%w: date %q must be in YYYYMMDD format", ErrInvalidParameter, date)
	}
	for i := 0; i < len(date); i++ {
		if date[i] < '0' || date[i] > '9' {
			return fmt.Errorf("%w: date %q must be in YYYYMMDD format", ErrInvalidParameter, date)
		}
	}
	return nil
}

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

// Indices 查询生活指数。
//
// 调 GET /v7/indices/{days}d?type={type}&location=...。
//
// 上游至今只有 v7 版本，没有新一代（实测 /weather/v1/indices/* 与
// /v1/indices/* 均 404）。
//
// days 只支持 1 与 3 —— 实测 2d / 7d / 10d / 15d / 30d 全部 404。
// type 取 0~16，0 表示全部。
//
// loc 必须是坐标对：上游对裸经度（如 "121.47"）一律回 400。
//
// 缓存键：`qwx:indices:<location>:t<type>:d<days>`。
func (c *Client) Indices(ctx context.Context, loc Location, typ, days int, lang string) (*IndicesResult, error) {
	if typ < IndicesTypeAll || typ > IndicesMaxType {
		return nil, fmt.Errorf("%w: indices type %d out of range [0,%d]", ErrInvalidParameter, typ, IndicesMaxType)
	}
	if days != 1 && days != 3 {
		return nil, fmt.Errorf("%w: indices days %d is not supported (only 1 and 3)", ErrInvalidParameter, days)
	}
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: indices requires coordinates", ErrInvalidLocation)
	}
	q := url.Values{}
	q.Set("location", loc.Query())
	q.Set("type", strconv.Itoa(typ))
	if lang != "" {
		q.Set("lang", lang)
	}

	var out IndicesResult
	err := c.do(ctx, request{
		method: "GET",
		path:   "/v7/indices/" + strconv.Itoa(days) + "d",
		query:  q,
	}, "qwx:indices:"+loc.Query()+":t"+strconv.Itoa(typ)+":d"+strconv.Itoa(days)+langSuffix(lang), forecastCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
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

// Minutely 查询分钟级降水预报。
//
// 调 GET /v7/minutely/5m?location=...。
//
// 上游至今只有 v7 版本，没有新一代。
//
// 三条硬约束（官方文档 + 实测）：
//
//  1. **只认坐标**：location 必须是 "经度,纬度"，不认 LocationID 也不认地名。
//  2. **只覆盖中国区域**：境外坐标上游没有数据。
//  3. **粒度是 5 分钟**：未来 2 小时共 24 条，上游每 5 分钟更新一次。
//
// 缓存键：`qwx:minutely:<lon>,<lat>`。TTL 与上游 5 分钟的更新节奏对齐。
func (c *Client) Minutely(ctx context.Context, loc Location, lang string) (*MinutelyResult, error) {
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: minutely forecast requires coordinates", ErrInvalidLocation)
	}
	q := url.Values{}
	q.Set("location", loc.Query())
	if lang != "" {
		q.Set("lang", lang)
	}

	var out MinutelyResult
	err := c.do(ctx, request{
		method: "GET",
		path:   "/v7/minutely/5m",
		query:  q,
	}, "qwx:minutely:"+loc.Query()+langSuffix(lang), 5*time.Minute, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
