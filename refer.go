package qwx

// Refer 是数据来源与许可信息。
//
// 旧一代接口（/v7/*）与 GeoAPI 在响应里带一个 refer 对象，说明数据来源与
// 授权方式。新一代接口没有 refer —— 同样的意思由 metadata.attributions 表达。
//
// 展示这些数据时需要一并展示来源与许可声明。
type Refer struct {
	// Sources 是原始数据来源，或数据源说明。上游可能给空。
	Sources []string `json:"sources"`
	// License 是数据许可或版权声明。上游可能给空。
	License []string `json:"license"`
}

// MetaData 是控制台 API 返回的元数据块。
//
// 响应里的 metadata 对象说明本次响应的数据标签与来源署名。展示上游数据时
// 需要保留 [MetaData.Attributions]。
//
// 新一代接口用的是 [github.com/KanoCifer/qwx/v1.Metadata]：它多一个
// ZeroResult 字段，用来区分「没查」与「查了是空」。
type MetaData struct {
	// Tag 是本次响应的数据标签。
	Tag string `json:"tag"`
	// Attributions 是数据来源署名。展示数据时需要保留。
	Attributions []string `json:"attributions"`
}
