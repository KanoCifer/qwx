package alert

import (
	"net/url"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
	"github.com/KanoCifer/qwx/v1"
)

// 预警端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// Endpoint 是预警实况端点。
	Endpoint = qwx.Endpoint("weatheralert/v1/current")
)

// 预警接口的默认缓存时长：上游 5 分钟更新一次预警。
const cacheTTL = 5 * time.Minute

// Options 是预警接口的可选参数。
type Options struct {
	// Lang 是语言代码，如 "zh"、"en"。留空用上游默认。
	Lang string
	// LocalTime 是否返回查询地点的当地时间。
	//
	// 新一代接口默认给 UTC，不传 localTime 时间会整整差 8 小时。
	LocalTime bool
}

// params 转换成查询参数。
func (o Options) params() url.Values {
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
func (o Options) cacheSuffix() string {
	suffix := cachekey.Lang(o.Lang)
	if o.LocalTime {
		suffix += ":localTime=1"
	}
	return suffix
}

// Alert 是一条预警。
type Alert struct {
	// ID 是预警 ID。
	ID string `json:"id"`
	// SenderName 是发布机构，如 "临桂区气象台"。
	SenderName string `json:"senderName"`
	// IssuedTime 是发布时间。
	IssuedTime string `json:"issuedTime"`
	// MessageType 是消息类型。
	MessageType MessageType `json:"messageType"`
	// EventType 是事件类型。
	EventType EventType `json:"eventType"`
	// Urgency 是紧急程度。可能为 nil。
	Urgency *string `json:"urgency,omitempty"`
	// Severity 是严重程度。可能为 nil。
	Severity *string `json:"severity,omitempty"`
	// Certainty 是确定性。可能为 nil。
	Certainty *string `json:"certainty,omitempty"`
	// Icon 是预警图标代码。
	Icon string `json:"icon"`
	// Color 是预警颜色。
	Color Color `json:"color"`
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

// MessageType 是预警消息类型。
type MessageType struct {
	// Code 是消息类型代码，如 "update"、"alert"。
	Code string `json:"code"`
	// Supersedes 是本条预警更新/替代的历史预警 ID。
	Supersedes []string `json:"supersedes"`
}

// EventType 是预警事件类型。
type EventType struct {
	// Name 是事件名称，如 "大风"。
	Name string `json:"name"`
	// Code 是事件代码，如 "1006"。
	//
	// 官方会新增代码，不要依赖枚举的完备性。
	Code string `json:"code"`
}

// Color 是预警颜色。
type Color struct {
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

// Result 是预警查询的结果。
type Result struct {
	// Metadata 是元数据。
	Metadata v1.Metadata `json:"metadata"`
	// Alerts 是预警列表。
	//
	// 没有预警时上游返回空列表（metadata.zeroResult 为 true）。
	Alerts []Alert `json:"alerts"`
}
