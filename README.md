# qwx

[![Go Reference](https://pkg.go.dev/badge/github.com/KanoCifer/qwx.svg)](https://pkg.go.dev/github.com/KanoCifer/qwx)
[![Go Report Card](https://goreportcard.com/badge/github.com/KanoCifer/qwx)](https://goreportcard.com/report/github.com/KanoCifer/qwx)

和风天气（QWeather）开发服务的 Go 客户端。

官方只维护 iOS 与 Android SDK，服务端语言没有官方实现。本包是社区维护的
Go SDK，覆盖文档中的核心接口族，返回类型化的 Go 结构体，零第三方依赖。

## 特性

- **核心接口全覆盖**：GeoAPI 城市/POI 检索、天气预报（新一代 `/weather/v1/*`）、
  空气质量、预警、海洋潮汐、生活指数、分钟级降水预报
- **两种认证**：JWT（Ed25519 签名，推荐）与 API KEY
- **零依赖**：只用标准库，`go.mod` 没有 `require`
- **缓存可插拔**：定义 `Cache` 接口，Redis / 内存 / 自定义都行
- **错误可分类**：用 `errors.Is` 判断错误类别，不用解析字符串
- **并发安全**：`Client` 无状态，可直接共享

## 安装

```bash
go get github.com/KanoCifer/qwx
```

## 快速开始

### 用 API KEY

```go
auth, err := qwx.NewAPIKeyAuth("YOUR_API_KEY")
if err != nil {
    log.Fatal(err)
}

client, err := qwx.NewClient(auth)
if err != nil {
    log.Fatal(err)
}

loc, err := qwx.ParseLocation("116.41,39.92") // 经度在前
if err != nil {
    log.Fatal(err)
}

current, err := client.Current(ctx, loc, qwx.WeatherOptions{
    Lang:      "zh",
    LocalTime: true, // 不传会拿到 UTC，时间差 8 小时
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(current.Condition.Text, current.Temperature.Value, current.Temperature.Unit)
```

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
client, err := qwx.NewClient(auth, qwx.WithCache(myRedisCache))
```

也可以先用内置的内存缓存开发调试：

```go
client, err := qwx.NewClient(auth, qwx.WithCache(qwx.NewMemoryCache()))
```

缓存键由 SDK 生成（两代接口刻意不共键，避免旧缓存喂给新解析器），
TTL 按端点的数据更新节奏设定。

## 支持的接口

| 方法 | 上游接口 |
| --- | --- |
| `LookupCity` | `GET /geo/v2/city/lookup` |
| `LookupPOI` | `GET /geo/v2/poi/lookup` |
| `POIRange` | `GET /geo/v2/poi/range` |
| `TopCities` | `GET /geo/v2/city/top` |
| `Current` | `GET /weather/v1/current/{lat}/{lon}` |
| `Hourly` | `GET /weather/v1/hourly/{lat}/{lon}` |
| `Daily` | `GET /weather/v1/daily/{lat}/{lon}` |
| `AirQuality` | `GET /airquality/v1/current/{lat}/{lon}` |
| `AirQualityDailyForecast` | `GET /airquality/v1/daily/{lat}/{lon}` |
| `AirQualityHourlyForecast` | `GET /airquality/v1/hourly/{lat}/{lon}` |
| `AirStation` | `GET /airquality/v1/stations/{locationId}` |
| `Alerts` | `GET /weatheralert/v1/current/{lat}/{lon}` |
| `Warnings` | `GET /v7/warning/now` |
| `Tide` | `GET /v7/ocean/tide` |
| `Indices` | `GET /v7/indices/{days}d` |
| `Minutely` | `GET /v7/minutely/5m` |

`Tide`、`Indices`、`Minutely` 至今只有 v7 版本 —— 实测 `/weather/v1/indices/*`、
`/v1/indices/*`、`/ocean/v1/tide/*` 全部 404，官方文档也仍是 v7 路径。

## 几个容易踩的坑

**坐标顺序相反。** 新一代接口用路径参数 `{纬度}/{经度}`，v7 与 GeoAPI 用查询参数
`"经度,纬度"`。写反了上游回 400 `invalidParams:["longitude","latitude"]`，
不会静默返回别处。SDK 里 `Location.Path()` 给新一代用，`Location.Query()` 给旧的用。

**新一代只认坐标。** `/weather/v1/current/101010100` → 404（body 为空）。
LocationID 要先用 `LookupCity` 换成坐标。

**必须传 `localTime=true`。** 新一代默认给 UTC（`"2026-10-21T05:00Z"`），
不传本地时间会整整差 8 小时。

**湿度与云量是比例不是百分数。** 新一代的 `humidity: 0.69` 是 69%，
v7 的 `"26"` 才是百分数。

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
