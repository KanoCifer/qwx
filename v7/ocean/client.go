// Package ocean 提供旧一代海洋接口（/v7/ocean/*）。
//
// 潮汐至今只有 v7 版本 —— 实测 /ocean/v1/tide/* 全部 404，官方文档也仍是 v7 路径。
//
// 旧一代与新一代在位置形态、字段单位、时间基准、错误体与缓存键上都不同，
// 本包不与 v1 下的包混用任何逻辑。
package ocean

import (
	"context"
	"fmt"
	"net/url"

	"github.com/KanoCifer/qwx"
)

// Client 是潮汐接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造潮汐接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("ocean: nil base client")
	}
	return &Client{base: base}
}

// Tide 查询潮汐。
//
// 调 GET /v7/ocean/tide?location=...&date=...。
//
// location 是潮汐站 ID（如 "P2352"），不是坐标、不是 LocationID。
// 用 geo 包的 LookupPOI 配 POITSTA 可以检索到本站点。
//
// date 是潮汐站当地日期，格式 "YYYYMMDD"。
//
// **日期必须是潮汐站当地的当天**：潮汐窗口的下界是当地当天，
// 用 UTC 日期会在北京时间 00:00–08:00 要到"昨天"，上游会回 400。
//
// 缓存键：`qwx:tide:<location>:<date>`。
//
// 键前缀是 `qwx:tide:` 而不是 `qwx:ocean:` —— 键字面量在已发布版本里定死，
// 改了会让已写入的缓存条目全部失效，不要"顺手对齐"。
//
// 默认 TTL 1 小时，可用 [qwx.WithTTLPolicy] 按 [Endpoint] 覆盖。
func (c *Client) Tide(ctx context.Context, stationID, date string) (*Result, error) {
	if stationID == "" {
		return nil, fmt.Errorf("%w: empty tide station ID", qwx.ErrInvalidParameter)
	}
	if err := validateDate(date); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("location", stationID)
	q.Set("date", date)

	var out Result
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/ocean/tide",
		Query:    q,
		Endpoint: Endpoint,
		CacheKey: "qwx:tide:" + stationID + ":" + date,
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// validateDate 校验日期格式为 YYYYMMDD。
func validateDate(date string) error {
	if len(date) != 8 {
		return fmt.Errorf("%w: date %q must be in YYYYMMDD format", qwx.ErrInvalidParameter, date)
	}
	for i := 0; i < len(date); i++ {
		if date[i] < '0' || date[i] > '9' {
			return fmt.Errorf("%w: date %q must be in YYYYMMDD format", qwx.ErrInvalidParameter, date)
		}
	}
	return nil
}
