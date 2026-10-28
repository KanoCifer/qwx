// Package astronomy 提供旧一代天文数据接口（/v7/astronomy/*）。
//
// 日出日落、月升月落与太阳高度角至今只有 v7 版本。
//
// 旧一代与新一代在位置形态、字段单位、时间基准、错误体与缓存键上都不同，
// 本包不与 v1 下的包混用任何逻辑。
package astronomy

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// Client 是天文数据接口的调用入口。
//
// 用基础客户端构造，无状态，可并发共享。
type Client struct {
	base qwx.Doer
}

// New 用基础客户端构造天文数据接口的调用入口。
//
// base 不能为 nil，为 nil 时 panic —— 这是编程错误，不是运行时错误。
func New(base qwx.Doer) *Client {
	if base == nil {
		panic("astronomy: nil base client")
	}
	return &Client{base: base}
}

// Sun 查询日出日落。
//
// 调 GET /v7/astronomy/sun?location=...&date=yyyyMMdd。
//
// loc 是坐标对或 LocationID —— 上游这个参数是 locationIdCoordQuery，
// 认 "经度,纬度" 与 LocationID，**不认地名**。只有地名的位置先用 geo 包
// 换成坐标或 LocationID。
//
// date 必填，格式 "yyyyMMdd"。客户端只校验格式（8 位数字且是真实日期），
// 日期范围交给上游判断：上游最多接受未来 60 天（含今天），超范围回 400。
//
// 缓存键：`qwx:astronomy:sun:<location>:<date>`。
//
// 默认 TTL 24 小时 —— 数据按日期固定，一天之内不变。可用
// [qwx.WithTTLPolicy] 按 [EndpointSun] 覆盖。
func (c *Client) Sun(ctx context.Context, loc qwx.Location, date string) (*SunResult, error) {
	if err := requireIDOrCoord(loc); err != nil {
		return nil, err
	}
	if err := validateDate(date); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("location", loc.Query())
	q.Set("date", date)

	var out SunResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/astronomy/sun",
		Query:    q,
		Endpoint: EndpointSun,
		CacheKey: sunCacheKey(loc, date),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Moon 查询月升月落与月相。
//
// 调 GET /v7/astronomy/moon?location=...&date=yyyyMMdd&lang=...。
//
// loc 是坐标对或 LocationID —— 同 [Client.Sun]，上游不认地名。
//
// date 必填，格式 "yyyyMMdd"。客户端只校验格式，日期范围交给上游判断：
// 上游最多接受未来 60 天（含今天）。
//
// opts.Lang 是可选语言代码，留空用上游默认。月相名称与图标随语言变，
// 所以 lang 进缓存键。
//
// 缓存键：`qwx:astronomy:moon:<location>:<date>` 加语言后缀。
//
// 默认 TTL 24 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointMoon] 覆盖。
func (c *Client) Moon(ctx context.Context, loc qwx.Location, date string, opts Options) (*MoonResult, error) {
	if err := requireIDOrCoord(loc); err != nil {
		return nil, err
	}
	if err := validateDate(date); err != nil {
		return nil, err
	}
	p := cachekey.Params{Lang: opts.Lang}
	q := p.Query()
	q.Set("location", loc.Query())
	q.Set("date", date)

	var out MoonResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/astronomy/moon",
		Query:    q,
		Endpoint: EndpointMoon,
		CacheKey: moonCacheKey(loc, date, p),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SolarElevationAngle 查询指定时刻的太阳高度角与方位角。
//
// 调 GET /v7/astronomy/solar-elevation-angle?location=...&date=...&time=...&tz=...&alt=...。
//
// loc **只认坐标对** —— 上游这个端点用 lonlatQuery，LocationID 会被拒绝。
//
// date 必填，格式 "yyyyMMdd"，客户端只校验格式。
//
// time 必填，格式 "HHmm" 24 时制，取值 0000~2359。
//
// tz 必填，查询地区所在时区的 UTC 偏移，形如 "0800"、"-0530"（正负号可省），
// 取值范围 [-12:00,+14:00]。
//
// alt 必填，海拔高度（米），负数非法 —— 海拔不会是负的。
//
// 参数越界返回 [qwx.ErrInvalidParameter]，位置形态不对返回
// [qwx.ErrInvalidLocation]，两种都在打上游之前返回。
//
// 缓存键：`qwx:astronomy:angle:<location>:<date>:<time>:<tz>:<alt>`。
// date、time、tz、alt 都影响结果，都要进键。
//
// 默认 TTL 24 小时，可用 [qwx.WithTTLPolicy] 按 [EndpointAngle] 覆盖。
func (c *Client) SolarElevationAngle(ctx context.Context, loc qwx.Location, date, timeStr, tz string, alt int) (*AngleResult, error) {
	if err := requireCoord(loc); err != nil {
		return nil, err
	}
	if err := validateDate(date); err != nil {
		return nil, err
	}
	if err := validateTime(timeStr); err != nil {
		return nil, err
	}
	if err := validateTimezone(tz); err != nil {
		return nil, err
	}
	if alt < 0 {
		return nil, fmt.Errorf("%w: altitude %d must not be negative", qwx.ErrInvalidParameter, alt)
	}
	q := url.Values{}
	q.Set("location", loc.Query())
	q.Set("date", date)
	q.Set("time", timeStr)
	q.Set("tz", tz)
	q.Set("alt", strconv.Itoa(alt))

	var out AngleResult
	err := c.base.Do(ctx, qwx.Request{
		Method:   "GET",
		Path:     "/v7/astronomy/solar-elevation-angle",
		Query:    q,
		Endpoint: EndpointAngle,
		CacheKey: angleCacheKey(loc, date, timeStr, tz, alt),
		TTL:      cacheTTL,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// sunCacheKey 日出日落的缓存键。
func sunCacheKey(loc qwx.Location, date string) string {
	return cachekey.New(cachekey.FamilyAstronomy).
		Path("sun", loc.Query(), date).
		String()
}

// moonCacheKey 月相的缓存键。
//
// lang 进键：月相名称与图标随语言变，是两份数据。
func moonCacheKey(loc qwx.Location, date string, p cachekey.Params) string {
	return cachekey.New(cachekey.FamilyAstronomy).
		Path("moon", loc.Query(), date).
		Params(p).
		String()
}

// angleCacheKey 太阳高度角的缓存键。
//
// date、time、tz、alt 都进键：任一不同都是另一份数据。
func angleCacheKey(loc qwx.Location, date, timeStr, tz string, alt int) string {
	return cachekey.New(cachekey.FamilyAstronomy).
		Path("angle", loc.Query(), date, timeStr, tz, strconv.Itoa(alt)).
		String()
}

// requireIDOrCoord 要求位置是坐标对或 LocationID。
//
// 上游的 locationIdCoordQuery 只认这两种形态，地名会被拒绝。
func requireIDOrCoord(loc qwx.Location) error {
	if !loc.HasCoordinates() && loc.ID == "" {
		return fmt.Errorf("%w: astronomy requires coordinates or a LocationID", qwx.ErrInvalidLocation)
	}
	return nil
}

// requireCoord 要求位置是坐标对。
//
// 上游的 lonlatQuery 只认坐标对，LocationID 会被拒绝。
func requireCoord(loc qwx.Location) error {
	if !loc.HasCoordinates() {
		return fmt.Errorf("%w: solar elevation angle requires coordinates", qwx.ErrInvalidLocation)
	}
	return nil
}

// validateDate 校验日期格式为 yyyyMMdd 且是真实日期。
//
// 日期范围（上游最多接受未来 60 天）不在这里判断 —— 那要拿"今天"比，
// 上游自己会回 400。
func validateDate(date string) error {
	if len(date) != 8 || !isDigits(date) {
		return fmt.Errorf("%w: date %q must be in yyyyMMdd format", qwx.ErrInvalidParameter, date)
	}
	if _, err := time.Parse("20060102", date); err != nil {
		return fmt.Errorf("%w: date %q is not a valid calendar date", qwx.ErrInvalidParameter, date)
	}
	return nil
}

// validateTime 校验时间格式为 HHmm，取值 0000~2359。
func validateTime(timeStr string) error {
	if len(timeStr) != 4 || !isDigits(timeStr) {
		return fmt.Errorf("%w: time %q must be in HHmm format", qwx.ErrInvalidParameter, timeStr)
	}
	if _, err := time.Parse("1504", timeStr); err != nil {
		return fmt.Errorf("%w: time %q is not a valid 24-hour time", qwx.ErrInvalidParameter, timeStr)
	}
	return nil
}

// validateTimezone 校验时区偏移格式为 HHmm，正负号可省。
//
// 取值范围 [-12:00,+14:00] —— 真实存在的时区偏移都在这个区间内。
func validateTimezone(tz string) error {
	digits := tz
	negative := false
	switch {
	case len(tz) == 5 && (tz[0] == '+' || tz[0] == '-'):
		negative = tz[0] == '-'
		digits = tz[1:]
	case len(tz) != 4:
		return fmt.Errorf("%w: tz %q must be in HHmm format, like 0800 or -0530", qwx.ErrInvalidParameter, tz)
	}
	if !isDigits(digits) {
		return fmt.Errorf("%w: tz %q must be in HHmm format, like 0800 or -0530", qwx.ErrInvalidParameter, tz)
	}
	hours, _ := strconv.Atoi(digits[:2])
	minutes, _ := strconv.Atoi(digits[2:])
	maxHours := 14
	if negative {
		maxHours = 12
	}
	if hours > maxHours || minutes > 59 {
		return fmt.Errorf("%w: tz %q is out of the UTC offset range [-12:00,+14:00]", qwx.ErrInvalidParameter, tz)
	}
	return nil
}

// isDigits 报告 s 是否全是数字。
func isDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
