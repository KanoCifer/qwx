// Package cachekey 声明影响结果的查询参数，并拼装缓存键。
//
// 键契约只有这一处定义：
//
//   - 族前缀的命名与代际隔离（见 [Family]）；
//   - 坐标的格式与顺序（见 [Key.PathCoord]）；
//   - 参数后缀的编码与顺序（见 [Params]）。
//
// 端点包只声明两件事：自己属于哪个族、有哪些影响结果的参数。同一份
// [Params] 同时生成上游查询串与缓存键后缀，参数漏进键不会发生。
//
// 键的字面量在已发布版本里定死：改动会让已写入的缓存条目全部失效。
// 本包只把分散的拼法收进一处，不改写任何已有键的字节。
package cachekey

import (
	"strconv"
	"strings"

	"github.com/KanoCifer/qwx/internal/coord"
)

// Family 是一个资源族的键前缀根。
//
// 两代与 GeoAPI 各用各的前缀，键不共：旧缓存不会被新解析器读到。
type Family string

// 各资源族的键前缀根。
//
// 前缀与包名不必对齐（潮汐族是 "qwx:tide:" 而不是 "qwx:ocean:"），
// 这是已发布键字面量的约定，不要"顺手对齐"。
const (
	// FamilyWeather 是新一代天气的键前缀根。
	FamilyWeather Family = "qwx:v1:"
	// FamilyAirQuality 是新一代空气质量的键前缀根。
	FamilyAirQuality Family = "qwx:aq:v1:"
	// FamilyAlert 是新一代预警的键前缀根。
	FamilyAlert Family = "qwx:alert:v1:"
	// FamilyGeo 是 GeoAPI 的键前缀根。
	FamilyGeo Family = "qwx:geo:"
	// FamilyTide 是旧一代潮汐的键前缀根。
	FamilyTide Family = "qwx:tide:"
	// FamilyIndices 是旧一代生活指数的键前缀根。
	FamilyIndices Family = "qwx:indices:"
	// FamilyMinutely 是旧一代分钟级降水的键前缀根。
	FamilyMinutely Family = "qwx:minutely:"
)

// Key 拼装一个缓存键。
//
// 形状固定：族前缀 + 位置段 + 参数段。位置段按追加顺序拼接，参数段
// 固定排在最后，顺序由 [Params] 决定。
type Key struct {
	family   Family
	segments []string
	params   Params
}

// New 以指定族为前缀开始拼一个键。
func New(family Family) *Key {
	return &Key{family: family}
}

// Path 追加位置段，如端点名、地点 ID、站点 ID、日期。
//
// 段之间用 ":" 连接，顺序按追加顺序。空串也是一段：不跳过，位置缺失
// 时键里要留下同样的形状。
func (k *Key) Path(segments ...string) *Key {
	k.segments = append(k.segments, segments...)
	return k
}

// PathCoord 追加坐标位置段，**纬度在前**。
//
// 新一代（/weather/v1/*、/airquality/v1/*、/weatheralert/v1/*）的路径
// 参数是纬度在前，写反了上游回 400。坐标统一两位小数，-0 归一为
// 0.00，避免 "-0.00" 这种怪值进键。
func (k *Key) PathCoord(lat, lon float64) *Key {
	return k.Path(coord.Format(lat) + "," + coord.Format(lon))
}

// Count 追加计数段，如 ":h24"。
//
// 计数总是进键：条数不同是不同数据，共键就是串味。计数段排在参数段
// 之前。
func (k *Key) Count(name string, n int) *Key {
	return k.Path(name + strconv.Itoa(n))
}

// Params 追加影响结果的查询参数后缀。
//
// 后缀的编码与顺序由 [Params] 决定，端点包不自己拼 ":lang=zh" 这类
// 字符串。
func (k *Key) Params(p Params) *Key {
	k.params = p
	return k
}

// String 拼出最终的缓存键。
func (k *Key) String() string {
	var b strings.Builder
	b.WriteString(string(k.family))
	b.WriteString(strings.Join(k.segments, ":"))
	b.WriteString(k.params.suffix())
	return b.String()
}
