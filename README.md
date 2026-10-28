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
- **两种认证**：JWT（Ed25519 签名，官方推荐）与 API KEY
- **零依赖**：核心包只用标准库，`go.mod` 没有 `require`
- **缓存可插拔**：定义 `Cache` 接口，Redis / 内存 / 自定义都行
- **错误可分类**：用 `errors.Is` 判断错误类别，不用解析字符串
- **并发安全**：`Client` 无状态，可直接共享

## 安装

```bash
go get github.com/KanoCifer/qwx
```

要求 Go 1.24 及以上。

需要 Redis 缓存时，额外引入独立的缓存模块（它依赖 go-redis，核心包不受影响）：

```bash
go get github.com/KanoCifer/qwx/rediscache
```

## 快速开始

三步：造认证、造基础客户端、造接口族的客户端。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/weather"
)

func main() {
	// 1. 认证。JWT 用控制台分配的凭据，API KEY 用 qwx.NewAPIKeyAuth。
	auth, err := qwx.NewJWTAuth(qwx.JWTConfig{
		PrivateKeyPEM: os.Getenv("QWEATHER_JWT_PRIVATE_KEY"),
		KeyID:         os.Getenv("QWEATHER_KEY_ID"),
		ProjectID:     os.Getenv("QWEATHER_PROJECT_ID"),
		DeveloperID:   os.Getenv("QWEATHER_DEVELOPER_ID"), // 必填，控制台「设置」页
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. 基础客户端。apiHost 是控制台「设置」页分配的专属 Host，必填。
	base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com")
	if err != nil {
		log.Fatal(err)
	}

	// 3. 各接口族用基础客户端构造自己的服务客户端。
	w := weather.New(base)

	ctx := context.Background()
	loc, err := qwx.ParseLocation("116.41,39.92") // 经度在前
	if err != nil {
		log.Fatal(err)
	}

	current, err := w.Current(ctx, loc, weather.Options{
		Lang:      "zh",
		LocalTime: true, // 不传会拿到 UTC，时间差 8 小时
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(current.Condition.Text, current.Temperature.Value, current.Temperature.Unit)
}
```

仓库里的 `examples/basic` 是一个更长的例子，覆盖天气、逐时逐日、城市检索与
生活指数，走 API KEY 认证，从环境变量读配置：

```bash
export QWEATHER_API_KEY="你的 API KEY"
export QWEATHER_API_HOST="abcxyz.qweatherapi.com"
go run ./examples/basic
```

API Host 只能从控制台「设置」页取，没有默认值：公共域名 `devapi.qweather.com`
（2026-01 停服）、`api.qweather.com`、`geoapi.qweather.com`（2026-06 停服）
陆续失效。Host 只写域名不带 scheme 时，SDK 自动补 `https://`。

## 使用指南

### 认证

两种认证不能同时使用，同时带上会让上游拒绝请求。默认选 JWT。

JWT 需要 Ed25519 密钥对，公钥上传到控制台：

```bash
openssl genpkey -algorithm ED25519 -out ed25519-private.pem
openssl pkey -pubout -in ed25519-private.pem > ed25519-public.pem
```

控制台创建凭据后拿到 Key ID、Project ID、Developer ID，三项都必填：

```go
import "github.com/KanoCifer/qwx"

auth, err := qwx.NewJWTAuth(qwx.JWTConfig{
	PrivateKeyPEM: string(privateKeyPEM),
	KeyID:         "YOUR_KEY_ID",
	ProjectID:     "YOUR_PROJECT_ID",
	DeveloperID:   "YOUR_DEVELOPER_ID",
})
```

三项各对应 JWT 的一个位置：`kid` 是凭据 ID，`sub` 是项目 ID，`iss` 是 Developer ID
（控制台「设置」页，形如 Q 开头的 10 位字母数字）。`iss` 官方口径是必填的签发者，
缺失时上游按它校验凭据归属会失败，所以 SDK 直接在构造时返回 `ErrInvalidConfig`，
不等到上游 401 才发现漏配。

JWT 有效期 15 分钟，SDK 在过期前 1 分钟自动重签，调用方不用管。签名有锁，
并发安全。

API KEY （不推荐使用）适合存量项目：

```go
import "github.com/KanoCifer/qwx"

auth, err := qwx.NewAPIKeyAuth("YOUR_API_KEY")
```

> 官方已公告：2027 年 1 月 1 日起限制用 API KEY 认证的每日请求量，
> 且 SDK 5+ 不再支持 API KEY。新项目建议直接用 JWT。

### 位置

`qwx.Location` 收敛了上游的三种位置形态：

```go
import "github.com/KanoCifer/qwx"

// 坐标对，字符串里经度在前
loc, err := qwx.ParseLocation("116.41,39.92")
if err != nil {
	return err
}

// 直接构造
loc = qwx.Location{Lat: 39.92, Lon: 116.41} // 字段是纬度在前
loc = qwx.Location{ID: "101010100"}          // LocationID
loc = qwx.Location{Name: "杭州"}              // 地名，只有 GeoAPI 城市/POI 检索认
```

各端点认哪种形态不一样：

| 形态 | 谁认 |
| --- | --- |
| 坐标对 | 全部端点 |
| LocationID | GeoAPI 全部端点；行政区划是 9~10 位数字，POI 是数字加字母后缀，潮汐站是 `P` 加数字 |
| 地名 | 只有 `geo.LookupCity` 与 `geo.LookupPOI` |

新一代只认坐标。手里只有 LocationID 时，先用 `geo.LookupCity` 换成坐标。

### 天气

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/weather"
)

w := weather.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

current, err := w.Current(ctx, loc, weather.Options{Lang: "zh", LocalTime: true})
if err != nil {
	return err
}
fmt.Println(current.Condition.Text, current.Temperature.Value, current.Temperature.Unit)
fmt.Println(current.Humidity * 100, "%") // 湿度是 0~1 的比例

hourly, err := w.Hourly(ctx, loc, 24, weather.Options{Lang: "zh", LocalTime: true})
if err != nil {
	return err
}
for _, h := range hourly.Hours {
	var prob float64
	if h.Precipitation.Probability != nil {
		prob = *h.Precipitation.Probability
	}
	fmt.Println(h.ForecastTime, h.Condition.Text, prob)
}

daily, err := w.Daily(ctx, loc, 3, weather.Options{Lang: "zh", LocalTime: true})
if err != nil {
	return err
}
for _, d := range daily.Days {
	fmt.Println(d.ForecastStartTime, d.TemperatureMax.Value, d.TemperatureMin.Value)
}
```

`Hourly` 的 hours 取 1~240，`Daily` 的 days 取 1~10，越界返回 `ErrInvalidParameter`，
不打上游。列表长度不假设，上游可能给得更少。

### 空气质量

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/airquality"
)

aq := airquality.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

cur, err := aq.Current(ctx, loc, airquality.Options{Lang: "zh"})
if err != nil {
	return err
}
for _, idx := range cur.Indexes {
	fmt.Println(idx.Code, idx.AQI, idx.Category)
}

// 附近监测站，再按站点查实测数据
for _, s := range cur.Stations {
	station, err := aq.Station(ctx, s.ID, airquality.Options{Lang: "zh"})
	if err != nil {
		return err
	}
	fmt.Println(s.Name, station.Pollutants)
}
```

不同标准的 AQI 量纲不同（QAQI 是 0~1，US EPA 是 0~500），不要跨标准比较。

### 预警

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/alert"
)

al := alert.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

result, err := al.Current(ctx, loc, alert.Options{Lang: "zh", LocalTime: true})
if err != nil {
	return err
}
for _, a := range result.Alerts {
	fmt.Println(a.SenderName, a.Headline, a.Severity, a.EffectiveTime)
}
```

预警只认坐标。没有预警时上游返回空列表，`Metadata.ZeroResult` 为 true。

### GeoAPI 检索

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/geo"
)

g := geo.New(base)
ctx := context.Background()

// 按地名检索城市
cities, err := g.LookupCity(ctx, qwx.Location{Name: "杭州"}, geo.Options{Lang: "zh", Number: 10})
if err != nil {
	return err
}
for _, c := range cities.Locations {
	lat, lon, err := c.Coordinates()
	if err != nil {
		return err
	}
	fmt.Println(c.ID, c.DisplayName(), lat, lon)
}

// 附近潮汐站
pois, err := g.LookupPOI(ctx, qwx.Location{Lat: 30.0, Lon: 122.0}, geo.POITSTA, geo.Options{Lang: "zh"})
if err != nil {
	return err
}
for _, p := range pois.POIs {
	fmt.Println(p.ID, p.Name)
}

// 热门城市
top, err := g.TopCities(ctx, geo.Options{Lang: "zh"})
if err != nil {
	return err
}
for _, c := range top.Cities {
	fmt.Println(c.DisplayName())
}
```

`POIRange` 检索指定坐标附近的 POI，只认坐标，`Radius` 是公里数：

```go
ranges, err := g.POIRange(ctx, qwx.Location{Lat: 30.0, Lon: 122.0}, geo.POIScenic,
	geo.POIRangeOptions{Lang: "zh", Number: 10, Radius: 20})
```

GeoAPI 的每个结果都带一个 `Refer`，是数据来源与许可声明。展示 GeoAPI 数据时
要一并展示：

```go
for _, s := range cities.Refer.Sources {
	fmt.Println(s)
}
```

### 潮汐

潮汐站 ID 用 `geo.LookupPOI` 配 `geo.POITSTA` 检索，日期用站点当地当天：

```go
import (
	"context"
	"time"

	"github.com/KanoCifer/qwx/v7/ocean"
)

o := ocean.New(base)
ctx := context.Background()

// date 是潮汐站当地日期，格式 YYYYMMDD
date := time.Now().Format("20060102")
result, err := o.Tide(ctx, "P2419", date)
if err != nil {
	return err
}
for _, t := range result.TideTable {
	fmt.Println(t.FXTime, t.Height, t.Type) // Type: H 高潮 / L 低潮
}
```

潮位高度保持字符串，保留上游给的小数位数。

### 生活指数

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v7/indices"
)

idx := indices.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

// typ 取 0~16，0 表示全部；days 只支持 1 与 3
result, err := idx.Indices(ctx, loc, indices.TypeFishing, 1, "zh")
if err != nil {
	return err
}
for _, d := range result.Daily {
	fmt.Println(d.Date, d.Name, d.Category, d.Text)
}
```

### 分钟级降水

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v7/minutely"
)

mi := minutely.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

result, err := mi.Minutely(ctx, loc, "zh")
if err != nil {
	return err
}
fmt.Println(result.Summary)
for _, m := range result.Minutely {
	fmt.Println(m.FXTime, m.Precip, m.Type)
}
```

只覆盖中国区域，境外坐标上游没有数据。`Precip` 是字符串：空串表示「无此数据」，
`"0.00"` 表示「真的没下雨」。

### 缓存

SDK 不内置缓存实现，注入你已有的：

```go
import "github.com/KanoCifer/qwx"

base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithCache(myRedisCache))
```

也可以先用内置的内存缓存开发调试：

```go
import "github.com/KanoCifer/qwx"

base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithCache(qwx.NewMemoryCache()))
```

用 Redis 就引入 `rediscache` 模块，它实现了 `qwx.Cache`：

```go
import (
	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/rediscache"
)

cache, err := rediscache.New(rediscache.Options{
	Addr: "127.0.0.1:6379",
	// Password: "...",  需要认证时填
	// KeyPrefix: "myapp:",  多个应用共用一个 Redis 实例时隔离键
})
if err != nil {
	return err
}
defer cache.Close()

base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithCache(cache))
```

`rediscache` 支持 TLS（`TLSConfig`）、ACL 用户名（`Username`）、指定库（`DB`）与
连接超时。构造时不会连 Redis，用 `cache.Ping(ctx)` 做启动自检。

缓存键由 SDK 生成（两代接口刻意不共键，避免旧缓存喂给新解析器），
TTL 按端点的数据更新节奏设定。缓存键不含 API Host，一个 Cache 实例不要跨账号共用。

要按端点调 TTL，传一份策略：

```go
import (
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/airquality"
	"github.com/KanoCifer/qwx/v1/weather"
)

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
`ocean.Endpoint`）。默认时长：

| 端点 | 默认 TTL |
| --- | --- |
| 天气实况 | 10 分钟 |
| 逐时 / 逐日预报 | 1 小时 |
| 空气质量实况 / 监测站 | 30 分钟 |
| 空气质量预报 | 1 小时 |
| 预警 / 分钟级降水 | 5 分钟 |
| 潮汐 / 生活指数 | 1 小时 |
| GeoAPI | 24 小时 |

需要按时间或负载动态调整时用 `qwx.TTLPolicyFunc`。单次请求要临时绕过缓存，
用 `qwx.WithTTLOverride(ctx, 0)`，优先级最高。

> 改 TTL 不改缓存键：已写入的旧条目按写入时的 TTL 到期，想立刻生效要换缓存实例或手动清键。

### 错误处理

`errors.Is` 判断类别，`errors.As` 取上游细节：

```go
import (
	"errors"

	"github.com/KanoCifer/qwx"
)

if errors.Is(err, qwx.ErrQuotaExceeded) {
	// 上游 429，稍后重试
} else if errors.Is(err, qwx.ErrUpstreamRejected) {
	// 上游 4xx，请求本身不合法，重试没用
} else if errors.Is(err, qwx.ErrUnauthorized) {
	// 凭据无效或权限不足
} else if errors.Is(err, qwx.ErrInvalidLocation) {
	// 位置形态这个端点不认
}

var ue *qwx.UpstreamError
if errors.As(err, &ue) {
	log.Printf("上游错误：%s，非法参数：%v", ue.Title, ue.InvalidParams)
}
```

错误类别：

| sentinel | 含义 |
| --- | --- |
| `ErrInvalidConfig` | 构造参数有问题（缺认证、空私钥、空 API Host） |
| `ErrInvalidLocation` | 位置形态这个端点不认 |
| `ErrInvalidParameter` | 端点参数越界 |
| `ErrInvalidResponse` | 上游返回的不是合法 JSON |
| `ErrUpstreamRejected` | 上游 4xx |
| `ErrQuotaExceeded` | 上游 429 |
| `ErrUnauthorized` | 上游 401 / 403 |
| `ErrUpstreamUnavailable` | 上游 5xx 或网络错误 |

客户端先校验，参数越界与位置形态不对都在本地返回，不打上游。上游 5xx 与网络
错误会按退避重试，最多 3 次；4xx 不重试。

### 并发与超时

`qwx.Client` 无状态，可以直接共享。所有方法都接受 `context.Context`，
超时与取消由 ctx 控制。

要换 HTTP 客户端就注入一个，注意 SDK 不会改它的超时：

```go
import (
	"net/http"
	"time"

	"github.com/KanoCifer/qwx"
)

base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com",
	qwx.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
```

### 测试

端点子包只依赖 `qwx.Doer` 接口，可以注入假实现测自己的逻辑：

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/weather"
)

type fakeDoer struct{}

func (fakeDoer) Do(_ context.Context, req qwx.Request, out any) error { ... }

w := weather.New(fakeDoer{})
```

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

包结构：

| 包 | 承载 |
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
| `qwx/rediscache` | Redis 缓存实现（独立模块，依赖 go-redis） |

一个包 = 一个上游资源族。`geo` 是唯一的例外：它的四个检索端点共享同一个地点
类型，拆开会让该类型失去归属。

`rediscache` 是独立的 Go 模块（自带 `go.mod`），不是上表的子包 —— 它不提供
接口，只提供缓存实现，依赖 go-redis 所以不并入核心模块。

`Tide`、`Indices`、`Minutely` 至今只有 v7 版本：实测 `/weather/v1/indices/*`、
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

**新一代只认坐标。** `/weather/v1/current/101010100` 回 404（body 为空）。
LocationID 要先用 `geo` 包的 `LookupCity` 换成坐标。

**必须传 `localTime=true`。** 新一代默认给 UTC（`"2026-10-21T05:00Z"`），
不传本地时间会整整差 8 小时。

**湿度与云量是比例不是百分数。** 新一代的 `humidity: 0.69` 是 69%。

**潮汐日期要用站点当地当天。** 潮汐窗口下界是当地当天，用 UTC 日期会在
北京时间凌晨 8 点前算成「昨天」，上游回 400。

**分钟级降水只认坐标、只覆盖中国。** `location` 必须是 `"经度,纬度"`，
不认 LocationID 也不认地名。

**生活指数只支持 1 天与 3 天。** 传 2 天、7 天、30 天全部 404，裸经度（如
`121.47`）上游回 400。

**POI 检索无数据时回 400 而不是空列表。** 上游在该类型附近找不到 POI 时
返回 400 `No Such Location`：内陆城市（如北京）查 TSTA（潮汐站）与 CSTA
（观测站）都会 400，沿海城市（如舟山）才查得到。调用方要按错误处理，
不能假设「查不到就是空」。

**Redis 缓存是独立模块，不是核心包的一部分。** `rediscache` 自带 `go.mod`
依赖 go-redis，核心包保持零依赖。引入方式：

```bash
go get github.com/KanoCifer/qwx/rediscache
```

它的构造函数不连 Redis，地址写错要靠 `cache.Ping(ctx)` 才暴露。`Set` 的 ttl
小于等于 0 会返回 `qwx.ErrInvalidParameter` 且不写入 —— 不带过期时间的缓存
会一直是旧数据。

## 官方文档

- 开发文档：https://dev.qweather.com/docs/
- 身份认证：https://dev.qweather.com/docs/configuration/authentication/
- 最佳实践（不要假设）：https://dev.qweather.com/docs/best-practices/no-assumptions/

## 许可

本包遵循 MIT 许可，见 [LICENSE](LICENSE)。

本包不是和风天气官方产品。使用上游数据需遵守
[和风天气开发者许可协议](https://www.qweather.com/terms/developers-eula/)，
展示天气数据时需保留上游返回的 attribution 信息。
