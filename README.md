# qwx

[![Go Reference](https://pkg.go.dev/badge/github.com/KanoCifer/qwx.svg)](https://pkg.go.dev/github.com/KanoCifer/qwx)
[![Go Report Card](https://goreportcard.com/badge/github.com/KanoCifer/qwx)](https://goreportcard.com/report/github.com/KanoCifer/qwx)

和风天气（QWeather）开发服务的 Go 客户端。

官方只维护 iOS 与 Android SDK，服务端语言没有官方实现。本包是社区维护的
Go SDK，覆盖文档中的核心接口族，返回类型化的 Go 结构体，零第三方依赖。

## 特性

- **核心接口全覆盖**：GeoAPI 城市/POI 检索、天气预报（新一代 `/weather/v1/*`）、
  空气质量、预警、海洋潮汐、生活指数、分钟级降水预报
- **按资源族分包**：一个包 = 一个上游资源族，字段形态与缓存键不互相污染
- **两种认证**：JWT（Ed25519 签名，推荐）与 API KEY
- **零依赖**：只用标准库，`go.mod` 没有 `require`
- **缓存可插拔**：定义 `Cache` 接口，Redis / 内存 / 自定义都行
- **错误可分类**：用 `errors.Is` 判断错误类别，不用解析字符串
- **并发安全**：`Client` 无状态，可直接共享

## 安装

```bash
go get github.com/KanoCifer/qwx
```

## 包结构

| 包 | 覆盖的上游接口 |
| --- | --- |
| `qwx` | 基础客户端、认证、缓存、`Location`、错误分类、两代共用的请求结构 |
| `qwx/v1` | 新一代跨端点族共用的基础类型（`Metadata`、`Value`） |
| `qwx/v1/weather` | 新一代天气 `/weather/v1/*` |
| `qwx/v1/airquality` | 新一代空气质量 `/airquality/v1/*` |
| `qwx/v1/alert` | 预警 `/weatheralert/v1/*` |
| `qwx/v7/ocean` | 旧一代潮汐 `/v7/ocean/tide` |
| `qwx/v7/indices` | 旧一代生活指数 `/v7/indices/*` |
| `qwx/v7/minutely` | 旧一代分钟级降水 `/v7/minutely/*` |
| `qwx/geo` | GeoAPI `/geo/v2/*` |

一个包 = 一个上游资源族。`geo` 是唯一的例外：它的四个检索端点共享同一个地点
类型，拆开会让该类型失去归属。

## 快速开始

### 用 API KEY

```go
auth, err := qwx.NewAPIKeyAuth("YOUR_API_KEY")
if err != nil {
    log.Fatal(err)
}

// API Host 必传：控制台「设置」页里的专属地址，形如 abcxyz.qweatherapi.com。
base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com")
if err != nil {
    log.Fatal(err)
}

loc, err := qwx.ParseLocation("116.41,39.92") // 经度在前
if err != nil {
    log.Fatal(err)
}

// 各接口族用基础客户端构造自己的服务客户端。
w := weather.New(base)

current, err := w.Current(ctx, loc, weather.Options{
    Lang:      "zh",
    LocalTime: true, // 不传会拿到 UTC，时间差 8 小时
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(current.Condition.Text, current.Temperature.Value, current.Temperature.Unit)
```

API Host 只能从控制台「设置」页取，没有默认值：公共域名 `devapi.qweather.com`、
`api.qweather.com`、`geoapi.qweather.com` 自 2026 年起陆续停服。Host 只写域名
不带 scheme 时，SDK 自动补 `https://`。

### 用 JWT

官方推荐 JWT。先生成 Ed25519 密钥对，公钥上传到控制台：

```bash
openssl genpkey -algorithm ED25519 -out ed25519-private.pem
openssl pkey -pubout -in ed25519-private.pem > ed25519-public.pem
```

控制台创建凭据后拿到 Key ID、Project ID、Developer ID：

```go
auth, err := qwx.NewJWTAuth(qwx.JWTConfig{
    PrivateKeyPEM: string(privateKeyPEM),
    KeyID:         "YOUR_KEY_ID",
    ProjectID:     "YOUR_PROJECT_ID",
    DeveloperID:   "YOUR_DEVELOPER_ID", // 可选
})
```

JWT 有效期 15 分钟，SDK 会在过期前自动重签，调用方不用管。

> 官方已公告：2027 年 1 月 1 日起限制用 API KEY 认证的每日请求量，
> 且 SDK 5+ 不再支持 API KEY。新项目建议直接用 JWT。

### 接缓存

SDK 不内置缓存实现，注入你已有的：

```go
base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithCache(myRedisCache))
```

也可以先用内置的内存缓存开发调试：

```go
base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithCache(qwx.NewMemoryCache()))
```

缓存键由 SDK 生成（两代接口刻意不共键，避免旧缓存喂给新解析器），
TTL 按端点的数据更新节奏设定。缓存键不含 API Host，一个 Cache 实例不要跨账号共用。

要按端点调 TTL，传一份策略：

```go
base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com",
    qwx.WithCache(myRedisCache),
    qwx.WithTTLPolicy(qwx.TTLMap{
        weather.EndpointCurrent:    30 * time.Second,
        airquality.EndpointCurrent: 0, // 该端点不缓存
    }),
)
```

端点常量在各端点包里（`weather.EndpointCurrent`、`geo.EndpointCity` 等）。
单端点包只导出一个端点时，名字就是 `Endpoint`（`alert.Endpoint`、
`ocean.Endpoint`）。单次请求要临时绕过缓存，用 `qwx.WithTTLOverride(ctx, 0)`。

> 改 TTL 不改缓存键：已写入的旧条目按写入时的 TTL 到期，想立刻生效要换缓存实例或手动清键。

## 支持的接口

| 包 | 方法 | 上游接口 |
| --- | --- | --- |
| `geo` | `LookupCity` | `GET /geo/v2/city/lookup` |
| `geo` | `LookupPOI` | `GET /geo/v2/poi/lookup` |
| `geo` | `POIRange` | `GET /geo/v2/poi/range` |
| `geo` | `TopCities` | `GET /geo/v2/city/top` |
| `v1/weather` | `Current` | `GET /weather/v1/current/{lat}/{lon}` |
| `v1/weather` | `Hourly` | `GET /weather/v1/hourly/{lat}/{lon}` |
| `v1/weather` | `Daily` | `GET /weather/v1/daily/{lat}/{lon}` |
| `v1/airquality` | `Current` | `GET /airquality/v1/current/{lat}/{lon}` |
| `v1/airquality` | `Daily` | `GET /airquality/v1/daily/{lat}/{lon}` |
| `v1/airquality` | `Hourly` | `GET /airquality/v1/hourly/{lat}/{lon}` |
| `v1/airquality` | `Station` | `GET /airquality/v1/stations/{locationId}` |
| `v1/alert` | `Current` | `GET /weatheralert/v1/current/{lat}/{lon}` |
| `v7/ocean` | `Tide` | `GET /v7/ocean/tide` |
| `v7/indices` | `Indices` | `GET /v7/indices/{days}d` |
| `v7/minutely` | `Minutely` | `GET /v7/minutely/5m` |

`Tide`、`Indices`、`Minutely` 至今只有 v7 版本 —— 实测 `/weather/v1/indices/*`、
`/v1/indices/*`、`/ocean/v1/tide/*` 全部 404，官方文档也仍是 v7 路径。

## 几个容易踩的坑

**API Host 必传，不能用公共域名。** 控制台「设置」页里的专属 Host 形如
`abcxyz.qweatherapi.com`，`NewClient(auth, host)` 的第二个参数就是它。
公共域名 `devapi.qweather.com`（2026-01 停服）、`api.qweather.com`、
`geoapi.qweather.com`（2026-06 停服）会陆续不可用。
注意 GeoAPI 在专属 Host 下的路径是 `/geo/v2/*`，与旧公共域名的 `/v2/*` 不同，
SDK 已按新路径发请求。

**坐标顺序相反。** 新一代接口用路径参数 `{纬度}/{经度}`，v7 与 GeoAPI 用查询参数
`"经度,纬度"`。写反了上游回 400 `invalidParams:["longitude","latitude"]`，
不会静默返回别处。SDK 里 `Location.Path()` 给新一代用，`Location.Query()` 给旧的用。

**新一代只认坐标。** `/weather/v1/current/101010100` → 404（body 为空）。
LocationID 要先用 `geo` 包的 `LookupCity` 换成坐标。

**必须传 `localTime=true`。** 新一代默认给 UTC（`"2026-10-21T05:00Z"`），
不传本地时间会整整差 8 小时。

**湿度与云量是比例不是百分数。** 新一代的 `humidity: 0.69` 是 69%。

**潮汐日期要用站点当地当天。** 潮汐窗口下界是当地当天，用 UTC 日期会在
北京时间 00:00–08:00 要到"昨天"，上游回 400。

**分钟级降水只认坐标、只覆盖中国。** `location` 必须是 `"经度,纬度"`，
不认 LocationID 也不认地名。

**POI 检索无数据时回 400 而不是空列表。** 上游在该类型附近找不到 POI 时
返回 400 `No Such Location`：内陆城市（如北京）查 TSTA（潮汐站）与 CSTA
（观测站）都会 400，沿海城市（如舟山）才查得到。调用方要按错误处理，
不能假设"查不到就是空"。

## 错误处理

```go
if errors.Is(err, qwx.ErrQuotaExceeded) {
    // 上游 429，稍后重试
} else if errors.Is(err, qwx.ErrUpstreamRejected) {
    // 上游 4xx，请求本身不合法，重试没用
} else if errors.Is(err, qwx.ErrUnauthorized) {
    // 凭据无效或权限不足
}

var ue *qwx.UpstreamError
if errors.As(err, &ue) {
    log.Printf("上游错误：%s，非法参数：%v", ue.Title, ue.InvalidParams)
}
```

## 官方文档

- 开发文档：https://dev.qweather.com/docs/
- 身份认证：https://dev.qweather.com/docs/configuration/authentication/
- 最佳实践（不要假设）：https://dev.qweather.com/docs/best-practices/no-assumptions/

## 许可

本包遵循 MIT 许可，见 [LICENSE](LICENSE)。

本包不是和风天气官方产品。使用上游数据需遵守
[和风天气开发者许可协议](https://www.qweather.com/terms/developers-eula/)，
展示天气数据时需保留上游返回的 attribution 信息。
