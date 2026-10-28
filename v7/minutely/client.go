// Package minutely 提供旧一代分钟级降水预报接口（/v7/minutely/*）。
//
// 分钟级降水至今只有 v7 版本，没有新一代。
//
// 旧一代与新一代在位置形态、字段单位、时间基准、错误体与缓存键上都不同，
// 本包不与 v1 下的包混用任何逻辑。
package minutely

import (
	"context"
	"fmt"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Client 是分钟级降水预报接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造分钟级降水预报接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("minutely: nil base client")
	}
	return &Client{base: base}
}

// Minutely 查询分钟级降水预报。
//
// 调 GET /v7/minutely/5m?location=...。
//
// 上游至今只有 v7 版本，没有新一代。
//
// 三条硬约束（官方文档 + 实测）：
//
//  1. **只认坐标**：location 必须是 "经度,纬度"，不认 LocationID 也不认地名。
//  2. **只覆盖中国区域**：境外坐标上游没有数据。
//  3. **粒度是 5 分钟**：未来 2 小时共 24 条，上游每 5 分钟更新一次。
//
// 缓存键：`qwx:minutely:<lon>,<lat>` 加语言后缀。TTL 与上游 5 分钟的更新节奏对齐。
//
// 默认 TTL 5 分钟，可用 [qwx.WithTTLPolicy] 按 [Endpoint] 覆盖。
func (c *Client) Minutely(ctx context.Context, loc qwx.Location, lang string) (*Result, error) {
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: minutely forecast requires coordinates", qwx.ErrInvalidLocation)
	}
	p := cachekey.Params{Lang: lang}
	q := p.Query()
	q.Set("location", loc.Query())

	var out Result
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/minutely/5m",
		Query:    q,
		Endpoint: Endpoint,
		CacheKey: minutelyCacheKey(loc, p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// minutelyCacheKey 分钟级降水的缓存键。
func minutelyCacheKey(loc qwx.Location, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyMinutely).
		Path(loc.Query()).
		Params(p).
		String()
}
