package cachekey

import (
	"net/url"
	"strconv"
	"strings"
)

// Params 是一组影响结果的查询参数。
//
// 同一份 Params 同时决定上游查询串与缓存键后缀：漏进键的参数不会出现，
// 键里也不会多出上游没收到的参数。
//
// 空值语义：
//
//   - Lang 为空：用上游默认，不上报也不进键。
//   - LocalTime 为 false：不请求当地时间，不上报也不进键。
//   - Number 为 0：用上游默认条数，不上报也不进键。
//   - Radius 为 nil：端点没有半径参数。非 nil 时半径进键；半径为 0
//     只进键、不上报，上游按自己的默认值处理。
//   - Tilt 与 Azimuth 为 nil：端点没有这两个参数。非 nil 时进键并上报，
//     含 0 —— 0 是合法取值，不是"没传"。
type Params struct {
	// Lang 是语言代码，如 "zh"、"en"。
	Lang string
	// LocalTime 是否请求当地时间。
	//
	// 新一代接口默认给 UTC，不传 localTime 时间会整整差 8 小时。
	LocalTime bool
	// Number 是返回条数上限。0 表示用上游默认。
	Number int
	// Radius 是搜索半径（公里）。
	//
	// 用指针区分"端点没有半径参数"与"半径为 0"。
	Radius *int
	// Adm 是上级行政区划的 LocationID，如 "101010100"。
	//
	// 只有 GeoAPI 的城市检索认这个参数。
	Adm string
	// Range 是搜索范围，如 "cn"、"world"。
	//
	// 只有 GeoAPI 的城市检索认这个参数。
	Range string
	// City 是限定城市，如 "杭州"。
	//
	// 只有 GeoAPI 的 POI 检索认这个参数。
	City string
	// Interval 是时间间隔分钟数，如 15、30、60。0 表示用上游默认。
	//
	// 只有太阳辐射预报认这个参数。
	Interval int
	// Tilt 是光伏系统的倾斜角，0~90。
	//
	// 用指针区分"端点没有倾斜角参数"与"倾斜角为 0"。0 是合法取值：
	// 平铺的光伏板倾斜角就是 0，所以非 nil 时要上报。
	Tilt *int
	// Azimuth 是光伏系统的方位角，0~359，0 表示正北。
	//
	// 用指针区分"端点没有方位角参数"与"方位角为 0"。0 是合法取值。
	Azimuth *int
	// Extra 是额外信息，如 "weather"、"poa"，多项用逗号分隔。
	//
	// 只有太阳辐射预报认这个参数。
	Extra string
	// Project 是项目 ID。
	//
	// 只有控制台的请求量统计认这个参数。
	Project string
	// Credential 是凭据 ID。
	//
	// 只有控制台的请求量统计认这个参数，与 Project 互斥。
	Credential string
}

// Query 生成上游查询参数。
func (p Params) Query() url.Values {
	q := url.Values{}
	if p.Lang != "" {
		q.Set("lang", p.Lang)
	}
	if p.LocalTime {
		q.Set("localTime", "true")
	}
	if p.Number > 0 {
		q.Set("number", strconv.Itoa(p.Number))
	}
	if p.Radius != nil && *p.Radius > 0 {
		q.Set("radius", strconv.Itoa(*p.Radius))
	}
	if p.Adm != "" {
		q.Set("adm", p.Adm)
	}
	if p.Range != "" {
		q.Set("range", p.Range)
	}
	if p.City != "" {
		q.Set("city", p.City)
	}
	if p.Interval > 0 {
		q.Set("interval", strconv.Itoa(p.Interval))
	}
	if p.Tilt != nil {
		q.Set("tilt", strconv.Itoa(*p.Tilt))
	}
	if p.Azimuth != nil {
		q.Set("azimuth", strconv.Itoa(*p.Azimuth))
	}
	if p.Extra != "" {
		q.Set("extra", p.Extra)
	}
	if p.Project != "" {
		q.Set("project", p.Project)
	}
	if p.Credential != "" {
		q.Set("credential", p.Credential)
	}
	return q
}

// suffix 是参数对缓存键的影响片段。
//
// 顺序固定：语言 → 本地时间 → 条数 → 半径 → 行政区划 → 搜索范围 → 城市
// → 时间间隔 → 倾斜角 → 方位角 → 额外信息 → 项目 → 凭据。已发布的键字面量
// 按这个顺序排，改动会让旧缓存条目失效。
//
// 后九个字段是新加的，追加在已有字段之后：老调用方不设它们，键的字节
// 与发布时完全一致。
func (p Params) suffix() string {
	var b strings.Builder
	if p.Lang != "" {
		b.WriteString(":lang=" + p.Lang)
	}
	if p.LocalTime {
		b.WriteString(":localTime=1")
	}
	if p.Number > 0 {
		b.WriteString(":n=" + strconv.Itoa(p.Number))
	}
	if p.Radius != nil {
		b.WriteString(":r=" + strconv.Itoa(*p.Radius))
	}
	if p.Adm != "" {
		b.WriteString(":adm=" + p.Adm)
	}
	if p.Range != "" {
		b.WriteString(":range=" + p.Range)
	}
	if p.City != "" {
		b.WriteString(":city=" + p.City)
	}
	if p.Interval > 0 {
		b.WriteString(":interval=" + strconv.Itoa(p.Interval))
	}
	if p.Tilt != nil {
		b.WriteString(":tilt=" + strconv.Itoa(*p.Tilt))
	}
	if p.Azimuth != nil {
		b.WriteString(":azimuth=" + strconv.Itoa(*p.Azimuth))
	}
	if p.Extra != "" {
		b.WriteString(":extra=" + p.Extra)
	}
	if p.Project != "" {
		b.WriteString(":project=" + p.Project)
	}
	if p.Credential != "" {
		b.WriteString(":credential=" + p.Credential)
	}
	return b.String()
}
