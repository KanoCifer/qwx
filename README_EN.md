<p align="center">
  <img src="./assets/banner_en.png" width="100%" alt="qwx" />
</p>

<p align="center">
  <a href="https://github.com/KanoCifer/qwx/actions/workflows/ci.yml" target="_blank" rel="noopener"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/KanoCifer/qwx/ci.yml?branch=main" height="20"></a>
  <a href="https://go.dev" target="_blank" rel="noopener"><img alt="Go" src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white" height="20"></a>
  <a href="https://github.com/KanoCifer/qwx/releases" target="_blank" rel="noopener"><img alt="Release" src="https://img.shields.io/github/v/release/KanoCifer/qwx?display_name=tag" height="20"></a>
  <a href="https://github.com/KanoCifer/qwx/blob/main/LICENSE" target="_blank" rel="noopener"><img alt="License" src="https://img.shields.io/github/license/KanoCifer/qwx" height="20"></a>
  <a href="https://pkg.go.dev/github.com/KanoCifer/qwx" target="_blank" rel="noopener"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/KanoCifer/qwx.svg" height="20"></a>
  <a href="https://goreportcard.com/report/github.com/KanoCifer/qwx" target="_blank" rel="noopener"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/KanoCifer/qwx" height="20"></a>
  <a href="https://github.com/KanoCifer/qwx#supported-endpoints" target="_blank" rel="noopener"><img alt="Endpoints" src="https://img.shields.io/badge/endpoints-25-blue" height="20"></a>
  <a href="https://github.com/KanoCifer/qwx/blob/main/go.mod" target="_blank" rel="noopener"><img alt="Core deps" src="https://img.shields.io/badge/core-0%20deps-success" height="20"></a>
  <a href="https://github.com/KanoCifer/qwx/stargazers" target="_blank" rel="noopener"><img alt="Stars" src="https://img.shields.io/github/stars/KanoCifer/qwx?style=social" height="20"></a>
</p>

<p align="center">
  <a href="README_EN.md"><b>English</b></a> · <a href="README.md">简体中文</a>
</p>

QWeather ships official SDKs for iOS and Android only — there is no official server-side implementation. qwx is a community-maintained Go SDK that covers the core endpoint families from the upstream documentation, returns typed Go structs, and keeps the core package free of third-party dependencies.

---

<details>
<summary><b>📖 目录</b></summary>

- [✨ Features](#features)
- [📦 Installation](#installation)
- [🚀 Quick start](#quick-start)
- [📖 Usage guide](#usage-guide)
  - [🔑 Authentication](#authentication)
  - [⚙️ Client options](#client-options)
  - [📍 Location](#location)
  - [🌐 Multi-language](#multi-language)
  - [🌤️ Weather](#weather)
  - [🌫️ Air quality](#air-quality)
  - [⚠️ Weather alerts](#weather-alerts)
  - [🗺️ GeoAPI lookup](#geoapi-lookup)
  - [🌊 Tides](#tides)
  - [🧭 Life indices](#life-indices)
  - [🌧️ Minute-level precipitation](#minute-level-precipitation)
  - [☀️ Solar radiation forecast](#solar-radiation-forecast)
  - [🌀 Typhoons](#typhoons)
  - [🔭 Astronomy](#astronomy)
  - [⏳ Historical weather](#historical-weather)
  - [💼 Console](#console)
  - [🗄️ Caching](#caching)
  - [📝 Logging](#logging)
  - [🚨 Error handling](#error-handling)
  - [⏱️ Concurrency and timeouts](#concurrency-and-timeouts)
  - [🧪 Testing](#testing)
- [🔌 Supported endpoints](#supported-endpoints)
- [🧩 The bundled Redis cache](#the-bundled-redis-cache)
- [📚 Documentation](#documentation)
- [🔗 Official documentation](#official-documentation)
- [⚖️ License](#license)

</details>

## ✨ Features

- **Broad endpoint coverage**: GeoAPI city and POI lookup, weather forecast (the new generation, `/weather/v1/*`), air quality, weather alerts, solar radiation forecast, ocean tides, life indices, minute-level precipitation, typhoons, astronomy, and historical weather
- **Every live endpoint covered (as of 2026-10-29)**: the upstream documentation site's 28 endpoint pages are all covered while still in service — 25 are wired up, and the 3 left out (`/v7/weather/now`, `/v7/weather/{days}`, `/v7/weather/{hours}`) are deprecated, shut down on 2027-02-01, and each has a new-generation counterpart. The upstream OpenAPI spec also carries 8 legacy v7 paths the documentation site does not publish (`/v7/warning/now`, `/v7/air/now`, `/v7/air/5d`, `/v7/solar-radiation/{hours}`, `/v7/grid-weather/*`, `/v7/historical/air`); they are either shut down or deprecated, and none of them is wired up
- **One package per resource family**: each package maps to a single upstream resource family, so field shapes and cache keys never bleed across families
- **Two authentication modes**: JWT (Ed25519-signed, the official recommendation) and API KEY
- **Zero dependencies**: the core package uses only the standard library — `go.mod` has no `require`
- **Pluggable cache**: a `Cache` interface accepts Redis, in-memory, or custom implementations
- **Pluggable logging**: built on the standard library `log/slog`; nothing is logged unless a logger is injected
- **Classified errors**: use `errors.Is` to check the error category instead of parsing error strings
- **Safe for concurrent use**: `Client` is stateless and can be shared directly

---

## 📦 Installation

```bash
go get github.com/KanoCifer/qwx
```

Requires Go 1.24 or later.

For a Redis cache, add the separate cache module. It depends on go-redis v9.23.0, which requires Go 1.26; the core module is unaffected:

```bash
go get github.com/KanoCifer/qwx/rediscache
```

---

## 🚀 Quick start

Three steps: build the auth, build the base client, build the client for an endpoint family.

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
	// 1. Auth. Use qwx.NewJWTAuth for JWT, qwx.NewAPIKeyAuth for API KEY.
	auth, err := qwx.NewJWTAuth(qwx.JWTConfig{
		PrivateKeyPEM: os.Getenv("QWEATHER_JWT_PRIVATE_KEY"),
		KeyID:         os.Getenv("QWEATHER_KEY_ID"),
		ProjectID:     os.Getenv("QWEATHER_PROJECT_ID"),
		DeveloperID:   os.Getenv("QWEATHER_DEVELOPER_ID"), // required, from the console "Settings" page
	})
	if err != nil {
		log.Fatal(err)
	}

	// 2. Base client. apiHost is the dedicated Host assigned on the console
	// "Settings" page and is required.
	base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com")
	if err != nil {
		log.Fatal(err)
	}

	// 3. Each endpoint family builds its own service client from the base client.
	w := weather.New(base)

	ctx := context.Background()
	loc, err := qwx.ParseLocation("116.41,39.92") // longitude first
	if err != nil {
		log.Fatal(err)
	}

	current, err := w.Current(ctx, loc, weather.Options{
		Lang:      qwx.LangZH,
		LocalTime: true, // without this you get UTC, which is 8 hours off
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(current.Condition.Text, current.Temperature.Value, current.Temperature.Unit)
}
```

The `examples/basic` directory in this repository holds a longer example that covers weather, hourly and daily forecasts, city lookup, and life indices. It uses API KEY authentication and reads its configuration from environment variables:

```bash
export QWEATHER_API_KEY="your API KEY"
export QWEATHER_API_HOST="abcxyz.qweatherapi.com"
go run ./examples/basic
```

The API Host can only be obtained from the console "Settings" page; there is no default. The public domains `devapi.qweather.com` (shut down 2026-01), `api.qweather.com`, and `geoapi.qweather.com` (shut down 2026-06) have all expired. When the Host is given as a bare domain without a scheme, the SDK prepends `https://` automatically.

---

## 📖 Usage guide

### 🔑 Authentication

The two modes cannot be combined — sending both makes the upstream reject the request. JWT is the default choice.

JWT requires an Ed25519 key pair. Upload the public key to the console:

```bash
openssl genpkey -algorithm ED25519 -out ed25519-private.pem
openssl pkey -pubout -in ed25519-private.pem > ed25519-public.pem
```

After creating a credential in the console, obtain the Key ID, Project ID, and Developer ID. All three are required:

```go
import "github.com/KanoCifer/qwx"

auth, err := qwx.NewJWTAuth(qwx.JWTConfig{
	PrivateKeyPEM: string(privateKeyPEM),
	KeyID:         "YOUR_KEY_ID",
	ProjectID:     "YOUR_PROJECT_ID",
	DeveloperID:   "YOUR_DEVELOPER_ID",
})
```

Each value maps to one position in the JWT: `kid` is the credential ID, `sub` is the project ID, and `iss` is the Developer ID (from the console "Settings" page, shaped like a 10-character alphanumeric string starting with Q). The upstream treats `iss` as a mandatory issuer and validates credential ownership against it, so the SDK returns `ErrInvalidConfig` at construction time instead of waiting for an upstream 401 to reveal the misconfiguration.

A JWT is valid for 15 minutes. The SDK re-signs it 1 minute before expiry, so callers never handle it. Signing is guarded by a lock and is safe for concurrent use.

API KEY is not recommended and suits existing projects only:

```go
import "github.com/KanoCifer/qwx"

auth, err := qwx.NewAPIKeyAuth("YOUR_API_KEY")
```

> Officially announced: from 2027-01-01 the daily request volume for API KEY authentication is restricted, and SDK 5+ no longer supports API KEY. New projects should use JWT.

### ⚙️ Client options

Every option is passed to the base client constructor:

| Option | Effect |
| --- | --- |
| `qwx.WithCache` | Injects a `Cache` implementation; without it every request reaches the upstream |
| `qwx.WithTTLPolicy` | Determines cache duration per endpoint |
| `qwx.WithLogger` | Injects a `log/slog` logger; the default is silent |
| `qwx.WithHTTPClient` | Replaces the underlying `http.Client`; the SDK does not change its timeout |
| `qwx.WithAuth` | Overrides the auth mode, equivalent to the `auth` argument of the constructor |
| `qwx.WithClock` | Injects a clock for asserting the JWT `iat` and `exp` in tests |

```go
base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com",
	qwx.WithCache(myRedisCache),
	qwx.WithTTLPolicy(qwx.TTLMap{weather.EndpointCurrent: 30 * time.Second}),
	qwx.WithLogger(slog.Default()),
)
```

### 📍 Location

`qwx.Location` unifies the three location forms the upstream accepts:

```go
import (
	"fmt"

	"github.com/KanoCifer/qwx"
)

// Coordinate pair; the string puts longitude first
coord, err := qwx.ParseLocation("116.41,39.92")
if err != nil {
	return err
}
fmt.Println(coord.Lat, coord.Lon)

// Construct directly; fields put latitude first
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

// LocationID
station := qwx.Location{ID: "101010100"}

// Place name, only GeoAPI city/POI lookup accepts it
city := qwx.Location{Name: "杭州"}

fmt.Println(loc, station, city)
```

Each endpoint accepts a different subset of these forms:

| Form | Accepted by |
| --- | --- |
| Coordinate pair | All endpoints |
| LocationID | `geo.LookupCity`, `geo.LookupPOI`, `astronomy.Sun`, `astronomy.Moon`; administrative divisions are 9~10 digits, POIs are digits with a letter suffix, tide stations are `P` plus digits |
| Place name | Only `geo.LookupCity` and `geo.LookupPOI` |

The new generation accepts coordinates only. To convert a LocationID, call `geo.LookupCity` first.

Coordinate order differs between generations: the new generation uses path parameters with latitude first (`Location.Path()`), while v7 and GeoAPI use query parameters with longitude first (`Location.Query()`). Reversing them makes the upstream return 400.

### 🌐 Multi-language

The upstream switches the response language with the `lang` query parameter. Languages are represented by `qwx.Lang`; the SDK ships the 31 languages listed on the official "Multi-language" page:

| Language | Variable | API code |
| --- | --- | --- |
| Simplified Chinese | `qwx.LangZH` | `zh` |
| Traditional Chinese | `qwx.LangZHHant` | `zh-hant` |
| English | `qwx.LangEN` | `en` |
| German | `qwx.LangDE` | `de` |
| Spanish | `qwx.LangES` | `es` |
| French | `qwx.LangFR` | `fr` |
| Italian | `qwx.LangIT` | `it` |
| Japanese | `qwx.LangJA` | `ja` |
| Korean | `qwx.LangKO` | `ko` |
| Russian | `qwx.LangRU` | `ru` |
| Hindi | `qwx.LangHI` | `hi` |
| Thai | `qwx.LangTH` | `th` |
| Arabic | `qwx.LangAR` | `ar` |
| Portuguese | `qwx.LangPT` | `pt` |
| Bengali | `qwx.LangBN` | `bn` |
| Malay | `qwx.LangMS` | `ms` |
| Dutch | `qwx.LangNL` | `nl` |
| Greek | `qwx.LangEL` | `el` |
| Latin | `qwx.LangLA` | `la` |
| Swedish | `qwx.LangSV` | `sv` |
| Indonesian | `qwx.LangID` | `id` |
| Polish | `qwx.LangPL` | `pl` |
| Turkish | `qwx.LangTR` | `tr` |
| Czech | `qwx.LangCS` | `cs` |
| Estonian | `qwx.LangET` | `et` |
| Vietnamese | `qwx.LangVI` | `vi` |
| Filipino | `qwx.LangFIL` | `fil` |
| Finnish | `qwx.LangFI` | `fi` |
| Hebrew | `qwx.LangHE` | `he` |
| Icelandic | `qwx.LangIS` | `is` |
| Norwegian | `qwx.LangNB` | `nb` |

An empty language (the zero value `qwx.Lang{}`) uses the upstream default: the official language of the country or region, falling back to English when there is none.

```go
cur, err := w.Current(ctx, loc, weather.Options{Lang: qwx.LangZH, LocalTime: true})
```

Look a language up by code; the equivalent codes the official page lists are recognised too:

```go
lang, ok := qwx.LookupLang("zh-hans") // another code for Simplified Chinese
```

A malformed language code returns `ErrInvalidParameter` without hitting the upstream. The solar radiation forecast, tides, typhoons, sunrise/sunset and solar elevation angle endpoints have no language parameter — the upstream does not accept one.

### 🌤️ Weather

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/weather"
)

w := weather.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

current, err := w.Current(ctx, loc, weather.Options{Lang: qwx.LangZH, LocalTime: true})
if err != nil {
	return err
}
fmt.Println(current.Condition.Text, current.Temperature.Value, current.Temperature.Unit)
fmt.Println(current.Humidity*100, "%") // humidity is a 0~1 ratio

hourly, err := w.Hourly(ctx, loc, 24, weather.Options{Lang: qwx.LangZH, LocalTime: true})
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

daily, err := w.Daily(ctx, loc, 3, weather.Options{Lang: qwx.LangZH, LocalTime: true})
if err != nil {
	return err
}
for _, d := range daily.Days {
	fmt.Println(d.ForecastStartTime, d.TemperatureMax.Value, d.TemperatureMin.Value)
}
```

`Hourly` takes 1~240 hours and `Daily` takes 1~10 days. Out-of-range values return `ErrInvalidParameter` without reaching the upstream. Do not assume list lengths: the upstream may return fewer entries.

### 🌫️ Air quality

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/airquality"
)

aq := airquality.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

cur, err := aq.Current(ctx, loc, airquality.Options{Lang: qwx.LangZH})
if err != nil {
	return err
}
for _, idx := range cur.Indexes {
	fmt.Println(idx.Code, idx.AQI, idx.Category)
}

// Nearby monitoring stations, then measured data per station
for _, s := range cur.Stations {
	station, err := aq.Station(ctx, s.ID, airquality.Options{Lang: qwx.LangZH})
	if err != nil {
		return err
	}
	fmt.Println(s.Name, station.Pollutants)
}

// Daily and hourly forecasts
daily, err := aq.Daily(ctx, loc, airquality.Options{Lang: qwx.LangZH})
if err != nil {
	return err
}
hourly, err := aq.Hourly(ctx, loc, airquality.Options{Lang: qwx.LangZH})
if err != nil {
	return err
}
fmt.Println(len(daily.Days), len(hourly.Hours))
```

AQI values use different scales per standard (QAQI is 0~1, US EPA is 0~500), so do not compare across standards.

### ⚠️ Weather alerts

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/alert"
)

al := alert.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

result, err := al.Current(ctx, loc, alert.Options{Lang: qwx.LangZH, LocalTime: true})
if err != nil {
	return err
}
for _, a := range result.Alerts {
	fmt.Println(a.SenderName, a.Headline, a.Severity, a.EffectiveTime)
}
```

Alerts accept coordinates only. When there is no alert, the upstream returns an empty list and `Metadata.ZeroResult` points to `true`.

### 🗺️ GeoAPI lookup

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/geo"
)

g := geo.New(base)
ctx := context.Background()

// City lookup by place name
cities, err := g.LookupCity(ctx, qwx.Location{Name: "杭州"}, geo.Options{Lang: qwx.LangZH, Number: 10})
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

// Nearby tide stations
pois, err := g.LookupPOI(ctx, qwx.Location{Lat: 30.0, Lon: 122.0}, geo.POITSTA, geo.Options{Lang: qwx.LangZH})
if err != nil {
	return err
}
for _, p := range pois.POIs {
	fmt.Println(p.ID, p.Name)
}

// Popular cities
top, err := g.TopCities(ctx, geo.Options{Lang: qwx.LangZH})
if err != nil {
	return err
}
for _, c := range top.Cities {
	fmt.Println(c.DisplayName())
}
```

`POIRange` searches for POIs near a coordinate. It accepts coordinates only, and `Radius` is in kilometres:

```go
ranges, err := g.POIRange(ctx, qwx.Location{Lat: 30.0, Lon: 122.0}, geo.POIScenic,
	geo.POIRangeOptions{Lang: qwx.LangZH, Number: 10, Radius: 20})
if err != nil {
	return err
}
for _, p := range ranges.POIs {
	fmt.Println(p.ID, p.Name)
}
```

The four GeoAPI lookup endpoints share one set of optional parameters:

| Parameter | Effect | Accepted by |
| --- | --- | --- |
| `Lang` | Language; empty uses the upstream default | All |
| `Number` | Maximum number of results; 0 uses the upstream default (10) | All |
| `Adm` | LocationID of the parent administrative division, to filter out duplicate city names | City lookup |
| `Range` | Search range as an ISO 3166 country code, such as `cn` or `world` | City lookup, popular cities |
| `City` | Restricts POIs to a city, by city name or LocationID | POI lookup |

Every GeoAPI result carries a `Refer` describing the data source and its licence. Display it alongside the data:

```go
for _, s := range cities.Refer.Sources {
	fmt.Println(s)
}
```

The five legacy (v7) endpoint families (tides, life indices, minute-level precipitation, typhoons, astronomy) carry the same `Refer` field, accessed the same way.

### 🌊 Tides

Look up the tide station ID with `geo.LookupPOI` and `geo.POITSTA`, and use the station's local calendar day:

```go
import (
	"context"
	"time"

	"github.com/KanoCifer/qwx/v7/ocean"
)

o := ocean.New(base)
ctx := context.Background()

// date is the station's local date, formatted YYYYMMDD
date := time.Now().Format("20060102")
result, err := o.Tide(ctx, "P2419", date)
if err != nil {
	return err
}
for _, t := range result.TideTable {
	fmt.Println(t.FXTime, t.Height, t.Type) // Type: H high tide / L low tide
}
```

Tide height stays a string, preserving the decimal precision given by the upstream.

### 🧭 Life indices

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v7/indices"
)

idx := indices.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

// typ takes 0~16, where 0 means all; days supports only 1 and 3
result, err := idx.Indices(ctx, loc, indices.TypeFishing, 1, "zh")
if err != nil {
	return err
}
for _, d := range result.Daily {
	fmt.Println(d.Date, d.Name, d.Category, d.Text)
}

// Several indices in one call: the upstream type parameter accepts comma-separated values
multi, err := idx.IndicesMulti(ctx, loc, []int{4, 5}, 1, "zh")
if err != nil {
	return err
}
fmt.Println(len(multi.Daily))
```

### 🌧️ Minute-level precipitation

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

This endpoint covers China only; the upstream has no data for coordinates outside the country. `Precip` is a string: an empty string means "no data", while `"0.00"` means "it is genuinely not raining".

### ☀️ Solar radiation forecast

A new-generation endpoint that accepts coordinates only, with latitude first in the path:

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v1/solarradiation"
)

sr := solarradiation.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}

// hours takes 1~72; interval accepts only 15, 30, or 60
result, err := sr.Forecast(ctx, loc, 24, solarradiation.Options{
	Interval: 60,
	Extra:    "weather",
})
if err != nil {
	return err
}
for _, f := range result.Forecasts {
	fmt.Println(f.ForecastTime, f.GHI.Value, f.GHI.Unit)
}
```

For array-plane irradiance, set `Extra` to `"poa"` and supply both `Tilt` and `Azimuth`. Both fields are pointers, which distinguishes "not sent" from "sent as 0":

```go
tilt, azimuth := 30, 180
result, err := sr.Forecast(ctx, loc, 24, solarradiation.Options{
	Interval: 60,
	Extra:    "poa",
	Tilt:     &tilt,
	Azimuth:  &azimuth,
})
if err != nil {
	return err
}
for _, f := range result.Forecasts {
	fmt.Println(f.ForecastTime, f.POA.Global.Value, f.POA.Global.Unit)
}
```

This family has no `lang` parameter; the upstream returns no text descriptions and rejects requests that carry one.

### 🌀 Typhoons

The three endpoints take no location and are queried by basin and typhoon ID:

```go
import (
	"context"

	"github.com/KanoCifer/qwx/v7/storm"
)

st := storm.New(base)
ctx := context.Background()

// the upstream currently supports only the NP basin
list, err := st.List(ctx, storm.BasinNP, 2026)
if err != nil {
	return err
}
for _, s := range list.Storms {
	fmt.Println(s.ID, s.Name, s.IsActive) // IsActive: "1" active / "0" closed
}

forecast, err := st.Forecast(ctx, "NP2026")
if err != nil {
	return err
}
fmt.Println(len(forecast.Forecast))

track, err := st.Track(ctx, "NP2026")
if err != nil {
	return err
}
fmt.Println(len(track.Track))
```

### 🔭 Astronomy

```go
import (
	"context"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v7/astronomy"
)

as := astronomy.New(base)
ctx := context.Background()
loc := qwx.Location{Lat: 39.92, Lon: 116.41}
date := time.Now().Format("20060102")

sun, err := as.Sun(ctx, loc, date)
if err != nil {
	return err
}
if sun.Sunrise != nil {
	fmt.Println(*sun.Sunrise, *sun.Sunset)
}

moon, err := as.Moon(ctx, loc, date, astronomy.Options{Lang: qwx.LangZH})
if err != nil {
	return err
}
fmt.Println(len(moon.MoonPhase))

// Solar elevation angle accepts coordinates only and also needs time, timezone, and altitude
angle, err := as.SolarElevationAngle(ctx, loc, date, "1200", "0800", 43)
if err != nil {
	return err
}
fmt.Println(angle.SolarElevationAngle, angle.SolarAzimuthAngle)
```

### ⏳ Historical weather

Historical weather (the upstream "time machine") accepts a LocationID only, and the date must fall in the last 10 days, excluding today.

```go
import (
	"context"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v7/historical"
)

hs := historical.New(base)
ctx := context.Background()
// The upstream serves the last 10 days only, excluding today
date := time.Now().AddDate(0, 0, -1).Format("20060102")

got, err := hs.Weather(ctx, qwx.Location{ID: "101010100"}, date, historical.Options{Lang: qwx.LangZH})
if err != nil {
	return err
}
if got.Daily != nil {
	fmt.Println(got.Daily.Date, got.Daily.TempMax, got.Daily.TempMin)
}
fmt.Println(len(got.Hourly))
```

### 💼 Console

The console API reports your own account's finance and request volume rather than
weather data. Credentials have no access by default: the account owner has to
enable it under "Project management - Credential - Console permissions" in the
console, otherwise upstream returns 403.

```go
import (
	"context"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/console"
)

c := console.New(base)
ctx := context.Background()

fin, err := c.Finance(ctx)
if err != nil {
	return err
}
fmt.Println(fin.Currency, fin.Balance, fin.AsOf)

// Pending bills
for _, b := range fin.PendingBills {
	fmt.Println(b.Number, b.Type, b.Status, b.AmountDue, b.DueDate)
}

// Request volume of the last 24 hours
stats, err := c.Stats(ctx, console.StatsOptions{})
if err != nil {
	return err
}
for _, s := range stats.Success {
	fmt.Println(s.API, s.Hours)
}
```

`Project` and `Credential` are mutually exclusive; leaving both empty covers
every project and credential of the account. Upstream returns a resource plan's
total request count as a string, so the SDK decodes it into a `json.Number`:

```go
for _, p := range fin.AvailableResourcePlans {
	n, _ := p.Requests.Int64()
	fmt.Println(p.BillNumber, n)
}
```

The data is cut off at the previous hour or earlier and may lag the console web
page by an hour or more — trust the `AsOf` field in the response.

### 🗄️ Caching

The SDK ships no cache implementation. Inject an existing one:

```go
import "github.com/KanoCifer/qwx"

base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithCache(myRedisCache))
```

The built-in in-memory cache `qwx.NewMemoryCache()` is **not recommended for production**: it has no background cleanup, expired entries are deleted only when they are hit, and memory usage in a long-running process only grows. Use it as a test double or a temporary single-instance cache:

```go
import "github.com/KanoCifer/qwx"

base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithCache(qwx.NewMemoryCache()))
```

For Redis, add the `rediscache` module, which implements `qwx.Cache`:

```go
import (
	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/rediscache"
)

cache, err := rediscache.New(rediscache.Options{
	Addr: "127.0.0.1:6379",
	// Password: "...",  set when authentication is required
	// KeyPrefix: "myapp:",  isolates keys when several apps share one Redis instance
})
if err != nil {
	return err
}
defer cache.Close()

base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithCache(cache))
```

`rediscache` supports TLS (`TLSConfig`), ACL usernames (`Username`), a database number (`DB`), and connection timeouts. The constructor does not connect to Redis; use `cache.Ping(ctx)` for a startup self-check.

Cache keys are generated by the SDK (the two generations deliberately share no keys, so old entries can never feed a new parser), and TTL follows each endpoint's data update cadence. Cache keys do not include the API Host, so one `Cache` instance should serve only one Host.

To tune TTL per endpoint, supply a policy:

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
		airquality.EndpointCurrent: 0, // do not cache this endpoint
	}),
)
```

Endpoint constants live in their endpoint packages (`weather.EndpointCurrent`, `geo.EndpointCity`, and so on). A package that exports a single endpoint names it `Endpoint` (`alert.Endpoint`, `ocean.Endpoint`). Default durations:

| Endpoint | Endpoint constant | Default TTL |
| --- | --- | --- |
| Weather current | `weather.EndpointCurrent` | 10 minutes |
| Hourly and daily forecast | `weather.EndpointHourly`, `weather.EndpointDaily` | 1 hour |
| Air quality current and station | `airquality.EndpointCurrent`, `airquality.EndpointStation` | 30 minutes |
| Air quality forecast | `airquality.EndpointDaily`, `airquality.EndpointHourly` | 1 hour |
| Weather alerts | `alert.Endpoint` | 5 minutes |
| Minute-level precipitation | `minutely.Endpoint` | 5 minutes |
| Tides and life indices | `ocean.Endpoint`, `indices.Endpoint` | 1 hour |
| Solar radiation forecast | `solarradiation.Endpoint` | 1 hour |
| Typhoon list | `storm.EndpointList` | 24 hours |
| Typhoon forecast and track | `storm.EndpointForecast`, `storm.EndpointTrack` | 1 hour |
| Astronomy | `astronomy.EndpointSun` and two others | 24 hours |
| GeoAPI | `geo.EndpointCity` and three others | 24 hours |

Use `qwx.TTLPolicyFunc` to adjust durations dynamically by time or load. To bypass the cache for a single request, use `qwx.WithTTLOverride(ctx, 0)`, which takes the highest precedence.

> Changing the TTL does not change the cache key: entries already written expire according to the TTL in effect when they were written. To apply a new TTL immediately, replace the cache instance or clear the keys manually.

### 📝 Logging

```go
import "github.com/KanoCifer/qwx"

base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithLogger(slog.Default()))
```

The handler decides the level; the SDK does not filter. What gets logged:

| Level | Message | Trigger |
| --- | --- | --- |
| Debug | `cache hit` / `cache miss` | Cache hit or miss |
| Warn | `cache read failed` / `cache write failed` | Cache failure; the request degrades to a direct upstream call |
| Info | `upstream request` | Upstream returned 2xx |
| Warn | `upstream rejected` | Upstream returned 4xx |
| Error | `upstream rejected` | Upstream returned 5xx |
| Warn | `upstream retry` | A 5xx or network error triggered a retry |
| Error | `upstream request failed` | Network error or timeout |
| Error | `decode response failed` | The response is not valid JSON |
| Error | `build request failed` / `sign request failed` | Request construction or signing failed |

Fields include `endpoint`, `method`, `path`, `status`, `duration_ms`, `cache_key`, `attempt`, `error`, `upstream_code`, and `upstream_title`.

Credentials travel in the request header and never reach the log; neither does the response body; and URLs in error text have their query strings stripped. Paths and cache keys may contain coordinates, and the SDK does not mask them — wrap your own handler when injecting a logger if masking is required.

The default logger is `slog.DiscardHandler`: it builds no fields, so the per-request overhead is a single `Enabled` call.

### 🚨 Error handling

Use `errors.Is` to check the category and `errors.As` to extract upstream details:

```go
import (
	"errors"

	"github.com/KanoCifer/qwx"
)

if errors.Is(err, qwx.ErrQuotaExceeded) {
	// upstream 429, retry later
} else if errors.Is(err, qwx.ErrUpstreamRejected) {
	// upstream 4xx, the request itself is invalid and retrying will not help
} else if errors.Is(err, qwx.ErrUnauthorized) {
	// invalid credentials or insufficient permissions
} else if errors.Is(err, qwx.ErrInvalidLocation) {
	// this endpoint does not accept this location form
}

var ue *qwx.UpstreamError
if errors.As(err, &ue) {
	log.Printf("upstream error: %s, invalid params: %v", ue.Title, ue.InvalidParams)
}
```

Error categories:

| Sentinel | Meaning |
| --- | --- |
| `ErrInvalidConfig` | Invalid constructor arguments (missing auth, empty private key, empty API Host) |
| `ErrInvalidLocation` | The endpoint does not accept this location form |
| `ErrInvalidParameter` | An endpoint parameter is out of range |
| `ErrInvalidResponse` | The upstream did not return valid JSON |
| `ErrUpstreamRejected` | Upstream 4xx |
| `ErrQuotaExceeded` | Upstream 429 |
| `ErrUnauthorized` | Upstream 401 / 403 |
| `ErrUpstreamUnavailable` | Upstream 5xx or a network error |

The client validates first: out-of-range parameters and unsupported location forms return locally without contacting the upstream. Upstream 5xx responses and network errors are retried with backoff at 1, 2, and 3 seconds, up to 3 retries; 4xx responses are not retried.

### ⏱️ Concurrency and timeouts

`qwx.Client` is stateless and can be shared directly. Every method takes a `context.Context`, so timeouts and cancellation are controlled by the context.

The default HTTP timeout is `qwx.DefaultTimeout` (30 seconds). To use a different HTTP client, inject one — note that the SDK does not modify its timeout:

```go
import (
	"net/http"
	"time"

	"github.com/KanoCifer/qwx"
)

base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com",
	qwx.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}))
```

### 🧪 Testing

Endpoint packages depend only on the `qwx.Doer` interface, so a fake implementation can be injected to test their logic in isolation:

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

---

## 🔌 Supported endpoints

| Package | Method | Upstream endpoint | Endpoint constant |
| --- | --- | --- | --- |
| `geo` | `LookupCity` | `GET /geo/v2/city/lookup` | `EndpointCity` |
| `geo` | `LookupPOI` | `GET /geo/v2/poi/lookup` | `EndpointPOI` |
| `geo` | `POIRange` | `GET /geo/v2/poi/range` | `EndpointPOIRange` |
| `geo` | `TopCities` | `GET /geo/v2/city/top` | `EndpointTopCities` |
| `v1/weather` | `Current` | `GET /weather/v1/current/{lat}/{lon}` | `EndpointCurrent` |
| `v1/weather` | `Hourly` | `GET /weather/v1/hourly/{lat}/{lon}` | `EndpointHourly` |
| `v1/weather` | `Daily` | `GET /weather/v1/daily/{lat}/{lon}` | `EndpointDaily` |
| `v1/airquality` | `Current` | `GET /airquality/v1/current/{lat}/{lon}` | `EndpointCurrent` |
| `v1/airquality` | `Daily` | `GET /airquality/v1/daily/{lat}/{lon}` | `EndpointDaily` |
| `v1/airquality` | `Hourly` | `GET /airquality/v1/hourly/{lat}/{lon}` | `EndpointHourly` |
| `v1/airquality` | `Station` | `GET /airquality/v1/stations/{locationId}` | `EndpointStation` |
| `v1/alert` | `Current` | `GET /weatheralert/v1/current/{lat}/{lon}` | `Endpoint` |
| `v1/solarradiation` | `Forecast` | `GET /solarradiation/v1/forecast/{lat}/{lon}` | `Endpoint` |
| `v7/ocean` | `Tide` | `GET /v7/ocean/tide` | `Endpoint` |
| `v7/indices` | `Indices` | `GET /v7/indices/{days}d` | `Endpoint` |
| `v7/indices` | `IndicesMulti` | `GET /v7/indices/{days}d` | `Endpoint` |
| `v7/minutely` | `Minutely` | `GET /v7/minutely/5m` | `Endpoint` |
| `v7/storm` | `List` | `GET /v7/tropical/storm-list` | `EndpointList` |
| `v7/storm` | `Forecast` | `GET /v7/tropical/storm-forecast` | `EndpointForecast` |
| `v7/storm` | `Track` | `GET /v7/tropical/storm-track` | `EndpointTrack` |
| `v7/astronomy` | `Sun` | `GET /v7/astronomy/sun` | `EndpointSun` |
| `v7/astronomy` | `Moon` | `GET /v7/astronomy/moon` | `EndpointMoon` |
| `v7/astronomy` | `SolarElevationAngle` | `GET /v7/astronomy/solar-elevation-angle` | `EndpointAngle` |
| `v7/historical` | `Weather` | `GET /v7/historical/weather` | `Endpoint` |
| `console` | `Finance` | `GET /finance/v1/summary` | `EndpointFinance` |
| `console` | `Stats` | `GET /metrics/v1/stats` | `EndpointStats` |

Package layout:

| Package | Contents |
| --- | --- |
| `qwx` | Base client, authentication, cache, `Location`, error classification, the request struct shared by both generations, `Refer`, `MetaData` |
| `qwx/v1` | Base types shared across new-generation endpoint families (`Metadata`, `Value`) |
| `qwx/v1/weather` | New-generation weather `/weather/v1/*` |
| `qwx/v1/airquality` | New-generation air quality `/airquality/v1/*` |
| `qwx/v1/alert` | Weather alerts `/weatheralert/v1/*` |
| `qwx/v1/solarradiation` | New-generation solar radiation forecast `/solarradiation/v1/*` |
| `qwx/v7/ocean` | Legacy tides `/v7/ocean/tide` |
| `qwx/v7/indices` | Legacy life indices `/v7/indices/*` |
| `qwx/v7/minutely` | Legacy minute-level precipitation `/v7/minutely/*` |
| `qwx/v7/storm` | Typhoons `/v7/tropical/*` |
| `qwx/v7/astronomy` | Astronomy `/v7/astronomy/*` |
| `qwx/v7/historical` | Legacy historical weather `/v7/historical/weather` |
| `qwx/geo` | GeoAPI `/geo/v2/*` |
| `qwx/console` | Console `/finance/v1/*`, `/metrics/v1/*` |
| `qwx/rediscache` | Redis cache implementation (separate module, depends on go-redis) |

One package = one upstream resource family. There are two exceptions: `geo`'s four lookup endpoints share the same place type, and `console`'s two endpoints live under one upstream documentation page and share one response skeleton — splitting either would leave those types without a home.

`rediscache` is a standalone Go module with its own `go.mod`, not a subpackage in the table above — it provides no endpoints, only a cache implementation, and it is kept out of the core module because it depends on go-redis.

`Tide`, `Indices`, and `Minutely` exist only in v7 to this day: `/weather/v1/indices/*`, `/v1/indices/*`, and `/ocean/v1/tide/*` all return 404 in testing, and the official documentation still documents the v7 paths. Historical weather is the same: there is no new-generation counterpart.

---

## 🧩 The bundled Redis cache

```bash
go get github.com/KanoCifer/qwx/rediscache
```

Its constructor does not connect to Redis, so a wrong address only surfaces through `cache.Ping(ctx)`. A `Set` with a ttl of 0 or less returns `qwx.ErrInvalidParameter` and writes nothing — cache entries without an expiry keep serving stale data forever.

---

## 📚 Documentation

| Task | Document |
| --- | --- |
| Authentication, credentials, token expiry, 401/403 | [docs/auth.md](docs/auth.md) |
| Writing calls: parameter limits, location forms, return types, field units | [docs/contract.md](docs/contract.md) |
| Upstream behaviour that contradicts the docs: 400, empty results, time mismatches | [docs/gotchas.md](docs/gotchas.md) |
| Retry and degradation strategy, upstream error details | [docs/errors.md](docs/errors.md) |
| Bringing your own cache, tuning TTL, debugging cache mix-ups | [docs/caching.md](docs/caching.md) |
| Wiring up logging, inspecting request logs | [docs/logging.md](docs/logging.md) |
| Running tests, live smoke tests | [docs/verification.md](docs/verification.md) |

---

## 🔗 Official documentation

- Developer docs: https://dev.qweather.com/docs/
- Authentication: https://dev.qweather.com/docs/configuration/authentication/
- Best practices (do not assume): https://dev.qweather.com/docs/best-practices/no-assumptions/

---

## ⚖️ License

This package is released under the MIT license. See [LICENSE](LICENSE).

This package is not an official QWeather product. Using upstream data requires compliance with the
[QWeather Developer Licence Agreement](https://www.qweather.com/terms/developers-eula/), and
weather data displays must retain the attribution information returned by the upstream.
