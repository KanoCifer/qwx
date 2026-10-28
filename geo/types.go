package geo

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Options 是 GeoAPI 的可选参数。
type Options struct {
	// Lang 是语言代码。留空用上游默认。
	Lang string
	// Number 是返回条数上限。0 用上游默认（10）。
	Number int
}

// params 转换成查询参数。
func (o Options) params() url.Values {
	q := url.Values{}
	if o.Lang != "" {
		q.Set("lang", o.Lang)
	}
	if o.Number > 0 {
		q.Set("number", strconv.Itoa(o.Number))
	}
	return q
}

// cacheSuffix 是 Options 对缓存键的影响部分。
func (o Options) cacheSuffix() string {
	suffix := cachekey.Lang(o.Lang)
	if o.Number > 0 {
		suffix += ":n=" + strconv.Itoa(o.Number)
	}
	return suffix
}

// POIRangeOptions 是 POI 范围检索的可选参数。
type POIRangeOptions struct {
	// Lang 是语言代码。
	Lang string
	// Number 是返回条数上限。0 用上游默认（10）。
	Number int
	// Radius 是搜索半径（公里）。0 用上游默认（5）。
	Radius int
}

// POIType 是 POI 检索的类型。
type POIType string

// POI 检索支持的类型。
const (
	// POIScenic 景点。
	POIScenic POIType = "scenic"
	// POICSTA 综合观测站。
	POICSTA POIType = "CSTA"
	// POITSTA 潮汐观测站。
	POITSTA POIType = "TSTA"
)

// Place 是一个地点条目。
//
// GeoAPI 的几个检索接口（城市检索、POI 检索、热门城市）返回的条目
// 结构一致，用同一个类型。
type Place struct {
	// ID 是 LocationID。
	ID string `json:"id"`
	// Name 是地名（如 "东城"）。
	Name string `json:"name"`
	// Lat 是纬度。
	Lat string `json:"lat"`
	// Lon 是经度。
	Lon string `json:"lon"`
	// Adm1 是一级行政区（省/直辖市），如 "北京市"。
	Adm1 string `json:"adm1"`
	// Adm2 是二级行政区（城市），如 "北京"。
	Adm2 string `json:"adm2"`
	// Country 是国家，如 "中国"。
	Country string `json:"country"`
	// TZ 是时区，如 "Asia/Shanghai"。
	TZ string `json:"tz"`
	// UTCOffset 是 UTC 偏移，如 "+08:00"。
	UTCOffset string `json:"utcOffset"`
	// IsDst 是是否夏令时（"0" / "1"）。
	IsDst string `json:"isDst"`
	// Type 是地点类型，如 "city"、"scenic"。
	Type string `json:"type"`
	// Rank 是地点排名，数字越小越重要。
	Rank string `json:"rank"`
	// FXLink 是该地点在和风天气官网的链接。
	FXLink string `json:"fxLink"`
}

// DisplayName 拼出"城市 + 区县"形式的地点名。
//
// 上游的字段分工是 **name=区县、adm2=城市、adm1=省/直辖市**，所以
// "城市+区县" 就是 adm2 + name —— "北京东城"、"杭州西湖"、"舟山定海"。
//
// 城市级条目（LocationID 101010100 这类）name 与 adm2 同名，这时只留
// 一份，否则会拼出"北京北京"。
func (p Place) DisplayName() string {
	name := strings.TrimSpace(p.Name)
	adm2 := strings.TrimSpace(p.Adm2)
	switch {
	case name == "":
		return adm2
	case adm2 == "":
		return name
	case name == adm2:
		return name
	case len(name) > len(adm2) && name[:len(adm2)] == adm2:
		// name 已经带上市名，不重复拼。
		return name
	default:
		return adm2 + name
	}
}

// Coordinates 解析条目里的经纬度。
//
// 上游把 lat/lon 给成字符串，这里转成 float64。
func (p Place) Coordinates() (lat, lon float64, err error) {
	loc, err := qwx.ParseLocation(p.Lon + "," + p.Lat)
	if err != nil {
		return 0, 0, err
	}
	return loc.Lat, loc.Lon, nil
}

// CityResult 是城市检索的结果。
type CityResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// Locations 是匹配到的地点。
	Locations []Place `json:"location"`
}

// POIResult 是 POI 检索的结果。
type POIResult struct {
	// Code 是上游状态码。
	Code string `json:"code"`
	// POIs 是匹配到的 POI。
	POIs []Place `json:"poi"`
}

// RangeResult 是 POI 范围检索的结果。
type RangeResult struct {
	// Code 是上游状态码。
	Code string `json:"code"`
	// POIs 是范围内的 POI。
	POIs []Place `json:"poi"`
}

// TopCityResult 是热门城市检索的结果。
type TopCityResult struct {
	// Code 是上游状态码。
	Code string `json:"code"`
	// Cities 是热门城市列表。
	Cities []Place `json:"topCityList"`
}
