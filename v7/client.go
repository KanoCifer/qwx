// Package v7 提供旧一代接口（/v7/*）。
//
// 旧一代与新一代在位置形态、字段单位、时间基准、错误体与缓存键上都不同，
// 本包不与 v1 下的包混用任何逻辑。
//
// 潮汐、生活指数、分钟级降水至今只有 v7 版本 —— 实测 /weather/v1/indices/*、
// /v1/indices/*、/ocean/v1/tide/* 全部 404，官方文档也仍是 v7 路径。
package v7

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// 缓存时长：预警与分钟级降水对齐上游 5 分钟的更新节奏，潮汐与生活指数 1 小时。
const (
	alertCacheTTL    = 5 * time.Minute
	minutelyCacheTTL = 5 * time.Minute
	forecastCacheTTL = time.Hour
)

// Client 是旧一代接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造旧一代接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("v7: nil base client")
	}
	return &Client{base: base}
}

// Warnings 查询 location 上的旧一代预警。
//
// 调 GET /v7/warning/now?location=...。
//
// 与新一代预警（qwx/v1/alert）的区别：v7 版本认 LocationID 与地名，
// 新一代只认坐标。两者数据结构完全不同，按需选择。
//
// 缓存键：`qwx:warning:v7:<location>` 加语言后缀。
func (c *Client) Warnings(ctx context.Context, loc qwx.Location, lang string) (*WarningsResult, error) {
	q := url.Values{}
	if lang != "" {
		q.Set("lang", lang)
	}
	q.Set("location", loc.Query())

	var out WarningsResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/warning/now",
		Query:    q,
		CacheKey: "qwx:warning:v7:" + loc.Query() + cachekey.Lang(lang),
		TTL:      alertCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
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
func (c *Client) Tide(ctx context.Context, stationID, date string) (*TideResult, error) {
	if stationID == "" {
		return nil, fmt.Errorf("%w: empty tide station ID", qwx.ErrInvalidParameter)
	}
	if err := validateDate(date); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("location", stationID)
	q.Set("date", date)

	var out TideResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/ocean/tide",
		Query:    q,
		CacheKey: "qwx:tide:" + stationID + ":" + date,
		TTL:      forecastCacheTTL,
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
// 缓存键：`qwx:indices:<location>:t<type>:d<days>` 加语言后缀。
func (c *Client) Indices(ctx context.Context, loc qwx.Location, typ, days int, lang string) (*IndicesResult, error) {
	if typ < IndicesTypeAll || typ > IndicesMaxType {
		return nil, fmt.Errorf("%w: indices type %d out of range [0,%d]", qwx.ErrInvalidParameter, typ, IndicesMaxType)
	}
	if days != 1 && days != 3 {
		return nil, fmt.Errorf("%w: indices days %d is not supported (only 1 and 3)", qwx.ErrInvalidParameter, days)
	}
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: indices requires coordinates", qwx.ErrInvalidLocation)
	}
	q := url.Values{}
	q.Set("location", loc.Query())
	q.Set("type", strconv.Itoa(typ))
	if lang != "" {
		q.Set("lang", lang)
	}

	var out IndicesResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/indices/" + strconv.Itoa(days) + "d",
		Query:    q,
		CacheKey: "qwx:indices:" + loc.Query() + ":t" + strconv.Itoa(typ) + ":d" + strconv.Itoa(days) + cachekey.Lang(lang),
		TTL:      forecastCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
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
func (c *Client) Minutely(ctx context.Context, loc qwx.Location, lang string) (*MinutelyResult, error) {
	if !loc.HasCoordinates() {
		return nil, fmt.Errorf("%w: minutely forecast requires coordinates", qwx.ErrInvalidLocation)
	}
	q := url.Values{}
	q.Set("location", loc.Query())
	if lang != "" {
		q.Set("lang", lang)
	}

	var out MinutelyResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/minutely/5m",
		Query:    q,
		CacheKey: "qwx:minutely:" + loc.Query() + cachekey.Lang(lang),
		TTL:      minutelyCacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
