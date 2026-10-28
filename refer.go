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
