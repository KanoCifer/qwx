//go:build live

// 本文件是打真实上游的冒烟测试，默认不参与 `go test ./...`。
//
// 用法：
//
//	QWEATHER_API_HOST=https://your-host.qweatherapi.com \
//	QWEATHER_JWT_PRIVATE_KEY="$(cat ed25519-private.pem)" \
//	QWEATHER_KEY_ID=YOUR_KEY_ID \
//	QWEATHER_PROJECT_ID=YOUR_PROJECT_ID \
//	go test -tags live -v -run TestLive ./...
package live

import (
	"context"
	"os"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/geo"
	"github.com/KanoCifer/qwx/v1/airquality"
	"github.com/KanoCifer/qwx/v1/alert"
	"github.com/KanoCifer/qwx/v1/weather"
	"github.com/KanoCifer/qwx/v7"
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
	old := v7.New(c)
	g := geo.New(c)

	t.Run("Current", func(t *testing.T) {
		got, err := w.Current(ctx, loc, weather.Options{Lang: "zh", LocalTime: true})
		if err != nil {
			t.Fatalf("Current: %v", err)
		}
		t.Logf("%s %v%s", got.Condition.Text, got.Temperature.Value, got.Temperature.Unit)
	})

	t.Run("Hourly", func(t *testing.T) {
		got, err := w.Hourly(ctx, loc, 24, weather.Options{Lang: "zh", LocalTime: true})
		if err != nil {
			t.Fatalf("Hourly: %v", err)
		}
		t.Logf("%d 条", len(got.Hours))
	})

	t.Run("Daily", func(t *testing.T) {
		got, err := w.Daily(ctx, loc, 3, weather.Options{Lang: "zh", LocalTime: true})
		if err != nil {
			t.Fatalf("Daily: %v", err)
		}
		t.Logf("%d 条", len(got.Days))
	})

	t.Run("LookupCity", func(t *testing.T) {
		got, err := g.LookupCity(ctx, qwx.Location{ID: "101010100"}, geo.Options{Lang: "zh"})
		if err != nil {
			t.Fatalf("LookupCity: %v", err)
		}
		for _, l := range got.Locations {
			t.Logf("%s %s", l.ID, l.DisplayName())
		}
	})

	t.Run("Indices", func(t *testing.T) {
		got, err := old.Indices(ctx, loc, v7.IndicesTypeFishing, 1, "zh")
		if err != nil {
			t.Fatalf("Indices: %v", err)
		}
		for _, d := range got.Daily {
			t.Logf("%s %s %s", d.Name, d.Category, d.Level)
		}
	})

	t.Run("Minutely", func(t *testing.T) {
		got, err := old.Minutely(ctx, loc, "zh")
		if err != nil {
			t.Fatalf("Minutely: %v", err)
		}
		t.Logf("%d 条 %s", len(got.Minutely), got.Summary)
	})

	t.Run("AirQuality", func(t *testing.T) {
		got, err := aq.Current(ctx, loc, airquality.Options{Lang: "zh"})
		if err != nil {
			t.Fatalf("AirQuality: %v", err)
		}
		for _, i := range got.Indexes {
			t.Logf("%s aqi=%v %s", i.Code, i.AQI, i.Category)
		}
	})

	t.Run("Alerts", func(t *testing.T) {
		got, err := al.Current(ctx, loc, alert.Options{Lang: "zh", LocalTime: true})
		if err != nil {
			t.Fatalf("Alerts: %v", err)
		}
		t.Logf("%d 条", len(got.Alerts))
	})
}
