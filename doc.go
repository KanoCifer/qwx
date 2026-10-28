// Package qwx 是和风天气（QWeather）开发服务的 Go 客户端。
//
// 它覆盖官方文档中的核心接口族：GeoAPI 城市/POI 检索、天气预报（新一代
// /weather/v1/*）、空气质量、预警、海洋潮汐、生活指数与分钟级降水预报。
//
// # 分包
//
// 本包只放跨接口族共用的部分：基础客户端、认证、缓存、位置、错误与新一代
// 基础类型。接口族按上游的代际拆成子包：
//
//   - [github.com/KanoCifer/qwx/v1/weather] —— 新一代天气
//   - [github.com/KanoCifer/qwx/v1/airquality] —— 新一代空气质量
//   - [github.com/KanoCifer/qwx/v1/alert] —— 新一代预警
//   - [github.com/KanoCifer/qwx/v7] —— 旧一代（/v7/*）
//   - [github.com/KanoCifer/qwx/geo] —— GeoAPI（/geo/v2/*）
//
// 两代的字段形态、单位、时间基准、错误体与缓存键都不同，拆包是为了让它们
// 不在同一处混用。
//
// # 用法
//
// 先构造基础客户端，再用它构造各接口族的服务客户端：
//
//	base, err := qwx.NewClient(auth)
//	if err != nil {
//		return err
//	}
//	w := weather.New(base)
//	cur, err := w.Current(ctx, loc, weather.Options{Lang: "zh", LocalTime: true})
//
// # 认证
//
// 官方支持两种认证方式，本包都支持，但**不要同时使用**：
//
//   - JWT（Ed25519 签名，推荐）：见 [NewJWTAuth]。需要控制台生成的
//     Ed25519 私钥，以及凭据的 Key ID、Project ID、Developer ID。
//   - API KEY：见 [NewAPIKeyAuth]。官方已公告从 2027 年 1 月 1 日起
//     限制用 API KEY 认证的每日请求量，新项目建议直接用 JWT。
//
// # 缓存
//
// 本包不内置缓存实现，而是定义 [Cache] 接口由调用方注入（Redis、内存均可）。
// 未注入时每次请求都打到上游。
//
// # 并发
//
// [Client] 无状态，可并发共享。所有方法都接受 [context.Context]。
package qwx
