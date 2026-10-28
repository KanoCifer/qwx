package alert

import (
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
	v1 "github.com/KanoCifer/qwx/v1"
)

// 预警端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// Endpoint 是预警实况端点。
	Endpoint = qwx.Endpoint("weatheralert/v1/current")
)

// 预警接口的默认缓存时长：上游 5 分钟更新一次预警。
const cacheTTL = 5 * time.Minute

// Options 是预警接口的可选参数。
//
// 它是 [github.com/KanoCifer/qwx/v1.Options] 的别名：天气与预警两个端点族
// 接受同一组可选参数，类型放在代际共享包里。
type Options = v1.Options

// params 是影响结果的查询参数。
//
// 同一份参数既生成上游查询串，也生成缓存键后缀 —— 新增参数时不会漏进键。
func params(o Options) cachekey.Params {
	return cachekey.Params{Lang: o.Lang.API, LocalTime: o.LocalTime}
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
