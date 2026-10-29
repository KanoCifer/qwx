package historical_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7/historical"
)

func TestWeather(t *testing.T) {
	body := `{
  "code": "200",
  "fxLink": "https://www.qweather.com",
  "weatherDaily": {
    "date": "2020-07-25",
    "sunrise": "05:08",
    "sunset": "19:33",
    "moonrise": "09:54",
    "moonset": "22:40",
    "moonPhase": "峨眉月",
    "tempMax": "33",
    "tempMin": "23",
    "humidity": "52",
    "precip": "0.0",
    "pressure": "1000"
  },
  "weatherHourly": [
    {"time": "2020-07-25 21:00", "temp": "29", "icon": "104", "text": "阴", "precip": "0.0", "wind360": "208", "windDir": "西南风", "windScale": "2", "windSpeed": "7", "humidity": "52", "pressure": "1000"},
    {"time": "2020-07-25 22:00", "temp": "29", "icon": "104", "text": "阴", "precip": "0.0", "wind360": "188", "windDir": "南风", "windScale": "2", "windSpeed": "7", "humidity": "57", "pressure": "1000"}
  ],
  "refer": {
    "sources": ["https://developer.qweather.com/attribution.html"],
    "license": ["QWeather Developers License"]
  }
}`
	fs := testutil.NewFakeServer(t, body)
	c := historical.New(testutil.NewClient(t, fs))

	got, err := c.Weather(context.Background(), qwx.Location{ID: "101010100"}, "20200725", historical.Options{Lang: qwx.LangZH})
	if err != nil {
		t.Fatalf("Weather: %v", err)
	}
	fs.AssertPath(t, "/v7/historical/weather")
	fs.AssertQuery(t, "location", "101010100")
	fs.AssertQuery(t, "date", "20200725")
	fs.AssertQuery(t, "lang", "zh")

	if got.Daily == nil {
		t.Fatal("daily = nil, want the daily summary")
	}
	if got.Daily.Date != "2020-07-25" {
		t.Errorf("daily.date = %q", got.Daily.Date)
	}
	// 高纬度地区可能没有日出，用指针区分"没有"与"空串"。
	if got.Daily.Sunrise == nil || *got.Daily.Sunrise != "05:08" {
		t.Errorf("daily.sunrise = %v", got.Daily.Sunrise)
	}
	// 数值保持字符串：小数位数是上游的表达精度。
	if got.Daily.Precip != "0.0" {
		t.Errorf("daily.precip = %q, want 0.0", got.Daily.Precip)
	}
	if len(got.Hourly) != 2 {
		t.Fatalf("hourly = %d, want 2", len(got.Hourly))
	}
	if got.Hourly[0].WindDir != "西南风" {
		t.Errorf("hourly[0].windDir = %q", got.Hourly[0].WindDir)
	}
	// 展示上游数据时要一并展示来源与许可声明。
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

// TestWeatherMissingOptionalTimes 上游可能不给日出日落与月升月落。
func TestWeatherMissingOptionalTimes(t *testing.T) {
	body := `{
  "code": "200",
  "weatherDaily": {"date": "2020-07-25", "sunrise": null, "sunset": null, "moonrise": null, "moonset": null, "moonPhase": "", "tempMax": "10", "tempMin": "2", "humidity": "60", "precip": "0.0", "pressure": "1001"}
}`
	fs := testutil.NewFakeServer(t, body)
	c := historical.New(testutil.NewClient(t, fs))

	got, err := c.Weather(context.Background(), qwx.Location{ID: "101010100"}, "20200725", historical.Options{})
	if err != nil {
		t.Fatalf("Weather: %v", err)
	}
	if got.Daily == nil {
		t.Fatal("daily = nil")
	}
	if got.Daily.Sunrise != nil || got.Daily.Sunset != nil {
		t.Errorf("sunrise/sunset = %v/%v, want nil", got.Daily.Sunrise, got.Daily.Sunset)
	}
	if got.Daily.Moonrise != nil || got.Daily.Moonset != nil {
		t.Errorf("moonrise/moonset = %v/%v, want nil", got.Daily.Moonrise, got.Daily.Moonset)
	}
}

// TestWeatherRequiresLocationID 上游只认 LocationID，坐标与地名都被拒。
func TestWeatherRequiresLocationID(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := historical.New(testutil.NewClient(t, fs))

	for _, loc := range []qwx.Location{
		{Lat: 39.92, Lon: 116.41},
		{Name: "北京"},
		{},
	} {
		_, err := c.Weather(context.Background(), loc, "20200725", historical.Options{})
		if !errors.Is(err, qwx.ErrInvalidLocation) {
			t.Fatalf("loc %+v: err = %v, want ErrInvalidLocation", loc, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestWeatherInvalidDate(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := historical.New(testutil.NewClient(t, fs))

	for _, date := range []string{"", "2020-07-25", "2020072", "202007255", "20201301", "20200230"} {
		_, err := c.Weather(context.Background(), qwx.Location{ID: "101010100"}, date, historical.Options{})
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Fatalf("date %q: err = %v, want ErrInvalidParameter", date, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

// TestInvalidLangRejects 非法语言代码在客户端就被挡下，不打上游。
func TestInvalidLangRejects(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := historical.New(d)

	_, err := c.Weather(context.Background(), qwx.Location{ID: "101010100"}, "20200725", historical.Options{Lang: qwx.Lang{API: "中文"}})
	if !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Fatalf("Weather error = %v, want ErrInvalidParameter", err)
	}
	if len(d.Requests) != 0 {
		t.Errorf("requests = %d, want 0 (invalid lang must not reach upstream)", len(d.Requests))
	}
}

func TestWeatherReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := historical.New(d)

	if _, err := c.Weather(context.Background(), qwx.Location{ID: "101010100"}, "20200725", historical.Options{}); err != nil {
		t.Fatalf("Weather: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != historical.Endpoint {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, historical.Endpoint)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
	// 缓存键字面量在已发布版本里定死，改了会让旧缓存条目全部失效。
	if want := "qwx:historical:weather:101010100:20200725"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}
}

// TestLanguageChangesCacheKey 换语言就是换一份数据，键必须跟着变。
func TestLanguageChangesCacheKey(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := historical.New(d)

	if _, err := c.Weather(context.Background(), qwx.Location{ID: "101010100"}, "20200725", historical.Options{Lang: qwx.LangEN}); err != nil {
		t.Fatalf("Weather: %v", err)
	}

	req := d.Last(t)
	if want := "qwx:historical:weather:101010100:20200725:lang=en"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}
}
