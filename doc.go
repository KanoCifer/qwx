// Package qwx 是和风天气（QWeather）开发服务的 Go 客户端。
//
// 它覆盖官方文档中的核心接口族：GeoAPI 城市/POI 检索、天气预报（新一代
// /weather/v1/*）、空气质量、预警、海洋潮汐、生活指数与分钟级降水预报。
//
// # 分包
//
// 本包只放跨接口族共用的部分：基础客户端、认证、缓存、位置、错误与两代
// 共用的请求结构。一个包 = 一个上游资源族：
//
//   - [github.com/KanoCifer/qwx/v1] —— 新一代跨端点族共用的基础类型
//   - [github.com/KanoCifer/qwx/v1/weather] —— 新一代天气
//   - [github.com/KanoCifer/qwx/v1/airquality] —— 新一代空气质量
//   - [github.com/KanoCifer/qwx/v1/alert] —— 预警
//   - [github.com/KanoCifer/qwx/v7/ocean] —— 旧一代潮汐
//   - [github.com/KanoCifer/qwx/v7/indices] —— 旧一代生活指数
//   - [github.com/KanoCifer/qwx/v7/minutely] —— 旧一代分钟级降水
//   - [github.com/KanoCifer/qwx/geo] —— GeoAPI（/geo/v2/*）
//
// geo 是唯一的例外：它的四个检索端点共享同一个地点类型，拆开会让该类型
// 失去归属。
//
// 旧一代与 GeoAPI 的结果带 refer 对象（数据来源与许可），用 [Refer] 表达；
// 新一代的来源信息在 [github.com/KanoCifer/qwx/v1.Metadata] 的 Attributions 里。
//
// 两代的字段形态、单位、时间基准、错误体与缓存键都不同，拆包是为了让它们
// 不在同一处混用。新一代的基础类型（[github.com/KanoCifer/qwx/v1.Metadata]、
// [github.com/KanoCifer/qwx/v1.Value]）放在新一代的包里，不在本包 ——
// 旧一代与 GeoAPI 用不到它们。
//
// # 用法
//
// 先构造基础客户端，再用它构造各接口族的服务客户端。基础客户端需要认证方式
// 和控制台分配的专属 API Host，两者都必填：
//
//	auth, err := qwx.NewAPIKeyAuth("YOUR_API_KEY")
//	if err != nil {
//		return err
//	}
//	base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com")
//	if err != nil {
//		return err
//	}
//	w := weather.New(base)
//	cur, err := w.Current(ctx, loc, weather.Options{Lang: "zh", LocalTime: true})
//
// API Host 形如 abcxyz.qweatherapi.com，在控制台「设置」页查看。公共域名
// （devapi.qweather.com、api.qweather.com、geoapi.qweather.com）自 2026 年起
// 陆续停服，SDK 不提供默认值。只给 host 时自动补 "https://"。
//
// # 认证
//
// 官方支持两种认证方式，本包都支持，但**不要同时使用**：
//
//   - JWT（Ed25519 签名，推荐）：见 [NewJWTAuth]。需要控制台生成的
//     Ed25519 私钥，以及凭据的 Key ID、Project ID 与 Developer ID。
//     三项都必填，缺任一项返回 [ErrInvalidConfig]。
//   - API KEY：见 [NewAPIKeyAuth]。官方已公告从 2027 年 1 月 1 日起
//     限制用 API KEY 认证的每日请求量，新项目建议直接用 JWT。
//
// # 缓存
//
// 本包不内置缓存实现，而是定义 [Cache] 接口由调用方注入（Redis、内存均可）。
// 未注入时每次请求都打到上游。
//
// Redis 实现是独立模块 [github.com/KanoCifer/qwx/rediscache]，依赖 go-redis，
// 核心包不引入它。需要时单独引入：
//
//	go get github.com/KanoCifer/qwx/rediscache
//
// 缓存时长默认由端点包决定，可用 [WithTTLPolicy] 按端点覆盖：
//
//	base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com",
//	    qwx.WithCache(cache),
//	    qwx.WithTTLPolicy(qwx.TTLMap{
//	        weather.EndpointCurrent: 30 * time.Second,
//	    }),
//	)
//
// 单次请求要临时绕过缓存，用 [WithTTLOverride]。
//
// # 日志
//
// SDK 用标准库 [log/slog]，不注入 logger 时不产生任何输出：
//
//	base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com",
//	    qwx.WithLogger(slog.Default()),
//	)
//
// 级别由 handler 决定。日志里有端点、路径、状态码、耗时与缓存键，
// 没有认证凭据，也没有响应体。
//
// # 并发
//
// [Client] 无状态，可并发共享。所有方法都接受 [context.Context]。
package qwx
