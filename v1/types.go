// Package v1 承载新一代接口（/weather/v1/*、/airquality/v1/*、/weatheralert/v1/*）
// 跨端点族共用的基础类型。
//
// 这一代的字段形态与旧一代不同：数值与单位分离，湿度、云量、降水概率是
// 0~1 的比例而不是百分数。只有两个以上端点族都用到的类型才放在这里 ——
// 只服务单一端点族的类型留在该端点族自己的包里。
//
// 官方文档强调「不要假设」：字段可能缺失，枚举值可能新增，
// 所以这里的可选字段一律用指针，缺失与零值可区分。
package v1

// Metadata 是新一代接口统一的元数据块。
type Metadata struct {
	// Tag 是本次响应的数据标签。
	Tag string `json:"tag"`
	// Attributions 是数据来源署名。展示数据时需要保留。
	Attributions []string `json:"attributions"`
	// ZeroResult 表示上游确实查了但结果为空。
	//
	// 老于该字段的接口版本没有这个字段，此时为 false —— 无法区分
	// 「没查」与「查了是空」，调用方需要自己结合业务判断。
	ZeroResult *bool `json:"zeroResult,omitempty"`
}

// Options 是新一代端点共用的可选参数。
//
// 天气与预警两个端点族接受同一组参数，所以类型放在这里 —— 只服务单一
// 端点族的参数留在该端点族自己的包里。
type Options struct {
	// Lang 是语言代码，如 "zh"、"en"。留空用上游默认。
	Lang string
	// LocalTime 是否返回查询地点的当地时间。
	//
	// 新一代接口默认给 UTC（"2026-10-21T05:00Z"），不传 localTime
	// 时间会整整差 8 小时。除非明确要 UTC，否则应设为 true。
	LocalTime bool
}

// Value 是带单位的数值。
//
// 新一代接口把数值与单位拆开，不再像 v7 那样把单位拼进字符串。
type Value struct {
	// Value 是数值。
	Value float64 `json:"value"`
	// Unit 是单位，如 "°C"、"m/s"、"mm"、"hPa"、"m"。
	Unit string `json:"unit"`
}
