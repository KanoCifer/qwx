package astronomy_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7/astronomy"
)

func TestSun(t *testing.T) {
	body := `{
  "code": "200",
  "updateTime": "2026-05-31T02:00+08:00",
  "fxLink": "https://dev.qweather.com/docs/api/astronomy/sunrise-sunset/",
  "sunrise": "05:12",
  "sunset": "19:24",
  "refer": {
    "sources": ["https://developer.qweather.com/attribution.html"],
    "license": ["QWeather Developers License"]
  }
}`
	fs := testutil.NewFakeServer(t, body)
	c := astronomy.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	got, err := c.Sun(context.Background(), loc, "20260531")
	if err != nil {
		t.Fatalf("Sun: %v", err)
	}
	fs.AssertPath(t, "/v7/astronomy/sun")
	fs.AssertQuery(t, "location", "116.41,39.92")
	fs.AssertQuery(t, "date", "20260531")

	if got.Sunrise == nil || *got.Sunrise != "05:12" {
		t.Errorf("sunrise = %v, want 05:12", got.Sunrise)
	}
	if got.Sunset == nil || *got.Sunset != "19:24" {
		t.Errorf("sunset = %v, want 19:24", got.Sunset)
	}
	// 展示上游数据时要一并展示来源与许可声明。
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

// TestSunAcceptsLocationID 上游这个参数认 LocationID。
func TestSunAcceptsLocationID(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	c := astronomy.New(testutil.NewClient(t, fs))

	if _, err := c.Sun(context.Background(), qwx.Location{ID: "101010100"}, "20260531"); err != nil {
		t.Fatalf("Sun: %v", err)
	}
	fs.AssertQuery(t, "location", "101010100")
}

func TestMoon(t *testing.T) {
	body := `{
  "code": "200",
  "updateTime": "2026-05-31T02:00+08:00",
  "fxLink": "https://dev.qweather.com/docs/api/astronomy/moon-and-moon-phase/",
  "moonrise": "20:11",
  "moonset": "06:12",
  "moonPhase": [
    {"fxTime": "2026-05-31T20:00+08:00", "value": "1", "name": "新月", "illumination": "0.5", "icon": "01"}
  ],
  "refer": {
    "sources": ["https://developer.qweather.com/attribution.html"],
    "license": ["QWeather Developers License"]
  }
}`
	fs := testutil.NewFakeServer(t, body)
	c := astronomy.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	got, err := c.Moon(context.Background(), loc, "20260531", astronomy.Options{Lang: "zh"})
	if err != nil {
		t.Fatalf("Moon: %v", err)
	}
	fs.AssertPath(t, "/v7/astronomy/moon")
	fs.AssertQuery(t, "location", "116.41,39.92")
	fs.AssertQuery(t, "date", "20260531")
	fs.AssertQuery(t, "lang", "zh")

	if got.Moonrise == nil || *got.Moonrise != "20:11" {
		t.Errorf("moonrise = %v, want 20:11", got.Moonrise)
	}
	if len(got.MoonPhase) != 1 || got.MoonPhase[0].Name != "新月" {
		t.Errorf("moonPhase = %+v, want 1 entry named 新月", got.MoonPhase)
	}
}

// TestMoonWithoutLang 月相端点的 lang 可选，留空用上游默认。
func TestMoonWithoutLang(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	c := astronomy.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Moon(context.Background(), loc, "20260531", astronomy.Options{}); err != nil {
		t.Fatalf("Moon: %v", err)
	}
	if got := fs.LastQuery.Get("lang"); got != "" {
		t.Errorf("lang = %q, want empty", got)
	}
}

func TestSolarElevationAngle(t *testing.T) {
	body := `{
  "code": "200",
  "solarElevationAngle": "35.42",
  "solarAzimuthAngle": "196.31",
  "solarHour": "1102",
  "hourAngle": "-12.55",
  "refer": {
    "sources": ["https://developer.qweather.com/attribution.html"],
    "license": ["QWeather Developers License"]
  }
}`
	fs := testutil.NewFakeServer(t, body)
	c := astronomy.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	got, err := c.SolarElevationAngle(context.Background(), loc, "20260531", "1200", "0800", 43)
	if err != nil {
		t.Fatalf("SolarElevationAngle: %v", err)
	}
	fs.AssertPath(t, "/v7/astronomy/solar-elevation-angle")
	fs.AssertQuery(t, "location", "116.41,39.92")
	fs.AssertQuery(t, "date", "20260531")
	fs.AssertQuery(t, "time", "1200")
	fs.AssertQuery(t, "tz", "0800")
	fs.AssertQuery(t, "alt", "43")

	if got.SolarElevationAngle != "35.42" {
		t.Errorf("solarElevationAngle = %q, want 35.42", got.SolarElevationAngle)
	}
	if got.SolarHour != "1102" {
		t.Errorf("solarHour = %q, want 1102", got.SolarHour)
	}
}

func TestRejectsBadDate(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := astronomy.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	// 客户端只校验格式：8 位数字且是真实日期，日期范围交给上游。
	for _, date := range []string{"", "2026-05-31", "2026053", "202605311", "20260230", "20261345"} {
		if _, err := c.Sun(context.Background(), loc, date); !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("Sun date=%q: err = %v, want ErrInvalidParameter", date, err)
		}
		if _, err := c.Moon(context.Background(), loc, date, astronomy.Options{}); !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("Moon date=%q: err = %v, want ErrInvalidParameter", date, err)
		}
		if _, err := c.SolarElevationAngle(context.Background(), loc, date, "1200", "0800", 43); !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("SolarElevationAngle date=%q: err = %v, want ErrInvalidParameter", date, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestSolarElevationAngleRejectsBadTime(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := astronomy.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	// HHmm 24 时制：0000~2359。
	for _, timeStr := range []string{"", "120", "12000", "2400", "1260", "abcd"} {
		_, err := c.SolarElevationAngle(context.Background(), loc, "20260531", timeStr, "0800", 43)
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("time=%q: err = %v, want ErrInvalidParameter", timeStr, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestSolarElevationAngleRejectsBadTimezone(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := astronomy.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	// HHmm 时区偏移，正负号可省，取值范围 [-12:00,+14:00]。
	for _, tz := range []string{"", "080", "08000", "0800x", "1500", "-1400", "0890"} {
		_, err := c.SolarElevationAngle(context.Background(), loc, "20260531", "1200", tz, 43)
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("tz=%q: err = %v, want ErrInvalidParameter", tz, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestSolarElevationAngleAcceptsNegativeTimezone(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	c := astronomy.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.SolarElevationAngle(context.Background(), loc, "20260531", "1200", "-0530", 43); err != nil {
		t.Fatalf("SolarElevationAngle: %v", err)
	}
	fs.AssertQuery(t, "tz", "-0530")
}

func TestSolarElevationAngleRejectsNegativeAlt(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := astronomy.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	_, err := c.SolarElevationAngle(context.Background(), loc, "20260531", "1200", "0800", -1)
	if !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Fatalf("err = %v, want ErrInvalidParameter", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestRejectsUnsupportedLocationForm(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := astronomy.New(testutil.NewClient(t, fs))

	// 上游不认地名：只有地名的位置没有可用的 location。
	for _, loc := range []qwx.Location{{Name: "北京"}, {}} {
		if _, err := c.Sun(context.Background(), loc, "20260531"); !errors.Is(err, qwx.ErrInvalidLocation) {
			t.Errorf("Sun loc=%+v: err = %v, want ErrInvalidLocation", loc, err)
		}
		if _, err := c.Moon(context.Background(), loc, "20260531", astronomy.Options{}); !errors.Is(err, qwx.ErrInvalidLocation) {
			t.Errorf("Moon loc=%+v: err = %v, want ErrInvalidLocation", loc, err)
		}
	}
	// 这个端点用 lonlatQuery，只认坐标对，LocationID 会被拒绝。
	if _, err := c.SolarElevationAngle(context.Background(), qwx.Location{ID: "101010100"}, "20260531", "1200", "0800", 43); !errors.Is(err, qwx.ErrInvalidLocation) {
		t.Errorf("SolarElevationAngle: err = %v, want ErrInvalidLocation", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestEndpointsAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := astronomy.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Sun(context.Background(), loc, "20260531"); err != nil {
		t.Fatalf("Sun: %v", err)
	}
	req := d.Last(t)
	if req.Endpoint != astronomy.EndpointSun {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, astronomy.EndpointSun)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}

	if _, err := c.Moon(context.Background(), loc, "20260531", astronomy.Options{Lang: "zh"}); err != nil {
		t.Fatalf("Moon: %v", err)
	}
	req = d.Last(t)
	if req.Endpoint != astronomy.EndpointMoon {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, astronomy.EndpointMoon)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}

	if _, err := c.SolarElevationAngle(context.Background(), loc, "20260531", "1200", "0800", 43); err != nil {
		t.Fatalf("SolarElevationAngle: %v", err)
	}
	req = d.Last(t)
	if req.Endpoint != astronomy.EndpointAngle {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, astronomy.EndpointAngle)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
//
// 改了会让已写入的缓存条目全部失效。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := astronomy.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Sun(context.Background(), loc, "20260531"); err != nil {
		t.Fatalf("Sun: %v", err)
	}
	if want := "qwx:astronomy:sun:116.41,39.92:20260531"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Moon(context.Background(), loc, "20260531", astronomy.Options{Lang: "zh"}); err != nil {
		t.Fatalf("Moon: %v", err)
	}
	if want := "qwx:astronomy:moon:116.41,39.92:20260531:lang=zh"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.SolarElevationAngle(context.Background(), loc, "20260531", "1200", "0800", 43); err != nil {
		t.Fatalf("SolarElevationAngle: %v", err)
	}
	if want := "qwx:astronomy:angle:116.41,39.92:20260531:1200:0800:43"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}

// TestCacheKeysDistinguishResultAffectingParams 影响结果的参数各产生不同的键。
//
// 共键就是串味：不同日期、时间、时区、海拔或语言的答案是不同的数据。
func TestCacheKeysDistinguishResultAffectingParams(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := astronomy.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	keys := map[string]bool{}
	add := func(label string) {
		t.Helper()
		key := d.Last(t).CacheKey
		if keys[key] {
			t.Errorf("%s: cacheKey %q is shared with another request", label, key)
		}
		keys[key] = true
	}

	for _, date := range []string{"20260531", "20260601"} {
		if _, err := c.Sun(context.Background(), loc, date); err != nil {
			t.Fatalf("Sun: %v", err)
		}
		add("Sun date=" + date)
	}
	for _, lang := range []string{"", "zh", "en"} {
		if _, err := c.Moon(context.Background(), loc, "20260531", astronomy.Options{Lang: lang}); err != nil {
			t.Fatalf("Moon: %v", err)
		}
		add("Moon lang=" + lang)
	}
	// 每组换一个基准时间，各组的键才不会撞在一起。
	for _, timeStr := range []string{"1200", "1800"} {
		if _, err := c.SolarElevationAngle(context.Background(), loc, "20260531", timeStr, "0800", 43); err != nil {
			t.Fatalf("SolarElevationAngle: %v", err)
		}
		add("Angle time=" + timeStr)
	}
	for _, tz := range []string{"0800", "-0530"} {
		if _, err := c.SolarElevationAngle(context.Background(), loc, "20260531", "0900", tz, 43); err != nil {
			t.Fatalf("SolarElevationAngle: %v", err)
		}
		add("Angle tz=" + tz)
	}
	for _, alt := range []int{0, 43} {
		if _, err := c.SolarElevationAngle(context.Background(), loc, "20260531", "2100", "0800", alt); err != nil {
			t.Fatalf("SolarElevationAngle: %v", err)
		}
		add("Angle alt=")
	}
}

// TestCacheWriteUsesEndpointTTL 缓存写入用端点内置的默认时长。
func TestCacheWriteUsesEndpointTTL(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	cache := testutil.NewRecordingCache()
	c := astronomy.New(testutil.NewClient(t, fs, qwx.WithCache(cache)))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Sun(context.Background(), loc, "20260531"); err != nil {
		t.Fatalf("Sun: %v", err)
	}
	ttl, ok := cache.TTL("qwx:astronomy:sun:116.41,39.92:20260531")
	if !ok {
		t.Fatal("sun result was not cached")
	}
	if ttl != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", ttl)
	}
}

// TestTTLPolicyOverridesEndpointDefault 端点常量是策略的匹配键。
func TestTTLPolicyOverridesEndpointDefault(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"code":"200"}`)
	cache := testutil.NewRecordingCache()
	c := astronomy.New(testutil.NewClient(t, fs,
		qwx.WithCache(cache),
		qwx.WithTTLPolicy(qwx.TTLMap{astronomy.EndpointSun: time.Hour}),
	))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Sun(context.Background(), loc, "20260531"); err != nil {
		t.Fatalf("Sun: %v", err)
	}
	ttl, ok := cache.TTL("qwx:astronomy:sun:116.41,39.92:20260531")
	if !ok {
		t.Fatal("sun result was not cached")
	}
	if ttl != time.Hour {
		t.Errorf("ttl = %v, want 1h", ttl)
	}
}
