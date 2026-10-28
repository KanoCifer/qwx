// Package indices 提供旧一代生活指数接口（/v7/indices/*）。
//
// 生活指数至今只有 v7 版本 —— 实测 /weather/v1/indices/* 与 /v1/indices/* 全部 404。
//
// 旧一代与新一代在位置形态、字段单位、时间基准、错误体与缓存键上都不同，
// 本包不与 v1 下的包混用任何逻辑。
package indices

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Client 是生活指数接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造生活指数接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("indices: nil base client")
	}
	return &Client{base: base}
}

// Indices 查询生活指数。
//
// 调 GET /v7/indices/{days}d?type={type}&location=...。
//
// 上游至今只有 v7 版本，没有新一代（实测 /weather/v1/indices/* 与
// /v1/indices/* 均 404）。
//
// days 只支持 1 与 3 —— 实测 2d / 7d / 10d / 15d / 30d 全部 404。
// type 取 0~16，0 表示全部。
//
// loc 必须是坐标对：上游对裸经度（如 "121.47"）一律回 400。
//
// lang 是语言，如 [qwx.LangZH]。留空用上游默认。
//
// 缓存键：`qwx:indices:<location>:t<type>:d<days>` 加语言后缀。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [Endpoint] 覆盖。
func (c *Client) Indices(ctx context.Context, loc qwx.Location, typ, days int, lang qwx.Lang) (*Result, error) {
	return c.IndicesMulti(ctx, loc, []int{typ}, days, lang)
}

// IndicesMulti 查询多种生活指数。
//
// 调 GET /v7/indices/{days}d?type={type1,type2}&location=...。
//
// 上游的 type 接受逗号分隔的多个类型（"3,5"），一次请求拿多种指数。
// 取全部（[TypeAll]）时不能再带其他类型，否则返回 [qwx.ErrInvalidParameter]。
//
// days 只支持 1 与 3。types 每项取值 1~16，不能为空。
//
// lang 是语言，如 [qwx.LangZH]。留空用上游默认。
//
// 缓存键：`qwx:indices:<location>:t<types>:d<days>` 加语言后缀，
// 多个类型用逗号连接（"t3,5"）。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [Endpoint] 覆盖。
func (c *Client) IndicesMulti(ctx context.Context, loc qwx.Location, types []int, days int, lang qwx.Lang) (*Result, error) {
	if len(types) == 0 {
		return nil, fmt.Errorf("%w: indices types must not be empty", qwx.ErrInvalidParameter)
	}
	for _, typ := range types {
		if typ < TypeAll || typ > MaxType {
			return nil, fmt.Errorf("%w: indices type %d out of range [%d,%d]", qwx.ErrInvalidParameter, typ, TypeAll, MaxType)
		}
	}
	if len(types) > 1 {
		for _, typ := range types {
			if typ == TypeAll {
				return nil, fmt.Errorf("%w: indices type %d cannot be combined with other types", qwx.ErrInvalidParameter, TypeAll)
			}
		}
	}
	if days != 1 && days != 3 {
		return nil, fmt.Errorf("%w: indices days %d is not supported (only 1 and 3)", qwx.ErrInvalidParameter, days)
	}
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: indices requires coordinates", qwx.ErrInvalidLocation)
	}
	if err := lang.Validate(); err != nil {
		return nil, err
	}
	typeSeg := indicesTypeSegment(types)
	p := cachekey.Params{Lang: lang.API}
	q := p.Query()
	q.Set("location", loc.Query())
	q.Set("type", typeSeg)

	var out Result
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/indices/" + strconv.Itoa(days) + "d",
		Query:    q,
		Endpoint: Endpoint,
		CacheKey: indicesCacheKey(loc, typeSeg, days, p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// indicesTypeSegment 拼出 type 参数与缓存键里的类型段。
//
// 多个类型用逗号连接：上游的 type 参数与键里的类型段共用这个形状。
// 单类型时是 "0" 到 "16"，与旧的单类型调用完全一致。
func indicesTypeSegment(types []int) string {
	if len(types) == 1 {
		return strconv.Itoa(types[0])
	}
	parts := make([]string, len(types))
	for i, typ := range types {
		parts[i] = strconv.Itoa(typ)
	}
	return strings.Join(parts, ",")
}

// indicesCacheKey 生活指数的缓存键。
//
// 指数类型与天数都进键：type=0 与 type=1、days=1 与 days=3 是不同的数据。
func indicesCacheKey(loc qwx.Location, typeSeg string, days int, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyIndices).
		Path(loc.Query()).
		Path("t"+typeSeg).
		Count("d", days).
		Params(p).
		String()
}
