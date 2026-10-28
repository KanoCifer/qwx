package qwx

// 本文件是新一代接口（/weather/v1/*、/airquality/v1/*、/weatheralert/v1/*）
// 共用的基础结构。
//
// 官方文档强调「不要假设」：字段可能缺失，枚举值可能新增，
// 所以这里的可选字段一律用指针，缺失与零值可区分。

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

// Value 是带单位的数值。
//
// 新一代接口把数值与单位拆开，不再像 v7 那样把单位拼进字符串。
type Value struct {
	// Value 是数值。
	Value float64 `json:"value"`
	// Unit 是单位，如 "°C"、"m/s"、"mm"、"hPa"、"m"。
	Unit string `json:"unit"`
}

// Condition 是天气现象。
type Condition struct {
	// Text 是天气现象的文字描述，随 lang 变化。
	Text string `json:"text"`
	// Code 是天气现象代码，如 "100"（晴）、"300"（小雨）。
	//
	// 官方会新增代码，不要依赖枚举的完备性。
	Code string `json:"code"`
}

// WindDirection 是风向。
type WindDirection struct {
	// Degree 是风向角度（0~360，0 为北）。
	Degree float64 `json:"degree"`
	// Compass 是 16 方位英文代码（如 "sw"、"wnw"），不是中文。
	Compass string `json:"compass"`
}

// Wind 是风速与风向。
type Wind struct {
	// Direction 是风向。
	Direction WindDirection `json:"direction"`
	// Speed 是风速。
	Speed Value `json:"speed"`
	// Scale 是风力等级（蒲福风级）。
	Scale float64 `json:"scale"`
}

// WindGust 是阵风。
type WindGust struct {
	// Value 是阵风风速。
	Value float64 `json:"value"`
	// Unit 是单位。
	Unit string `json:"unit"`
}

// Precipitation 是降水情况。
type Precipitation struct {
	// Amount 是降水量。
	Amount Value `json:"amount"`
	// Intensity 是降水强度。
	Intensity Value `json:"intensity"`
	// Probability 是降水概率（0~1）。
	//
	// 实时天气接口没有这个字段，逐小时预报才有。用指针区分「没有」
	// 与「概率为 0」。
	Probability *float64 `json:"probability,omitempty"`
	// Type 是降水类型："none" / "rain" / "snow"。
	Type string `json:"type"`
}

// Pressure 是气压。
type Pressure struct {
	// Value 是气压值。
	Value float64 `json:"value"`
	// Unit 是单位（hPa）。
	Unit string `json:"unit"`
}

// Visibility 是能见度。
type Visibility struct {
	// Value 是能见度值。
	Value float64 `json:"value"`
	// Unit 是单位（m）。
	Unit string `json:"unit"`
}

// DewPoint 是露点温度。
type DewPoint struct {
	// Value 是露点温度。
	Value float64 `json:"value"`
	// Unit 是单位（°C）。
	Unit string `json:"unit"`
}

// Color 是颜色。
type Color struct {
	// Red 是红通道。
	Red int `json:"red"`
	// Green 是绿通道。
	Green int `json:"green"`
	// Blue 是蓝通道。
	Blue int `json:"blue"`
	// Alpha 是透明度。
	Alpha int `json:"alpha"`
}
