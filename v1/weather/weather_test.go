package weather_test

import (
	"context"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v1/weather"
)

func TestCurrentWeather(t *testing.T) {
	body := `{
	  "metadata": {"tag": "abc", "attributions": ["https://developer.qweather.com/attribution.html"]},
	  "condition": {"text": "少云", "code": "102"},
	  "temperature": {"value": 31.71, "unit": "°C"},
	  "feelsLike": {"value": 33.64, "unit": "°C"},
	  "humidity": 0.69,
	  "wind": {"direction": {"degree": 226, "compass": "sw"},
	           "speed": {"value": 4.74, "unit": "m/s"}, "scale": 3},
	  "windGust": {"value": 7.07, "unit": "m/s"},
	  "precipitation": {"amount": {"value": 0, "unit": "mm"},
	                    "intensity": {"value": 0, "unit": "mm/h"}, "type": "none"},
	  "pressure": {"value": 1001.5, "unit": "hPa"},
	  "visibility": {"value": 29020, "unit": "m"},
	  "dewPoint": {"value": 25.36, "unit": "°C"},
	  "cloudCover": 0.05,
	  "uvIndex": 3
	}`
	fs := testutil.NewFakeServer(t, body)
	c := weather.New(testutil.NewClient(t, fs))

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	got, err := c.Current(context.Background(), loc, weather.Options{Lang: "zh", LocalTime: true})
	if err != nil {
		t.Fatalf("Current: %v", err)
	}

	// 路径是纬度在前、经度在后。
	fs.AssertPath(t, "/weather/v1/current/39.92/116.41")
	fs.AssertQuery(t, "lang", "zh")
	fs.AssertQuery(t, "localTime", "true")

	if got.Condition.Code != "102" {
		t.Errorf("condition code = %q, want 102", got.Condition.Code)
	}
	if got.Temperature.Value != 31.71 {
		t.Errorf("temperature = %v, want 31.71", got.Temperature.Value)
	}
	if got.Temperature.Unit != "°C" {
		t.Errorf("temperature unit = %q, want °C", got.Temperature.Unit)
	}
	// 湿度与云量是 0~1 的比例，不是百分数。
	if got.Humidity != 0.69 {
		t.Errorf("humidity = %v, want 0.69", got.Humidity)
	}
	if got.CloudCover != 0.05 {
		t.Errorf("cloudCover = %v, want 0.05", got.CloudCover)
	}
	if got.Wind.Direction.Compass != "sw" {
		t.Errorf("wind compass = %q, want sw", got.Wind.Direction.Compass)
	}
	if got.Metadata.Tag != "abc" {
		t.Errorf("metadata tag = %q, want abc", got.Metadata.Tag)
	}
	// 实时天气没有降水概率字段，应为 nil 而非 0。
	if got.Precipitation.Probability != nil {
		t.Errorf("probability = %v, want nil", *got.Precipitation.Probability)
	}
}

func TestCurrentRejectsLocationID(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := weather.New(testutil.NewClient(t, fs))

	// LocationID 没有坐标，新一代接口只认坐标路径参数。
	_, err := c.Current(context.Background(), qwx.Location{ID: "101010100"}, weather.Options{})
	if err == nil {
		t.Fatal("expected error for LocationID without coordinates")
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0 (must not reach upstream)", fs.Hits)
	}
}

func TestHourlyAndDailyParams(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"metadata":{"tag":"t"},"hours":[]}`)
	c := weather.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Hourly(context.Background(), loc, 24, weather.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}
	fs.AssertPath(t, "/weather/v1/hourly/39.92/116.41")
	fs.AssertQuery(t, "hours", "24")

	if _, err := c.Daily(context.Background(), loc, 7, weather.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}
	fs.AssertPath(t, "/weather/v1/daily/39.92/116.41")
	fs.AssertQuery(t, "days", "7")
}

func TestHourlyRejectsOutOfRange(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := weather.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	for _, hours := range []int{0, -1, weather.MaxHourlyHours + 1} {
		if _, err := c.Hourly(context.Background(), loc, hours, weather.Options{}); err == nil {
			t.Errorf("hours=%d: expected error", hours)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestDailyRejectsOutOfRange(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := weather.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Daily(context.Background(), loc, weather.MaxDailyDays+1, weather.Options{}); err == nil {
		t.Error("expected error")
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestCacheKeyIncludesParams(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"metadata":{"tag":"t"}}`)
	c := weather.New(testutil.NewClient(t, fs, qwx.WithCache(qwx.NewMemoryCache())))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Current(context.Background(), loc, weather.Options{Lang: "zh"}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	// 语言不同是不同的数据，不能共用缓存。
	if _, err := c.Current(context.Background(), loc, weather.Options{Lang: "en"}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	if fs.Hits != 2 {
		t.Errorf("hits = %d, want 2 (different lang must not share cache)", fs.Hits)
	}
}
