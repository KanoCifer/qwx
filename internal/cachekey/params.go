package cachekey

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/KanoCifer/qwx/internal/upstream"
)

// Params 是一组影响结果的查询参数。
//
// 同一份 Params 同时决定上游查询串与缓存键后缀：漏进键的参数不会出现，
// 键里也不会多出上游没收到的参数。
//
// 参数按归属分组：通用参数在顶层，只服务某个资源族的参数放进该族的
// 子结构。加一个 GeoAPI 专有的参数时，动的是 Geo，不是所有族共用的袋子。
//
// 空值语义：
//
//   - Lang 为空：用上游默认，不上报也不进键。
//   - LocalTime 为 false：不请求当地时间，不上报也不进键。
//   - Number 为 0：用上游默认条数，不上报也不进键。
type Params struct {
	// Lang 是语言代码，如 "zh"、"en"。
	Lang string
	// LocalTime 是否请求当地时间。
	//
	// 新一代接口默认给 UTC，不传 localTime 时间会整整差 8 小时。
	LocalTime bool
	// Number 是返回条数上限。0 表示用上游默认。
	Number int
	// Geo 是 GeoAPI 专有的参数。
	Geo GeoParams
	// Solar 是太阳辐射预报专有的参数。
	Solar SolarParams
	// Console 是控制台专有的参数。
	Console ConsoleParams
}

// GeoParams 是 GeoAPI 专有的查询参数。
type GeoParams struct {
	// Radius 是搜索半径（公里）。
	//
	// nil 表示端点没有半径参数。非 nil 时半径进键；半径为 0 时不发给上游
	// （上游按自己的默认值处理），但仍然进键。
	Radius *int
	// Adm 是上级行政区划的 LocationID，如 "101010100"。
	//
	// 只有 GeoAPI 的城市检索认这个参数。
	Adm string
	// Range 是搜索范围，如 "cn"、"world"。
	//
	// 只有 GeoAPI 的城市检索与热门城市检索认这个参数。
	Range string
	// City 是限定城市，如 "杭州"。
	//
	// 只有 GeoAPI 的 POI 检索认这个参数。
	City string
}

// SolarParams 是太阳辐射预报专有的查询参数。
type SolarParams struct {
	// Interval 是时间间隔分钟数，如 15、30、60。0 表示用上游默认。
	Interval int
	// Tilt 是光伏系统的倾斜角，0~90。
	//
	// nil 表示不传。非 nil 时原样发给上游并进键，含 0 —— 0 是合法取值，
	// 不是"没传"。
	Tilt *int
	// Azimuth 是光伏系统的方位角，0~359，0 表示正北。
	//
	// nil 表示不传。非 nil 时原样发给上游并进键，0 是合法取值。
	Azimuth *int
	// Extra 是额外信息，如 "weather"、"poa"，多项用逗号分隔。
	Extra string
}

// ConsoleParams 是控制台专有的查询参数。
type ConsoleParams struct {
	// Project 是项目 ID。
	Project string
	// Credential 是凭据 ID，与 Project 互斥。
	Credential string
}

// V1Params 构造新一代端点族共用的参数集。
//
// 天气与预警两个端点族接受同一组可选参数：语言与当地时间。映射只在这里
// 写一次 —— 新增第三个共用这组参数的端点族时也走这里。
func V1Params(lang string, localTime bool) Params {
	return Params{Lang: lang, LocalTime: localTime}
}

// Query 生成上游查询参数。
func (p Params) Query() url.Values {
	q := url.Values{}
	if p.Lang != "" {
		q.Set(upstream.Lang, p.Lang)
	}
	if p.LocalTime {
		q.Set(upstream.LocalTime, "true")
	}
	if p.Number > 0 {
		q.Set(upstream.Number, strconv.Itoa(p.Number))
	}
	if p.Geo.Radius != nil && *p.Geo.Radius > 0 {
		q.Set(upstream.Radius, strconv.Itoa(*p.Geo.Radius))
	}
	if p.Geo.Adm != "" {
		q.Set(upstream.Adm, p.Geo.Adm)
	}
	if p.Geo.Range != "" {
		q.Set(upstream.Range, p.Geo.Range)
	}
	if p.Geo.City != "" {
		q.Set(upstream.City, p.Geo.City)
	}
	if p.Solar.Interval > 0 {
		q.Set(upstream.Interval, strconv.Itoa(p.Solar.Interval))
	}
	if p.Solar.Tilt != nil {
		q.Set(upstream.Tilt, strconv.Itoa(*p.Solar.Tilt))
	}
	if p.Solar.Azimuth != nil {
		q.Set(upstream.Azimuth, strconv.Itoa(*p.Solar.Azimuth))
	}
	if p.Solar.Extra != "" {
		q.Set(upstream.Extra, p.Solar.Extra)
	}
	if p.Console.Project != "" {
		q.Set(upstream.Project, p.Console.Project)
	}
	if p.Console.Credential != "" {
		q.Set(upstream.Credential, p.Console.Credential)
	}
	return q
}

// suffix 是参数对缓存键的影响片段。
//
// 顺序固定：语言 → 本地时间 → 条数 → 半径 → 行政区划 → 搜索范围 → 城市
// → 时间间隔 → 倾斜角 → 方位角 → 额外信息 → 项目 → 凭据。已发布的键字面量
// 按这个顺序排，改动会让旧缓存条目失效。
//
// 族专有的参数追加在通用参数之后：老调用方不设它们，键的字节与发布时
// 完全一致。
func (p Params) suffix() string {
	var b strings.Builder
	if p.Lang != "" {
		b.WriteString(":" + upstream.Lang + "=" + p.Lang)
	}
	if p.LocalTime {
		b.WriteString(":localTime=1")
	}
	if p.Number > 0 {
		b.WriteString(":n=" + strconv.Itoa(p.Number))
	}
	if p.Geo.Radius != nil {
		b.WriteString(":r=" + strconv.Itoa(*p.Geo.Radius))
	}
	if p.Geo.Adm != "" {
		b.WriteString(":" + upstream.Adm + "=" + p.Geo.Adm)
	}
	if p.Geo.Range != "" {
		b.WriteString(":" + upstream.Range + "=" + p.Geo.Range)
	}
	if p.Geo.City != "" {
		b.WriteString(":" + upstream.City + "=" + p.Geo.City)
	}
	if p.Solar.Interval > 0 {
		b.WriteString(":" + upstream.Interval + "=" + strconv.Itoa(p.Solar.Interval))
	}
	if p.Solar.Tilt != nil {
		b.WriteString(":" + upstream.Tilt + "=" + strconv.Itoa(*p.Solar.Tilt))
	}
	if p.Solar.Azimuth != nil {
		b.WriteString(":" + upstream.Azimuth + "=" + strconv.Itoa(*p.Solar.Azimuth))
	}
	if p.Solar.Extra != "" {
		b.WriteString(":" + upstream.Extra + "=" + p.Solar.Extra)
	}
	if p.Console.Project != "" {
		b.WriteString(":" + upstream.Project + "=" + p.Console.Project)
	}
	if p.Console.Credential != "" {
		b.WriteString(":" + upstream.Credential + "=" + p.Console.Credential)
	}
	return b.String()
}
