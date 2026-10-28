package airquality_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v1/airquality"
)

func TestCurrent(t *testing.T) {
	body := `{
	  "metadata": {"tag": "t"},
	  "indexes": [{
	    "code": "us-epa", "name": "AQI (US)", "aqi": 46, "aqiDisplay": "46",
	    "level": "1", "category": "Good",
	    "color": {"red": 0, "green": 228, "blue": 0, "alpha": 1},
	    "primaryPollutant": {"code": "pm2p5", "name": "PM 2.5"},
	    "health": {"effect": "No health effects.",
	               "advice": {"generalPopulation": "正常", "sensitivePopulation": "正常"}}
	  }],
	  "pollutants": [{
	    "code": "pm2p5", "name": "PM 2.5", "fullName": "Fine particulate matter",
	    "concentration": {"value": 11.0, "unit": "μg/m3"},
	    "subIndexes": [{"code": "us-epa", "aqi": 46, "aqiDisplay": "46"}]
	  }],
	  "stations": [{"id": "P51762", "name": "North Holywood"}]
	}`
	fs := testutil.NewFakeServer(t, body)
	c := airquality.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 54.99, Lon: -1.61}

	got, err := c.Current(context.Background(), loc, airquality.Options{Lang: qwx.LangZH})
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	fs.AssertPath(t, "/airquality/v1/current/54.99/-1.61")
	fs.AssertQuery(t, "lang", "zh")

	if len(got.Indexes) != 1 || got.Indexes[0].AQI != 46 {
		t.Errorf("indexes = %+v", got.Indexes)
	}
	if got.Indexes[0].PrimaryPollutant == nil {
		t.Error("primaryPollutant should not be nil")
	}
	if len(got.Pollutants) != 1 || got.Pollutants[0].Concentration.Value != 11.0 {
		t.Errorf("pollutants = %+v", got.Pollutants)
	}
	if len(got.Stations) != 1 || got.Stations[0].ID != "P51762" {
		t.Errorf("stations = %+v", got.Stations)
	}
}

func TestStation(t *testing.T) {
	body := `{
	  "metadata": {"tag": "t"},
	  "pollutants": [{
	    "code": "pm2p5", "name": "PM 2.5",
	    "concentration": {"value": 9.5, "unit": "μg/m3"},
	    "subIndexes": []
	  }]
	}`
	fs := testutil.NewFakeServer(t, body)
	c := airquality.New(testutil.NewClient(t, fs))

	got, err := c.Station(context.Background(), "P51762", airquality.Options{})
	if err != nil {
		t.Fatalf("Station: %v", err)
	}
	fs.AssertPath(t, "/airquality/v1/stations/P51762")

	if len(got.Pollutants) != 1 || got.Pollutants[0].Concentration.Value != 9.5 {
		t.Errorf("pollutants = %+v", got.Pollutants)
	}
}

func TestStationRejectsEmptyID(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := airquality.New(testutil.NewClient(t, fs))

	if _, err := c.Station(context.Background(), "", airquality.Options{}); err == nil {
		t.Error("expected error")
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestDailyAndHourly(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"metadata":{"tag":"t"},"days":[],"hours":[]}`)
	c := airquality.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Daily(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}
	fs.AssertPath(t, "/airquality/v1/daily/39.92/116.41")

	if _, err := c.Hourly(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}
	fs.AssertPath(t, "/airquality/v1/hourly/39.92/116.41")
}

func TestCurrentReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Current(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Current: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != airquality.EndpointCurrent {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, airquality.EndpointCurrent)
	}
	if req.TTL != 30*time.Minute {
		t.Errorf("ttl = %v, want 30m", req.TTL)
	}
}

func TestDailyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Daily(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != airquality.EndpointDaily {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, airquality.EndpointDaily)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

func TestHourlyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Hourly(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != airquality.EndpointHourly {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, airquality.EndpointHourly)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

func TestStationReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)

	if _, err := c.Station(context.Background(), "P51762", airquality.Options{}); err != nil {
		t.Fatalf("Station: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != airquality.EndpointStation {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, airquality.EndpointStation)
	}
	if req.TTL != 30*time.Minute {
		t.Errorf("ttl = %v, want 30m", req.TTL)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
//
// 改了会让已写入的缓存条目全部失效。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := airquality.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Current(context.Background(), loc, airquality.Options{Lang: qwx.LangZH}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	if want := "qwx:aq:v1:current:39.92,116.41:lang=zh"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Daily(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}
	if want := "qwx:aq:v1:daily:39.92,116.41"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Hourly(context.Background(), loc, airquality.Options{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}
	if want := "qwx:aq:v1:hourly:39.92,116.41"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Station(context.Background(), "P51762", airquality.Options{Lang: qwx.LangZH}); err != nil {
		t.Fatalf("Station: %v", err)
	}
	if want := "qwx:aq:v1:station:P51762:lang=zh"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}
