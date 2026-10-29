//go:build live

// 本文件是打真实上游的冒烟测试，默认不参与 `go test ./...`。
//
// 用法：
//
//	QWEATHER_API_HOST=https://your-host.qweatherapi.com \
//	QWEATHER_JWT_PRIVATE_KEY="$(cat ed25519-private.pem)" \
//	QWEATHER_KEY_ID=YOUR_KEY_ID \
//	QWEATHER_PROJECT_ID=YOUR_PROJECT_ID \
//	QWEATHER_DEVELOPER_ID=YOUR_DEVELOPER_ID \
//	go test -tags live -v -run TestLive ./...
package live

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/console"
	"github.com/KanoCifer/qwx/geo"
	"github.com/KanoCifer/qwx/v1/airquality"
	"github.com/KanoCifer/qwx/v1/alert"
	"github.com/KanoCifer/qwx/v1/solarradiation"
	"github.com/KanoCifer/qwx/v1/weather"
	"github.com/KanoCifer/qwx/v7/astronomy"
	"github.com/KanoCifer/qwx/v7/historical"
	"github.com/KanoCifer/qwx/v7/indices"
	"github.com/KanoCifer/qwx/v7/minutely"
	"github.com/KanoCifer/qwx/v7/storm"
)

func liveClient(t *testing.T) *qwx.Client {
	t.Helper()
	host := os.Getenv("QWEATHER_API_HOST")
	priv := os.Getenv("QWEATHER_JWT_PRIVATE_KEY")
	if host == "" || priv == "" {
		t.Skip("未设置 QWEATHER_API_HOST / QWEATHER_JWT_PRIVATE_KEY，跳过")
	}
	auth, err := qwx.NewJWTAuth(qwx.JWTConfig{
		PrivateKeyPEM: priv,
		KeyID:         os.Getenv("QWEATHER_KEY_ID"),
		ProjectID:     os.Getenv("QWEATHER_PROJECT_ID"),
		DeveloperID:   os.Getenv("QWEATHER_DEVELOPER_ID"),
	})
	if err != nil {
		t.Fatalf("NewJWTAuth: %v", err)
	}
	c, err := qwx.NewClient(auth, host)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestLive(t *testing.T) {
	c := liveClient(t)
	ctx := context.Background()
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	w := weather.New(c)
	aq := airquality.New(c)
	al := alert.New(c)
	idx := indices.New(c)
	mi := minutely.New(c)
	g := geo.New(c)
	sr := solarradiation.New(c)
	st := storm.New(c)
	as := astronomy.New(c)
	hs := historical.New(c)
	cs := console.New(c)

	t.Run("Current", func(t *testing.T) {
		got, err := w.Current(ctx, loc, weather.Options{Lang: qwx.LangZH, LocalTime: true})
		if err != nil {
			t.Fatalf("Current: %v", err)
		}
		t.Logf("%s %v%s", got.Condition.Text, got.Temperature.Value, got.Temperature.Unit)
	})

	t.Run("Hourly", func(t *testing.T) {
		got, err := w.Hourly(ctx, loc, 24, weather.Options{Lang: qwx.LangZH, LocalTime: true})
		if err != nil {
			t.Fatalf("Hourly: %v", err)
		}
		t.Logf("%d 条", len(got.Hours))
	})

	t.Run("Daily", func(t *testing.T) {
		got, err := w.Daily(ctx, loc, 3, weather.Options{Lang: qwx.LangZH, LocalTime: true})
		if err != nil {
			t.Fatalf("Daily: %v", err)
		}
		t.Logf("%d 条", len(got.Days))
	})

	t.Run("LookupCity", func(t *testing.T) {
		got, err := g.LookupCity(ctx, qwx.Location{ID: "101010100"}, geo.Options{Lang: qwx.LangZH})
		if err != nil {
			t.Fatalf("LookupCity: %v", err)
		}
		for _, l := range got.Locations {
			t.Logf("%s %s", l.ID, l.DisplayName())
		}
	})

	t.Run("Indices", func(t *testing.T) {
		got, err := idx.Indices(ctx, loc, indices.TypeFishing, 1, qwx.LangZH)
		if err != nil {
			t.Fatalf("Indices: %v", err)
		}
		for _, d := range got.Daily {
			t.Logf("%s %s %s", d.Name, d.Category, d.Level)
		}
	})

	t.Run("Minutely", func(t *testing.T) {
		got, err := mi.Minutely(ctx, loc, qwx.LangZH)
		if err != nil {
			t.Fatalf("Minutely: %v", err)
		}
		t.Logf("%d 条 %s", len(got.Minutely), got.Summary)
	})

	t.Run("AirQuality", func(t *testing.T) {
		got, err := aq.Current(ctx, loc, airquality.Options{Lang: qwx.LangZH})
		if err != nil {
			t.Fatalf("AirQuality: %v", err)
		}
		for _, i := range got.Indexes {
			t.Logf("%s aqi=%v %s", i.Code, i.AQI, i.Category)
		}
	})

	t.Run("AirStation", func(t *testing.T) {
		// 监测站 ID 取自官方文档示例。上游导航里没挂这个端点，
		// 用的是 OpenAPI 规范里的路径 /airquality/v1/station(s)/{locationId}。
		got, err := aq.Station(ctx, "P58911", airquality.Options{Lang: qwx.LangZH})
		if err != nil {
			t.Fatalf("Station: %v", err)
		}
		for _, p := range got.Pollutants {
			t.Logf("%s %v%s", p.Code, p.Concentration.Value, p.Concentration.Unit)
		}
	})

	t.Run("Alerts", func(t *testing.T) {
		got, err := al.Current(ctx, loc, alert.Options{Lang: qwx.LangZH, LocalTime: true})
		if err != nil {
			t.Fatalf("Alerts: %v", err)
		}
		t.Logf("%d 条", len(got.Alerts))
	})

	t.Run("SolarRadiation", func(t *testing.T) {
		got, err := sr.Forecast(ctx, loc, 24, solarradiation.Options{Interval: 60})
		if err != nil {
			t.Fatalf("SolarRadiation: %v", err)
		}
		for _, f := range got.Forecasts {
			t.Logf("%s ghi=%v%s", f.ForecastTime, f.GHI.Value, f.GHI.Unit)
		}
	})

	t.Run("SolarRadiationHoursUpperBound", func(t *testing.T) {
		// 直接打上游，绕过 SDK 的客户端校验，探上游真实的上限。
		// hours=72 是官方文档页写的上限，hours=73 应当被上游拒绝。
		for _, hours := range []int{72, 73} {
			var out map[string]any
			req := qwx.Request{
				Method:   "GET",
				Path:     "/solarradiation/v1/forecast/39.92/116.41",
				Query:    url.Values{"hours": []string{strconv.Itoa(hours)}},
				Endpoint: solarradiation.Endpoint,
			}
			err := c.Do(ctx, req, &out)
			t.Logf("hours=%d err=%v", hours, err)
		}
	})

	t.Run("StormList", func(t *testing.T) {
		got, err := st.List(ctx, storm.BasinNP, time.Now().Year())
		if err != nil {
			t.Fatalf("StormList: %v", err)
		}
		for _, s := range got.Storms {
			t.Logf("%s %s active=%s", s.ID, s.Name, s.IsActive)
		}
	})

	t.Run("ConsoleFinance", func(t *testing.T) {
		got, err := cs.Finance(ctx)
		if err != nil {
			// 控制台权限要帐号所有者单独开通，没开通时上游回 403。
			if errors.Is(err, qwx.ErrUnauthorized) {
				t.Skip("凭据未开通控制台权限")
			}
			t.Fatalf("Finance: %v", err)
		}
		t.Logf("%s 余额=%v 截止=%s", got.Currency, got.Balance, got.AsOf)
	})

	t.Run("ConsoleStats", func(t *testing.T) {
		got, err := cs.Stats(ctx, console.StatsOptions{})
		if err != nil {
			if errors.Is(err, qwx.ErrUnauthorized) {
				t.Skip("凭据未开通控制台权限")
			}
			t.Fatalf("Stats: %v", err)
		}
		t.Logf("截止=%s，%d 个 API", got.AsOf, len(got.Success))
	})

	t.Run("HistoricalWeather", func(t *testing.T) {
		// 上游只给最近 10 天、不含今天的数据。
		date := time.Now().AddDate(0, 0, -1).Format("20060102")
		got, err := hs.Weather(ctx, qwx.Location{ID: "101010100"}, date, historical.Options{Lang: qwx.LangZH})
		if err != nil {
			t.Fatalf("HistoricalWeather: %v", err)
		}
		if got.Daily != nil {
			t.Logf("%s 最高 %s 最低 %s", got.Daily.Date, got.Daily.TempMax, got.Daily.TempMin)
		}
		t.Logf("%d 条逐小时", len(got.Hourly))
	})

	t.Run("Astronomy", func(t *testing.T) {
		date := time.Now().Format("20060102")
		got, err := as.Sun(ctx, loc, date)
		if err != nil {
			t.Fatalf("Astronomy: %v", err)
		}
		if got.Sunrise != nil && got.Sunset != nil {
			t.Logf("%s %s", *got.Sunrise, *got.Sunset)
		} else {
			t.Logf("%s 无日出日落", date)
		}
	})
}
