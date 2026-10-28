package geo

import (
	"strings"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// GeoAPI 端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// EndpointCity 是城市检索端点。
	EndpointCity = qwx.Endpoint("geo/v2/city/lookup")
	// EndpointPOI 是 POI 检索端点。
	EndpointPOI = qwx.Endpoint("geo/v2/poi/lookup")
	// EndpointPOIRange 是 POI 范围检索端点。
	EndpointPOIRange = qwx.Endpoint("geo/v2/poi/range")
	// EndpointTopCities 是热门城市端点。
	EndpointTopCities = qwx.Endpoint("geo/v2/city/top")
)

// GeoAPI 的默认缓存时长：地理信息几乎不变，缓存久一点。
const cacheTTL = 24 * time.Hour

// Options 是 GeoAPI 的可选参数。
//
// 四个检索端点接受的参数子集不同，各端点只上报自己认的参数：行政区划
// 只有城市检索认，搜索范围城市检索与热门城市认，限定城市只有 POI 检索认。
type Options struct {
	// Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
	Lang qwx.Lang
	// Number 是返回条数上限。0 用上游默认（10）。
	Number int
	// Adm 是上级行政区划的 LocationID，如 "101010100"。
	//
	// 用于排除重名城市或过滤结果。只有城市检索认这个参数。
	Adm string
	// Range 是搜索范围，用 ISO 3166 国家代码，如 "cn"、"world"。
	//
	// 留空时上游在所有城市里搜，"world" 才能查到境外城市。
	// 城市检索与热门城市认这个参数。
	Range string
	// City 是限定 POI 所在城市，城市名或 LocationID，如 "101010100"。
	//
	// 城市名是精准匹配，匹配不上上游返回空。只有 POI 检索认这个参数。
	City string
}

// cityParams 是城市检索用的查询参数。
//
// 同一份参数既生成上游查询串，也生成缓存键后缀 —— 新增参数时不会漏进键。
func (o Options) cityParams() cachekey.Params {
	return cachekey.Params{Lang: o.Lang.API, Number: o.Number, Adm: o.Adm, Range: o.Range}
}

// poiParams 是 POI 检索用的查询参数。
func (o Options) poiParams() cachekey.Params {
	return cachekey.Params{Lang: o.Lang.API, Number: o.Number, City: o.City}
}

// topParams 是热门城市用的查询参数。
func (o Options) topParams() cachekey.Params {
	return cachekey.Params{Lang: o.Lang.API, Number: o.Number, Range: o.Range}
}

// POIRangeOptions 是 POI 范围检索的可选参数。
type POIRangeOptions struct {
	// Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
	Lang qwx.Lang
	// Number 是返回条数上限。0 用上游默认（10）。
	Number int
	// Radius 是搜索半径（公里）。0 用上游默认（5）。
	Radius int
}

// poiRangeParams 是 POI 范围检索用的查询参数。
//
// 半径用指针传：半径为 0 与"没有半径参数"在键里要分开，而请求里都不
// 发 radius（上游按默认值处理）。
func (o POIRangeOptions) poiRangeParams() cachekey.Params {
	return cachekey.Params{Lang: o.Lang.API, Number: o.Number, Radius: &o.Radius}
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

// Refer 是数据来源与许可信息。
//
// 它是 [qwx.Refer] 的别名：GeoAPI 与旧一代接口的 refer 对象同形，
// 放根包由四族共用。保留这个别名是为了不改动已发布的 geo.Refer。
type Refer = qwx.Refer

// CityResult 是城市检索的结果。
type CityResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// Locations 是匹配到的地点。
	Locations []Place `json:"location"`
	// Refer 是数据来源与许可信息。
	Refer Refer `json:"refer"`
}

// POIResult 是 POI 检索的结果。
type POIResult struct {
	// Code 是上游状态码。
	Code string `json:"code"`
	// POIs 是匹配到的 POI。
	POIs []Place `json:"poi"`
	// Refer 是数据来源与许可信息。
	Refer Refer `json:"refer"`
}

// RangeResult 是 POI 范围检索的结果。
type RangeResult struct {
	// Code 是上游状态码。
	Code string `json:"code"`
	// POIs 是范围内的 POI。
	POIs []Place `json:"poi"`
	// Refer 是数据来源与许可信息。
	Refer Refer `json:"refer"`
}

// TopCityResult 是热门城市检索的结果。
type TopCityResult struct {
	// Code 是上游状态码。
	Code string `json:"code"`
	// Cities 是热门城市列表。
	Cities []Place `json:"topCityList"`
	// Refer 是数据来源与许可信息。
	Refer Refer `json:"refer"`
}
