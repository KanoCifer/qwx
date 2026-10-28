package qwx

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GeoAPI 的缓存时长：地理信息几乎不变，缓存久一点。
const geoCacheTTL = 24 * time.Hour

// GeoOptions 是 GeoAPI 的可选参数。
type GeoOptions struct {
	// Lang 是语言代码。留空用上游默认。
	Lang string
	// Number 是返回条数上限。0 用上游默认（10）。
	Number int
}

// params 转换成查询参数。
func (o GeoOptions) params() url.Values {
	q := url.Values{}
	if o.Lang != "" {
		q.Set("lang", o.Lang)
	}
	if o.Number > 0 {
		q.Set("number", strconv.Itoa(o.Number))
	}
	return q
}

// GeoLocation 是一个地点条目。
//
// GeoAPI 的几个检索接口（城市检索、POI 检索、热门城市）返回的条目
// 结构一致，用同一个类型。
type GeoLocation struct {
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
func (l GeoLocation) DisplayName() string {
	name := strings.TrimSpace(l.Name)
	adm2 := strings.TrimSpace(l.Adm2)
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
func (l GeoLocation) Coordinates() (lat, lon float64, err error) {
	return parseCoordinatePair(l.Lon + "," + l.Lat)
}

// CityLookupResult 是城市检索的结果。
type CityLookupResult struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// Locations 是匹配到的地点。
	Locations []GeoLocation `json:"location"`
}

// LookupCity 检索城市。
//
// 调 GET /geo/v2/city/lookup?location=...。
//
// loc 可以是坐标对、LocationID 或地名 —— 这是唯一认地名的接口。
// 也支持坐标反查（传 "经度,纬度" 拿该点所在的城市）。
//
// 缓存键：`qwx:geo:city:<location>` 加查询参数后缀。
func (c *Client) LookupCity(ctx context.Context, loc Location, opts GeoOptions) (*CityLookupResult, error) {
	q := opts.params()
	q.Set("location", loc.Query())

	var out CityLookupResult
	err := c.do(ctx, request{
		method: "GET",
		path:   "/geo/v2/city/lookup",
		query:  q,
	}, "qwx:geo:city:"+loc.Query()+geoCacheSuffix(opts), geoCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
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

// POILookupResult 是 POI 检索的结果。
type POILookupResult struct {
	// Code 是上游状态码。
	Code string `json:"code"`
	// POIs 是匹配到的 POI。
	POIs []GeoLocation `json:"poi"`
}

// LookupPOI 检索 POI。
//
// 调 GET /geo/v2/poi/lookup?location=...&type=...。
//
// loc 可以是坐标对、LocationID 或地名。typ 必填。
//
// **上游在该类型附近没有数据时回 400 "No Such Location"，而不是空列表** ——
// 这是实测行为：内陆城市（如北京）查 TSTA（潮汐站）与 CSTA（观测站）
// 都会 400，沿海城市（如舟山 P2419、高雄）才查得到。调用方要按错误处理，
// 不能假设"查不到就是空"。
//
// 缓存键：`qwx:geo:poi:<type>:<location>` 加查询参数后缀。
func (c *Client) LookupPOI(ctx context.Context, loc Location, typ POIType, opts GeoOptions) (*POILookupResult, error) {
	switch typ {
	case POIScenic, POICSTA, POITSTA:
	default:
		return nil, fmt.Errorf("%w: unsupported POI type %q", ErrInvalidParameter, typ)
	}
	q := opts.params()
	q.Set("location", loc.Query())
	q.Set("type", string(typ))

	var out POILookupResult
	err := c.do(ctx, request{
		method: "GET",
		path:   "/geo/v2/poi/lookup",
		query:  q,
	}, "qwx:geo:poi:"+string(typ)+":"+loc.Query()+geoCacheSuffix(opts), geoCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// POIRangeResult 是 POI 范围检索的结果。
type POIRangeResult struct {
	// Code 是上游状态码。
	Code string `json:"code"`
	// POIs 是范围内的 POI。
	POIs []GeoLocation `json:"poi"`
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

// POIRange 检索指定坐标附近的 POI。
//
// 调 GET /geo/v2/poi/range?location=...&type=...&radius=...。
//
// loc 必须是坐标对 —— 范围检索以点为中心，不认 LocationID。
//
// 与 [Client.LookupPOI] 一样，上游在该类型附近无数据时回 400
// "No Such Location"，不是空列表。
//
// 缓存键：`qwx:geo:poirange:<type>:<location>:r<radius>` 加查询参数后缀。
func (c *Client) POIRange(ctx context.Context, loc Location, typ POIType, opts POIRangeOptions) (*POIRangeResult, error) {
	switch typ {
	case POIScenic, POICSTA, POITSTA:
	default:
		return nil, fmt.Errorf("%w: unsupported POI type %q", ErrInvalidParameter, typ)
	}
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: POI range requires coordinates", ErrInvalidLocation)
	}
	q := url.Values{}
	if opts.Lang != "" {
		q.Set("lang", opts.Lang)
	}
	if opts.Number > 0 {
		q.Set("number", strconv.Itoa(opts.Number))
	}
	if opts.Radius > 0 {
		q.Set("radius", strconv.Itoa(opts.Radius))
	}
	q.Set("location", loc.Query())
	q.Set("type", string(typ))

	suffix := ""
	if opts.Lang != "" {
		suffix += ":lang=" + opts.Lang
	}
	if opts.Number > 0 {
		suffix += ":n=" + strconv.Itoa(opts.Number)
	}
	suffix += ":r=" + strconv.Itoa(opts.Radius)

	var out POIRangeResult
	err := c.do(ctx, request{
		method: "GET",
		path:   "/geo/v2/poi/range",
		query:  q,
	}, "qwx:geo:poirange:"+string(typ)+":"+loc.Query()+suffix, geoCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// TopCityResult 是热门城市检索的结果。
type TopCityResult struct {
	// Code 是上游状态码。
	Code string `json:"code"`
	// Cities 是热门城市列表。
	Cities []GeoLocation `json:"topCityList"`
}

// TopCities 检索热门城市。
//
// 调 GET /geo/v2/city/top。
//
// 缓存键：`qwx:geo:topcity` 加查询参数后缀。
func (c *Client) TopCities(ctx context.Context, opts GeoOptions) (*TopCityResult, error) {
	var out TopCityResult
	err := c.do(ctx, request{
		method: "GET",
		path:   "/geo/v2/city/top",
		query:  opts.params(),
	}, "qwx:geo:topcity"+geoCacheSuffix(opts), geoCacheTTL, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// geoCacheSuffix 是 GeoOptions 对缓存键的影响部分。
func geoCacheSuffix(opts GeoOptions) string {
	suffix := ""
	if opts.Lang != "" {
		suffix += ":lang=" + opts.Lang
	}
	if opts.Number > 0 {
		suffix += ":n=" + strconv.Itoa(opts.Number)
	}
	return suffix
}
