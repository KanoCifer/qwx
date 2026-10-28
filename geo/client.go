// Package geo 提供 GeoAPI 城市/POI 检索（/geo/v2/*）。
//
// GeoAPI 是唯一认地名的接口族：坐标对、LocationID、地名三种位置形态都收。
package geo

import (
	"context"
	"fmt"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

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
// opts.Adm 限定上级行政区划（排除重名城市），opts.Range 限定搜索范围，
// 用 ISO 3166 国家代码，如 "cn"、"world"。留空时上游在所有城市里搜。
//
// 缓存键：`qwx:geo:city:<location>` 加查询参数后缀。
//
// 默认 TTL 24 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointCity] 覆盖。
func (c *Client) LookupCity(ctx context.Context, loc qwx.Location, opts Options) (*CityResult, error) {
	p := opts.cityParams()
	q := p.Query()
	q.Set("location", loc.Query())

	var out CityResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/geo/v2/city/lookup",
		Query:    q,
		Endpoint: EndpointCity,
		CacheKey: cityCacheKey(loc, p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// cityCacheKey 城市检索的缓存键。
func cityCacheKey(loc qwx.Location, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyGeo).
		Path("city", loc.Query()).
		Params(p).
		String()
}

// LookupPOI 检索 POI。
//
// 调 GET /geo/v2/poi/lookup?location=...&type=...。
//
// loc 可以是坐标对、LocationID 或地名。typ 必填。
//
// opts.City 限定 POI 所在城市，城市名或 LocationID。城市名是精准匹配，
// 匹配不上上游返回空。
//
// **上游在该类型附近没有数据时回 400 "No Such Location"，而不是空列表** ——
// 这是实测行为：内陆城市（如北京）查 TSTA（潮汐站）与 CSTA（观测站）
// 都会 400，沿海城市（如舟山 P2419、高雄）才查得到。调用方要按错误处理，
// 不能假设"查不到就是空"。
//
// 缓存键：`qwx:geo:poi:<type>:<location>` 加查询参数后缀。
//
// 默认 TTL 24 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointPOI] 覆盖。
func (c *Client) LookupPOI(ctx context.Context, loc qwx.Location, typ POIType, opts Options) (*POIResult, error) {
	switch typ {
	case POIScenic, POICSTA, POITSTA:
	default:
		return nil, fmt.Errorf("%w: unsupported POI type %q", qwx.ErrInvalidParameter, typ)
	}
	p := opts.poiParams()
	q := p.Query()
	q.Set("location", loc.Query())
	q.Set("type", string(typ))

	var out POIResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/geo/v2/poi/lookup",
		Query:    q,
		Endpoint: EndpointPOI,
		CacheKey: poiCacheKey(typ, loc, p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// poiCacheKey POI 检索的缓存键。
func poiCacheKey(typ POIType, loc qwx.Location, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyGeo).
		Path("poi", string(typ), loc.Query()).
		Params(p).
		String()
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
// 缓存键：`qwx:geo:poirange:<type>:<location>` 加查询参数后缀
// （含半径）。
//
// 默认 TTL 24 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointPOIRange] 覆盖。
func (c *Client) POIRange(ctx context.Context, loc qwx.Location, typ POIType, opts POIRangeOptions) (*RangeResult, error) {
	switch typ {
	case POIScenic, POICSTA, POITSTA:
	default:
		return nil, fmt.Errorf("%w: unsupported POI type %q", qwx.ErrInvalidParameter, typ)
	}
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: POI range requires coordinates", qwx.ErrInvalidLocation)
	}
	p := opts.poiRangeParams()
	q := p.Query()
	q.Set("location", loc.Query())
	q.Set("type", string(typ))

	var out RangeResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/geo/v2/poi/range",
		Query:    q,
		Endpoint: EndpointPOIRange,
		CacheKey: poiRangeCacheKey(typ, loc, p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// poiRangeCacheKey POI 范围检索的缓存键。
func poiRangeCacheKey(typ POIType, loc qwx.Location, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyGeo).
		Path("poirange", string(typ), loc.Query()).
		Params(p).
		String()
}

// TopCities 检索热门城市。
//
// 调 GET /geo/v2/city/top。
//
// opts.Range 限定搜索范围，用 ISO 3166 国家代码，如 "cn"、"world"。
//
// 缓存键：`qwx:geo:topcity` 加查询参数后缀。
//
// 默认 TTL 24 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointTopCities] 覆盖。
func (c *Client) TopCities(ctx context.Context, opts Options) (*TopCityResult, error) {
	p := opts.topParams()
	var out TopCityResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/geo/v2/city/top",
		Query:    p.Query(),
		Endpoint: EndpointTopCities,
		CacheKey: topCityCacheKey(p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// topCityCacheKey 热门城市检索的缓存键。
func topCityCacheKey(p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyGeo).
		Path("topcity").
		Params(p).
		String()
}
