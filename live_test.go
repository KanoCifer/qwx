//go:build live

// 本文件是打真实上游的冒烟测试，默认不参与 `go test ./...`。
//
// 用法：
//
//	QWEATHER_BASE_URL=https://your-host.qweatherapi.com \
//	QWEATHER_JWT_PRIVATE_KEY="$(cat ed25519-private.pem)" \
//	QWEATHER_KEY_ID=YOUR_KEY_ID \
//	QWEATHER_PROJECT_ID=YOUR_PROJECT_ID \
//	go test -tags live -v -run TestLive ./...
package qwx

import (
	"context"
	"os"
	"testing"
)

func liveClient(t *testing.T) *Client {
	t.Helper()
	base := os.Getenv("QWEATHER_BASE_URL")
	priv := os.Getenv("QWEATHER_JWT_PRIVATE_KEY")
	if base == "" || priv == "" {
		t.Skip("未设置 QWEATHER_BASE_URL / QWEATHER_JWT_PRIVATE_KEY，跳过")
	}
	auth, err := NewJWTAuth(JWTConfig{
		PrivateKeyPEM: priv,
		KeyID:         os.Getenv("QWEATHER_KEY_ID"),
		ProjectID:     os.Getenv("QWEATHER_PROJECT_ID"),
	})
	if err != nil {
		t.Fatalf("NewJWTAuth: %v", err)
	}
	c, err := NewClient(auth, WithBaseURL(base))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestLive(t *testing.T) {
	c := liveClient(t)
	ctx := context.Background()
	loc := Location{Lat: 39.92, Lon: 116.41}

	t.Run("Current", func(t *testing.T) {
		got, err := c.Current(ctx, loc, WeatherOptions{Lang: "zh", LocalTime: true})
		if err != nil {
			t.Fatalf("Current: %v", err)
		}
		t.Logf("%s %v%s", got.Condition.Text, got.Temperature.Value, got.Temperature.Unit)
	})

	t.Run("Hourly", func(t *testing.T) {
		got, err := c.Hourly(ctx, loc, 24, WeatherOptions{Lang: "zh", LocalTime: true})
		if err != nil {
			t.Fatalf("Hourly: %v", err)
		}
		t.Logf("%d 条", len(got.Hours))
	})

	t.Run("Daily", func(t *testing.T) {
		got, err := c.Daily(ctx, loc, 3, WeatherOptions{Lang: "zh", LocalTime: true})
		if err != nil {
			t.Fatalf("Daily: %v", err)
		}
		t.Logf("%d 条", len(got.Days))
	})

	t.Run("LookupCity", func(t *testing.T) {
		got, err := c.LookupCity(ctx, Location{ID: "101010100"}, GeoOptions{Lang: "zh"})
		if err != nil {
			t.Fatalf("LookupCity: %v", err)
		}
		for _, l := range got.Locations {
			t.Logf("%s %s", l.ID, l.DisplayName())
		}
	})

	t.Run("Indices", func(t *testing.T) {
		got, err := c.Indices(ctx, loc, IndicesTypeFishing, 1, "zh")
		if err != nil {
			t.Fatalf("Indices: %v", err)
		}
		for _, d := range got.Daily {
			t.Logf("%s %s %s", d.Name, d.Category, d.Level)
		}
	})

	t.Run("Minutely", func(t *testing.T) {
		got, err := c.Minutely(ctx, loc, "zh")
		if err != nil {
			t.Fatalf("Minutely: %v", err)
		}
		t.Logf("%d 条 %s", len(got.Minutely), got.Summary)
	})

	t.Run("AirQuality", func(t *testing.T) {
		got, err := c.AirQuality(ctx, loc, AirQualityOptions{Lang: "zh"})
		if err != nil {
			t.Fatalf("AirQuality: %v", err)
		}
		for _, i := range got.Indexes {
			t.Logf("%s aqi=%v %s", i.Code, i.AQI, i.Category)
		}
	})

	t.Run("Alerts", func(t *testing.T) {
		got, err := c.Alerts(ctx, loc, WeatherOptions{Lang: "zh", LocalTime: true})
		if err != nil {
			t.Fatalf("Alerts: %v", err)
		}
		t.Logf("%d 条", len(got.Alerts))
	})
}
