package solarradiation_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v1/solarradiation"
)

func TestForecast(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	tilt, azimuth := 30, 180
	got, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{
		LocalTime: true,
		Interval:  15,
		Tilt:      &tilt,
		Azimuth:   &azimuth,
		Extra:     "weather,poa",
	})
	if err != nil {
		t.Fatalf("Forecast: %v", err)
	}

	// 路径是纬度在前、经度在后。
	fs.AssertPath(t, "/solarradiation/v1/forecast/39.92/116.41")
	fs.AssertQuery(t, "hours", "24")
	fs.AssertQuery(t, "interval", "15")
	fs.AssertQuery(t, "tilt", "30")
	fs.AssertQuery(t, "azimuth", "180")
	fs.AssertQuery(t, "extra", "weather,poa")
	fs.AssertQuery(t, "localTime", "true")

	if len(got.Forecasts) != 2 {
		t.Fatalf("forecasts = %d, want 2", len(got.Forecasts))
	}
	if got.Forecasts[0].GHI.Value != 152.57 {
		t.Errorf("ghi = %v, want 152.57", got.Forecasts[0].GHI.Value)
	}
}

func TestForecastOmitsUnsetOptions(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Forecast(context.Background(), loc, 12, solarradiation.Options{}); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	fs.AssertQuery(t, "hours", "12")
	for _, key := range []string{"interval", "tilt", "azimuth", "extra", "localTime"} {
		if v := fs.LastQuery.Get(key); v != "" {
			t.Errorf("query %q = %q, want unset", key, v)
		}
	}
}

// TestForecastSendsZeroTiltAndAzimuth 0 是合法取值，不是「没传」。
func TestForecastSendsZeroTiltAndAzimuth(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	tilt, azimuth := 0, 0
	if _, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{
		Tilt:    &tilt,
		Azimuth: &azimuth,
		Extra:   "poa",
	}); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	fs.AssertQuery(t, "tilt", "0")
	fs.AssertQuery(t, "azimuth", "0")
}

func TestForecastRejectsInvalidLocation(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))

	// LocationID 没有坐标，新一代接口只认坐标路径参数。
	_, err := c.Forecast(context.Background(), qwx.Location{ID: "101010100"}, 24, solarradiation.Options{})
	if err == nil {
		t.Fatal("expected error for LocationID without coordinates")
	}
	if !errors.Is(err, qwx.ErrInvalidLocation) {
		t.Errorf("err = %v, want ErrInvalidLocation", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0 (must not reach upstream)", fs.Hits)
	}
}

func TestForecastRejectsHoursOutOfRange(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	for _, hours := range []int{0, -1, solarradiation.MaxForecastHours + 1} {
		_, err := c.Forecast(context.Background(), loc, hours, solarradiation.Options{})
		if err == nil {
			t.Errorf("hours=%d: expected error", hours)
			continue
		}
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("hours=%d: err = %v, want ErrInvalidParameter", hours, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0 (must not reach upstream)", fs.Hits)
	}
}

func TestForecastRejectsInvalidInterval(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	for _, interval := range []int{5, 45, 90} {
		_, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{Interval: interval})
		if err == nil {
			t.Errorf("interval=%d: expected error", interval)
			continue
		}
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("interval=%d: err = %v, want ErrInvalidParameter", interval, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0 (must not reach upstream)", fs.Hits)
	}
}

func TestForecastRejectsInvalidTilt(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	for _, tilt := range []int{-1, 91} {
		v := tilt
		_, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{Tilt: &v})
		if err == nil {
			t.Errorf("tilt=%d: expected error", tilt)
			continue
		}
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("tilt=%d: err = %v, want ErrInvalidParameter", tilt, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0 (must not reach upstream)", fs.Hits)
	}
}

func TestForecastRejectsInvalidAzimuth(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	for _, azimuth := range []int{-1, 360} {
		v := azimuth
		_, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{Azimuth: &v})
		if err == nil {
			t.Errorf("azimuth=%d: expected error", azimuth)
			continue
		}
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("azimuth=%d: err = %v, want ErrInvalidParameter", azimuth, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0 (must not reach upstream)", fs.Hits)
	}
}

func TestForecastRejectsInvalidExtra(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	for _, extra := range []string{"wind", "weather,wind", "weather,weather", "Weather"} {
		_, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{Extra: extra})
		if err == nil {
			t.Errorf("extra=%q: expected error", extra)
			continue
		}
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("extra=%q: err = %v, want ErrInvalidParameter", extra, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0 (must not reach upstream)", fs.Hits)
	}
}

// TestForecastRequiresTiltAndAzimuthForPOA extra 含 poa 时上游要求两个都传。
func TestForecastRequiresTiltAndAzimuthForPOA(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	tilt, azimuth := 30, 180
	cases := []struct {
		name string
		opts solarradiation.Options
	}{
		{"neither", solarradiation.Options{Extra: "poa"}},
		{"only tilt", solarradiation.Options{Extra: "poa", Tilt: &tilt}},
		{"only azimuth", solarradiation.Options{Extra: "poa", Azimuth: &azimuth}},
		{"weather,poa without tilt", solarradiation.Options{Extra: "weather,poa", Azimuth: &azimuth}},
	}
	for _, tc := range cases {
		_, err := c.Forecast(context.Background(), loc, 24, tc.opts)
		if err == nil {
			t.Errorf("%s: expected error", tc.name)
			continue
		}
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("%s: err = %v, want ErrInvalidParameter", tc.name, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0 (must not reach upstream)", fs.Hits)
	}
}

func TestForecastReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := solarradiation.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{}); err != nil {
		t.Fatalf("Forecast: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != solarradiation.Endpoint {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, solarradiation.Endpoint)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

// TestTTLPolicyOverridesDefault 策略按端点标识覆盖默认时长。
func TestTTLPolicyOverridesDefault(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	rec := testutil.NewRecordingCache()
	base := testutil.NewClient(t, fs,
		qwx.WithCache(rec),
		qwx.WithTTLPolicy(qwx.TTLMap{solarradiation.Endpoint: 5 * time.Minute}),
	)
	c := solarradiation.New(base)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{}); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	key := "qwx:solarradiation:v1:forecast:39.92,116.41:h24"
	ttl, ok := rec.TTL(key)
	if !ok {
		t.Fatalf("no cache entry for %q", key)
	}
	if ttl != 5*time.Minute {
		t.Errorf("ttl = %v, want 5m", ttl)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
//
// 改了会让已写入的缓存条目全部失效。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := solarradiation.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{}); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	if want := "qwx:solarradiation:v1:forecast:39.92,116.41:h24"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	tilt, azimuth := 30, 180
	if _, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{
		LocalTime: true,
		Interval:  15,
		Tilt:      &tilt,
		Azimuth:   &azimuth,
		Extra:     "weather,poa",
	}); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	want := "qwx:solarradiation:v1:forecast:39.92,116.41:h24" +
		":localTime=1:interval=15:tilt=30:azimuth=180:extra=weather,poa"
	if d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}

// TestCacheKeySeparatesAffectingParams 影响结果的参数必须进键。
func TestCacheKeySeparatesAffectingParams(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := solarradiation.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	tilt30, tilt45 := 30, 45
	azimuth180, azimuth90 := 180, 90
	// 每组相对基准只改一个参数，键必须两两不同。
	cases := []struct {
		name  string
		hours int
		opts  solarradiation.Options
	}{
		{"base", 24, solarradiation.Options{Interval: 15, Tilt: &tilt30, Azimuth: &azimuth180, Extra: "poa"}},
		{"hours", 48, solarradiation.Options{Interval: 15, Tilt: &tilt30, Azimuth: &azimuth180, Extra: "poa"}},
		{"interval", 24, solarradiation.Options{Interval: 30, Tilt: &tilt30, Azimuth: &azimuth180, Extra: "poa"}},
		{"tilt", 24, solarradiation.Options{Interval: 15, Tilt: &tilt45, Azimuth: &azimuth180, Extra: "poa"}},
		{"azimuth", 24, solarradiation.Options{Interval: 15, Tilt: &tilt30, Azimuth: &azimuth90, Extra: "poa"}},
		{"extra", 24, solarradiation.Options{Interval: 15, Tilt: &tilt30, Azimuth: &azimuth180, Extra: "weather"}},
	}
	keys := map[string]bool{}
	for _, tc := range cases {
		if _, err := c.Forecast(context.Background(), loc, tc.hours, tc.opts); err != nil {
			t.Fatalf("Forecast: %v", err)
		}
		keys[d.Last(t).CacheKey] = true
	}
	if len(keys) != len(cases) {
		t.Errorf("keys = %v, want %d 个不同的键", keys, len(cases))
	}
}

// TestForecastCacheSeparatesAffectingParams 影响结果的参数不同不能共用缓存。
func TestForecastCacheSeparatesAffectingParams(t *testing.T) {
	fs := testutil.NewFakeServer(t, upstreamBody)
	c := solarradiation.New(testutil.NewClient(t, fs, qwx.WithCache(qwx.NewMemoryCache())))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{}); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	if _, err := c.Forecast(context.Background(), loc, 24, solarradiation.Options{}); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	if fs.Hits != 1 {
		t.Errorf("hits = %d, want 1 (same key must hit cache)", fs.Hits)
	}

	if _, err := c.Forecast(context.Background(), loc, 48, solarradiation.Options{}); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	if fs.Hits != 2 {
		t.Errorf("hits = %d, want 2 (different hours must not share cache)", fs.Hits)
	}
}
