package minutely

import (
	"time"

	"github.com/KanoCifer/qwx"
)

// 分钟级降水端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// Endpoint 是分钟级降水端点。
	Endpoint = qwx.Endpoint("v7/minutely/5m")
)

// 分钟级降水接口的默认缓存时长：上游 5 分钟更新一次。
const cacheTTL = 5 * time.Minute

// Result 是分钟级降水预报的结果。
type Result struct {
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
	// 上游给的是固定两位小数的字符串。这里保持字符串：空串表示"无此数据"，
	// 与 "0.00"（真的没下雨）不是一回事，转成数值会把两者压成同一个 0。
	Precip string `json:"precip"`
	// Type 是降水类型："rain" / "snow"。
	Type string `json:"type"`
}
