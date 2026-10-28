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
	return q
}

// suffix 是参数对缓存键的影响片段。
//
// 顺序固定：语言 → 本地时间 → 条数 → 半径。已发布的键字面量按这个
// 顺序排，改动会让旧缓存条目失效。
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
	return b.String()
}
