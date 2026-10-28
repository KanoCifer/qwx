// Package geo 提供 GeoAPI 城市/POI 检索（/geo/v2/*）。
//
// GeoAPI 是唯一认地名的接口族：坐标对、LocationID、地名三种位置形态都收。
package geo

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// GeoAPI 的缓存时长：地理信息几乎不变，缓存久一点。
const cacheTTL = 24 * time.Hour

// Client 是 GeoAPI 的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造 GeoAPI 的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("geo: nil base client")
	}
	return &Client{base: base}
}

// LookupCity 检索城市。
//
// 调 GET /geo/v2/city/lookup?location=...。
//
// loc 可以是坐标对、LocationID 或地名 —— 这是唯一认地名的接口。
// 也支持坐标反查（传 "经度,纬度" 拿该点所在的城市）。
//
// 缓存键：`qwx:geo:city:<location>` 加查询参数后缀。
func (c *Client) LookupCity(ctx context.Context, loc qwx.Location, opts Options) (*CityResult, error) {
	q := opts.params()
	q.Set("location", loc.Query())

	var out CityResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/geo/v2/city/lookup",
		Query:    q,
		CacheKey: "qwx:geo:city:" + loc.Query() + opts.cacheSuffix(),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
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
func (c *Client) LookupPOI(ctx context.Context, loc qwx.Location, typ POIType, opts Options) (*POIResult, error) {
	switch typ {
	case POIScenic, POICSTA, POITSTA:
	default:
		return nil, fmt.Errorf("%w: unsupported POI type %q", qwx.ErrInvalidParameter, typ)
	}
	q := opts.params()
	q.Set("location", loc.Query())
	q.Set("type", string(typ))

	var out POIResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/geo/v2/poi/lookup",
		Query:    q,
		CacheKey: "qwx:geo:poi:" + string(typ) + ":" + loc.Query() + opts.cacheSuffix(),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
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
func (c *Client) POIRange(ctx context.Context, loc qwx.Location, typ POIType, opts POIRangeOptions) (*RangeResult, error) {
	switch typ {
	case POIScenic, POICSTA, POITSTA:
	default:
		return nil, fmt.Errorf("%w: unsupported POI type %q", qwx.ErrInvalidParameter, typ)
	}
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: POI range requires coordinates", qwx.ErrInvalidLocation)
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

	suffix := cachekey.Lang(opts.Lang)
	if opts.Number > 0 {
		suffix += ":n=" + strconv.Itoa(opts.Number)
	}
	suffix += ":r=" + strconv.Itoa(opts.Radius)

	var out RangeResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/geo/v2/poi/range",
		Query:    q,
		CacheKey: "qwx:geo:poirange:" + string(typ) + ":" + loc.Query() + suffix,
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// TopCities 检索热门城市。
//
// 调 GET /geo/v2/city/top。
//
// 缓存键：`qwx:geo:topcity` 加查询参数后缀。
func (c *Client) TopCities(ctx context.Context, opts Options) (*TopCityResult, error) {
	var out TopCityResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/geo/v2/city/top",
		Query:    opts.params(),
		CacheKey: "qwx:geo:topcity" + opts.cacheSuffix(),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
