package weather_test

import (
	"context"
	"testing"
	"time"

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
	// 实时天气有降水强度。
	if got.Precipitation.Intensity == nil {
		t.Fatal("intensity = nil, want 0 mm/h")
	}
	if got.Precipitation.Intensity.Unit != "mm/h" {
		t.Errorf("intensity unit = %q, want mm/h", got.Precipitation.Intensity.Unit)
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

func TestDailyDayNightShape(t *testing.T) {
	// 上游 daily 端点的白天/夜间时段用 windGustMax，与 current、hourly
	// 端点的 windGust 不是同一个键。
	body := `{
	  "metadata": {"tag": "abc"},
	  "days": [{
	    "forecastStartTime": "2024-08-10T22:00Z",
	    "forecastEndTime": "2024-08-11T22:00Z",
	    "astro": {"sunrise": "2024-08-11T04:22Z", "sunset": "2024-08-11T19:34Z",
	               "moonPhase": "waning-gibbous"},
	    "temperatureMax": {"value": 29.94, "unit": "°C"},
	    "temperatureMin": {"value": 20.93, "unit": "°C"},
	    "temperatureAvg": {"value": 25.57, "unit": "°C"},
	    "uvIndexMax": 6,
	    "daytime": {
	      "forecastStartTime": "2024-08-11T05:00Z",
	      "forecastEndTime": "2024-08-11T17:00Z",
	      "condition": {"text": "小雨", "code": "305"},
	      "temperatureMax": {"value": 29.94, "unit": "°C"},
	      "temperatureMin": {"value": 20.93, "unit": "°C"},
	      "wind": {"direction": {"degree": 270, "compass": "w"},
	               "speed": {"value": 1.87, "unit": "m/s"}, "scale": 2},
	      "windGustMax": {"value": 6.64, "unit": "m/s"},
	      "precipitation": {"amount": {"value": 0.75, "unit": "mm"},
	                        "probability": 0.64, "type": "rain"},
	      "cloudCover": 0.32,
	      "humidity": 0.52
	    },
	    "nighttime": {
	      "forecastStartTime": "2024-08-11T17:00Z",
	      "forecastEndTime": "2024-08-12T05:00Z",
	      "condition": {"text": "晴间多云", "code": "103"},
	      "temperatureMax": {"value": 29.8, "unit": "°C"},
	      "temperatureMin": {"value": 19.95, "unit": "°C"},
	      "wind": {"direction": {"degree": 343, "compass": "nnw"},
	               "speed": {"value": 2.33, "unit": "m/s"}, "scale": 2},
	      "windGustMax": {"value": 7.16, "unit": "m/s"},
	      "precipitation": {"amount": {"value": 0, "unit": "mm"},
	                        "probability": 0, "type": "none"},
	      "cloudCover": 0.34,
	      "humidity": 0.56
	    }
	  }]
	}`
	fs := testutil.NewFakeServer(t, body)
	c := weather.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	got, err := c.Daily(context.Background(), loc, 1, weather.Options{})
	if err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if len(got.Days) != 1 {
		t.Fatalf("days = %d, want 1", len(got.Days))
	}
	day := got.Days[0]
	if day.Astro == nil || day.Astro.MoonPhase != "waning-gibbous" {
		t.Errorf("astro = %+v, want moonPhase waning-gibbous", day.Astro)
	}
	if day.TemperatureAvg.Value != 25.57 {
		t.Errorf("temperatureAvg = %v, want 25.57", day.TemperatureAvg.Value)
	}
	if day.UVIndexMax != 6 {
		t.Errorf("uvIndexMax = %v, want 6", day.UVIndexMax)
	}
	if day.Daytime.WindGustMax.Value != 6.64 {
		t.Errorf("daytime windGustMax = %v, want 6.64", day.Daytime.WindGustMax.Value)
	}
	if day.Nighttime.WindGustMax.Value != 7.16 {
		t.Errorf("nighttime windGustMax = %v, want 7.16", day.Nighttime.WindGustMax.Value)
	}
	if day.Daytime.Precipitation.Probability == nil || *day.Daytime.Precipitation.Probability != 0.64 {
		t.Errorf("daytime probability = %v, want 0.64", day.Daytime.Precipitation.Probability)
	}
	// 概率为 0 与没有该字段是两回事：nighttime 给了 0，不能是 nil。
	if day.Nighttime.Precipitation.Probability == nil {
		t.Fatal("nighttime probability = nil, want 0")
	}
	// 逐日预报的白天/夜间时段上游不给降水强度，不能读出零值强度。
	if day.Daytime.Precipitation.Intensity != nil {
		t.Errorf("daytime intensity = %+v, want nil", *day.Daytime.Precipitation.Intensity)
	}
	if day.Nighttime.Precipitation.Intensity != nil {
		t.Errorf("nighttime intensity = %+v, want nil", *day.Nighttime.Precipitation.Intensity)
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

func TestCurrentReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Current(context.Background(), loc, weather.Options{}); err != nil {
		t.Fatalf("Current: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != weather.EndpointCurrent {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, weather.EndpointCurrent)
	}
	if req.TTL != 10*time.Minute {
		t.Errorf("ttl = %v, want 10m", req.TTL)
	}
}

func TestHourlyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Hourly(context.Background(), loc, 24, weather.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != weather.EndpointHourly {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, weather.EndpointHourly)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

func TestDailyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Daily(context.Background(), loc, 3, weather.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != weather.EndpointDaily {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, weather.EndpointDaily)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
//
// 改了会让已写入的缓存条目全部失效。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Current(context.Background(), loc, weather.Options{Lang: "zh", LocalTime: true}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	if want := "qwx:v1:current:39.92,116.41:lang=zh:localTime=1"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	// 没有可选参数时键里不留后缀。
	if _, err := c.Current(context.Background(), loc, weather.Options{}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	if want := "qwx:v1:current:39.92,116.41"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Hourly(context.Background(), loc, 24, weather.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}
	if want := "qwx:v1:hourly:39.92,116.41:h24"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Daily(context.Background(), loc, 7, weather.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if want := "qwx:v1:daily:39.92,116.41:d7"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}

// TestCacheKeySeparatesAffectingParams 影响结果的参数必须进键。
func TestCacheKeySeparatesAffectingParams(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := weather.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	keys := map[string]bool{}
	for _, opts := range []weather.Options{{Lang: "zh"}, {Lang: "en"}, {Lang: "zh", LocalTime: true}} {
		if _, err := c.Current(context.Background(), loc, opts); err != nil {
			t.Fatalf("Current: %v", err)
		}
		keys[d.Last(t).CacheKey] = true
	}
	if len(keys) != 3 {
		t.Errorf("keys = %v, want 3 个不同的键", keys)
	}

	hourly := map[string]bool{}
	for _, hours := range []int{24, 48} {
		if _, err := c.Hourly(context.Background(), loc, hours, weather.Options{}); err != nil {
			t.Fatalf("Hourly: %v", err)
		}
		hourly[d.Last(t).CacheKey] = true
	}
	if len(hourly) != 2 {
		t.Errorf("keys = %v, want 2 个不同的键", hourly)
	}
}
