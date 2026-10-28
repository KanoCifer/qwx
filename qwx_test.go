package qwx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fakeServer 是一个记录请求的假上游。
type fakeServer struct {
	*httptest.Server
	// lastReq 是最后一次收到的请求。
	lastReq *http.Request
	// lastQuery 是最后一次收到的查询参数。
	lastQuery url.Values
	// lastPath 是最后一次收到的路径。
	lastPath string
	// hits 记录收到的请求次数。
	hits int
	// handler 可覆盖默认行为。
	handler func(w http.ResponseWriter, r *http.Request)
}

// newFakeServer 构造一个假上游。
func newFakeServer(t *testing.T, body string) *fakeServer {
	t.Helper()
	fs := &fakeServer{}
	fs.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fs.hits++
		fs.lastReq = r
		fs.lastPath = r.URL.Path
		fs.lastQuery = r.URL.Query()
		if fs.handler != nil {
			fs.handler(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(fs.Close)
	return fs
}

// newTestClient 构造一个指向假上游的客户端。
func newTestClient(t *testing.T, fs *fakeServer, opts ...Option) *Client {
	t.Helper()
	base := []Option{WithBaseURL(fs.URL)}
	c, err := NewClient(&APIKeyAuth{key: "test-key"}, append(base, opts...)...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

// assertPath 断言请求路径。
func assertPath(t *testing.T, fs *fakeServer, want string) {
	t.Helper()
	if fs.lastPath != want {
		t.Errorf("path = %q, want %q", fs.lastPath, want)
	}
}

// assertQuery 断言查询参数包含给定键值。
func assertQuery(t *testing.T, fs *fakeServer, key, want string) {
	t.Helper()
	if got := fs.lastQuery.Get(key); got != want {
		t.Errorf("query %q = %q, want %q", key, got, want)
	}
}

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
	fs := newFakeServer(t, body)
	c := newTestClient(t, fs)

	loc, err := ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	got, err := c.Current(context.Background(), loc, WeatherOptions{Lang: "zh", LocalTime: true})
	if err != nil {
		t.Fatalf("Current: %v", err)
	}

	// 路径是纬度在前、经度在后。
	assertPath(t, fs, "/weather/v1/current/39.92/116.41")
	assertQuery(t, fs, "lang", "zh")
	assertQuery(t, fs, "localTime", "true")

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
	fs := newFakeServer(t, "{}")
	c := newTestClient(t, fs)

	// LocationID 没有坐标，新一代接口只认坐标路径参数。
	_, err := c.Current(context.Background(), Location{ID: "101010100"}, WeatherOptions{})
	if err == nil {
		t.Fatal("expected error for LocationID without coordinates")
	}
	if fs.hits != 0 {
		t.Errorf("hits = %d, want 0 (must not reach upstream)", fs.hits)
	}
}

func TestHourlyAndDailyParams(t *testing.T) {
	fs := newFakeServer(t, `{"metadata":{"tag":"t"},"hours":[]}`)
	c := newTestClient(t, fs)
	loc := Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Hourly(context.Background(), loc, 24, WeatherOptions{}); err != nil {
		t.Fatalf("Hourly: %v", err)
	}
	assertPath(t, fs, "/weather/v1/hourly/39.92/116.41")
	assertQuery(t, fs, "hours", "24")

	if _, err := c.Daily(context.Background(), loc, 7, WeatherOptions{}); err != nil {
		t.Fatalf("Daily: %v", err)
	}
	assertPath(t, fs, "/weather/v1/daily/39.92/116.41")
	assertQuery(t, fs, "days", "7")
}

func TestHourlyRejectsOutOfRange(t *testing.T) {
	fs := newFakeServer(t, "{}")
	c := newTestClient(t, fs)
	loc := Location{Lat: 39.92, Lon: 116.41}

	for _, hours := range []int{0, -1, MaxHourlyHours + 1} {
		if _, err := c.Hourly(context.Background(), loc, hours, WeatherOptions{}); err == nil {
			t.Errorf("hours=%d: expected error", hours)
		}
	}
	if fs.hits != 0 {
		t.Errorf("hits = %d, want 0", fs.hits)
	}
}

func TestDailyRejectsOutOfRange(t *testing.T) {
	fs := newFakeServer(t, "{}")
	c := newTestClient(t, fs)
	loc := Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Daily(context.Background(), loc, MaxDailyDays+1, WeatherOptions{}); err == nil {
		t.Error("expected error")
	}
	if fs.hits != 0 {
		t.Errorf("hits = %d, want 0", fs.hits)
	}
}

func TestLookupCity(t *testing.T) {
	body := `{
	  "code": "200",
	  "location": [{
	    "name": "东城", "id": "101011600", "lat": "39.91755", "lon": "116.41876",
	    "adm2": "北京", "adm1": "北京市", "country": "中国",
	    "tz": "Asia/Shanghai", "utcOffset": "+08:00", "isDst": "0",
	    "type": "city", "rank": "35",
	    "fxLink": "https://www.qweather.com/weather/dongcheng-101011600.html"
	  }]
	}`
	fs := newFakeServer(t, body)
	c := newTestClient(t, fs)

	loc, err := ParseLocation("101011600")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	got, err := c.LookupCity(context.Background(), loc, GeoOptions{Lang: "zh"})
	if err != nil {
		t.Fatalf("LookupCity: %v", err)
	}

	assertPath(t, fs, "/geo/v2/city/lookup")
	assertQuery(t, fs, "location", "101011600")
	assertQuery(t, fs, "lang", "zh")

	if len(got.Locations) != 1 {
		t.Fatalf("locations = %d, want 1", len(got.Locations))
	}
	l := got.Locations[0]
	if l.Adm2 != "北京" || l.Name != "东城" {
		t.Errorf("adm2/name = %q/%q", l.Adm2, l.Name)
	}
	// name=区县、adm2=城市，拼出来是"城市+区县"。
	if got := l.DisplayName(); got != "北京东城" {
		t.Errorf("DisplayName = %q, want 北京东城", got)
	}
}

func TestDisplayNameDeduplicatesCity(t *testing.T) {
	// 城市级条目 name 与 adm2 同名，不能拼成"北京北京"。
	l := GeoLocation{Name: "北京", Adm2: "北京"}
	if got := l.DisplayName(); got != "北京" {
		t.Errorf("DisplayName = %q, want 北京", got)
	}
}

func TestCoordinatesFromGeoLocation(t *testing.T) {
	l := GeoLocation{Lat: "39.91755", Lon: "116.41876"}
	lat, lon, err := l.Coordinates()
	if err != nil {
		t.Fatalf("Coordinates: %v", err)
	}
	if lat != 39.91755 || lon != 116.41876 {
		t.Errorf("lat/lon = %v/%v, want 39.91755/116.41876", lat, lon)
	}
}

func TestIndices(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2021-12-16T18:35+08:00",
	  "daily": [{
	    "date": "2021-12-16", "type": "1", "name": "运动指数",
	    "level": "3", "category": "较不宜", "text": "天气较好，但考虑天气寒冷。"
	  }]
	}`
	fs := newFakeServer(t, body)
	c := newTestClient(t, fs)
	loc := Location{Lat: 39.92, Lon: 116.41}

	got, err := c.Indices(context.Background(), loc, IndicesTypeFishing, 1, "")
	if err != nil {
		t.Fatalf("Indices: %v", err)
	}
	assertPath(t, fs, "/v7/indices/1d")
	assertQuery(t, fs, "type", "4")
	assertQuery(t, fs, "location", "116.41,39.92")

	if len(got.Daily) != 1 {
		t.Fatalf("daily = %d, want 1", len(got.Daily))
	}
	if got.Daily[0].Level != "3" {
		t.Errorf("level = %q, want 3", got.Daily[0].Level)
	}
}

func TestIndicesRejectsUnsupportedDays(t *testing.T) {
	fs := newFakeServer(t, "{}")
	c := newTestClient(t, fs)
	loc := Location{Lat: 39.92, Lon: 116.41}

	// 上游只认 1d 与 3d，其余天数码 404。
	for _, days := range []int{2, 7, 10, 30} {
		if _, err := c.Indices(context.Background(), loc, 0, days, ""); err == nil {
			t.Errorf("days=%d: expected error", days)
		}
	}
	if fs.hits != 0 {
		t.Errorf("hits = %d, want 0", fs.hits)
	}
}

func TestMinutely(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2021-12-16T18:55+08:00",
	  "summary": "95分钟后雨就停了",
	  "minutely": [
	    {"fxTime": "2021-12-16T18:55+08:00", "precip": "0.00", "type": "rain"},
	    {"fxTime": "2021-12-16T19:00+08:00", "precip": "0.15", "type": "rain"}
	  ]
	}`
	fs := newFakeServer(t, body)
	c := newTestClient(t, fs)
	loc := Location{Lat: 39.92, Lon: 116.41}

	got, err := c.Minutely(context.Background(), loc, "")
	if err != nil {
		t.Fatalf("Minutely: %v", err)
	}
	assertPath(t, fs, "/v7/minutely/5m")
	assertQuery(t, fs, "location", "116.41,39.92")

	if got.Summary != "95分钟后雨就停了" {
		t.Errorf("summary = %q", got.Summary)
	}
	if len(got.Minutely) != 2 {
		t.Fatalf("minutely = %d, want 2", len(got.Minutely))
	}
	// precip 保持字符串：空串与 "0.00" 含义不同。
	if got.Minutely[0].Precip != "0.00" {
		t.Errorf("precip = %q, want 0.00", got.Minutely[0].Precip)
	}
}

func TestMinutelyRequiresCoordinates(t *testing.T) {
	fs := newFakeServer(t, "{}")
	c := newTestClient(t, fs)

	// 上游只认坐标，LocationID 与地名都不接受。
	_, err := c.Minutely(context.Background(), Location{ID: "101010100"}, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if fs.hits != 0 {
		t.Errorf("hits = %d, want 0", fs.hits)
	}
}

func TestTide(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2021-02-04T05:02+08:00",
	  "tideTable": [{"fxTime": "2021-02-06T03:48+08:00", "height": "2.17", "type": "H"}],
	  "tideHourly": [{"fxTime": "2021-02-06T00:00+08:00", "height": "1.02"}]
	}`
	fs := newFakeServer(t, body)
	c := newTestClient(t, fs)

	got, err := c.Tide(context.Background(), "P2352", "20210206")
	if err != nil {
		t.Fatalf("Tide: %v", err)
	}
	assertPath(t, fs, "/v7/ocean/tide")
	assertQuery(t, fs, "location", "P2352")
	assertQuery(t, fs, "date", "20210206")

	if len(got.TideTable) != 1 || got.TideTable[0].Type != "H" {
		t.Errorf("tideTable = %+v", got.TideTable)
	}
	// 高度保持字符串，保留上游的小数位表达。
	if got.TideTable[0].Height != "2.17" {
		t.Errorf("height = %q, want 2.17", got.TideTable[0].Height)
	}
}

func TestTideRejectsBadDate(t *testing.T) {
	fs := newFakeServer(t, "{}")
	c := newTestClient(t, fs)

	for _, date := range []string{"", "2021-02-06", "2021020", "202102061"} {
		if _, err := c.Tide(context.Background(), "P2352", date); err == nil {
			t.Errorf("date=%q: expected error", date)
		}
	}
	if fs.hits != 0 {
		t.Errorf("hits = %d, want 0", fs.hits)
	}
}

func TestAlerts(t *testing.T) {
	body := `{
	  "metadata": {"tag": "t", "zeroResult": false},
	  "alerts": [{
	    "id": "202510241119105837988676",
	    "senderName": "临桂区气象台",
	    "issuedTime": "2025-10-24T11:19+08:00",
	    "messageType": {"code": "update", "supersedes": ["202510181140100706230391"]},
	    "eventType": {"name": "大风", "code": "1006"},
	    "severity": "minor",
	    "color": {"code": "blue", "red": 30, "green": 50, "blue": 205, "alpha": 1},
	    "effectiveTime": "2025-10-24T11:19+08:00",
	    "headline": "大风蓝色预警信号"
	  }]
	}`
	fs := newFakeServer(t, body)
	c := newTestClient(t, fs)
	loc := Location{Lat: 25.42, Lon: 110.29}

	got, err := c.Alerts(context.Background(), loc, WeatherOptions{})
	if err != nil {
		t.Fatalf("Alerts: %v", err)
	}
	assertPath(t, fs, "/weatheralert/v1/current/25.42/110.29")

	if len(got.Alerts) != 1 {
		t.Fatalf("alerts = %d, want 1", len(got.Alerts))
	}
	a := got.Alerts[0]
	if a.EventType.Code != "1006" || a.EventType.Name != "大风" {
		t.Errorf("eventType = %+v", a.EventType)
	}
	if a.Color.Code != "blue" {
		t.Errorf("color code = %q, want blue", a.Color.Code)
	}
	// severity 有值，urgency 缺失。
	if a.Severity == nil || *a.Severity != "minor" {
		t.Errorf("severity = %v", a.Severity)
	}
	if a.Urgency != nil {
		t.Errorf("urgency should be nil, got %v", *a.Urgency)
	}
}

func TestAirQuality(t *testing.T) {
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
	fs := newFakeServer(t, body)
	c := newTestClient(t, fs)
	loc := Location{Lat: 54.99, Lon: -1.61}

	got, err := c.AirQuality(context.Background(), loc, AirQualityOptions{Lang: "zh"})
	if err != nil {
		t.Fatalf("AirQuality: %v", err)
	}
	assertPath(t, fs, "/airquality/v1/current/54.99/-1.61")

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

func TestLookupPOI(t *testing.T) {
	body := `{
	  "code": "200",
	  "poi": [{"name": "景山公园", "id": "10101010012A", "lat": "39.91999", "lon": "116.38999",
	           "adm2": "北京", "adm1": "北京", "type": "scenic", "rank": "671"}]
	}`
	fs := newFakeServer(t, body)
	c := newTestClient(t, fs)

	got, err := c.LookupPOI(context.Background(), Location{Lat: 39.92, Lon: 116.41}, POIScenic, GeoOptions{})
	if err != nil {
		t.Fatalf("LookupPOI: %v", err)
	}
	assertPath(t, fs, "/geo/v2/poi/lookup")
	assertQuery(t, fs, "type", "scenic")

	if len(got.POIs) != 1 || got.POIs[0].Name != "景山公园" {
		t.Errorf("pois = %+v", got.POIs)
	}
}

func TestLookupPOIRejectsBadType(t *testing.T) {
	fs := newFakeServer(t, "{}")
	c := newTestClient(t, fs)

	if _, err := c.LookupPOI(context.Background(), Location{Lat: 39.92, Lon: 116.41}, POIType("bad"), GeoOptions{}); err == nil {
		t.Error("expected error")
	}
	if fs.hits != 0 {
		t.Errorf("hits = %d, want 0", fs.hits)
	}
}

func TestCacheHitAvoidsUpstream(t *testing.T) {
	fs := newFakeServer(t, `{"metadata":{"tag":"t"},"condition":{"text":"晴","code":"100"}}`)
	cache := NewMemoryCache()
	c := newTestClient(t, fs, WithCache(cache))
	loc := Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Current(context.Background(), loc, WeatherOptions{}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	if fs.hits != 1 {
		t.Fatalf("hits = %d, want 1", fs.hits)
	}

	// 第二次应命中缓存，不再打上游。
	got, err := c.Current(context.Background(), loc, WeatherOptions{})
	if err != nil {
		t.Fatalf("Current (cached): %v", err)
	}
	if fs.hits != 1 {
		t.Errorf("hits = %d, want 1 (cache miss)", fs.hits)
	}
	if got.Condition.Code != "100" {
		t.Errorf("condition code = %q, want 100", got.Condition.Code)
	}
}

func TestCacheKeyIncludesParams(t *testing.T) {
	fs := newFakeServer(t, `{"metadata":{"tag":"t"}}`)
	cache := NewMemoryCache()
	c := newTestClient(t, fs, WithCache(cache))
	loc := Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.Current(context.Background(), loc, WeatherOptions{Lang: "zh"}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	// 语言不同是不同的数据，不能共用缓存。
	if _, err := c.Current(context.Background(), loc, WeatherOptions{Lang: "en"}); err != nil {
		t.Fatalf("Current: %v", err)
	}
	if fs.hits != 2 {
		t.Errorf("hits = %d, want 2 (different lang must not share cache)", fs.hits)
	}
}

func TestCacheFailureDegradesGracefully(t *testing.T) {
	fs := newFakeServer(t, `{"metadata":{"tag":"t"}}`)
	c := newTestClient(t, fs, WithCache(failingCache{}))
	loc := Location{Lat: 39.92, Lon: 116.41}

	// 缓存故障不应让请求失败。
	if _, err := c.Current(context.Background(), loc, WeatherOptions{}); err != nil {
		t.Fatalf("Current with broken cache: %v", err)
	}
	if fs.hits != 1 {
		t.Errorf("hits = %d, want 1", fs.hits)
	}
}

// failingCache 是一个总是失败的缓存实现。
type failingCache struct{}

func (failingCache) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, errCacheBroken
}

func (failingCache) Set(context.Context, string, []byte, time.Duration) error {
	return errCacheBroken
}

var errCacheBroken = &cacheError{}

type cacheError struct{}

func (*cacheError) Error() string { return "cache broken" }

func TestUpstreamErrorClassification(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"bad request", 400, `{"error":{"status":400,"title":"Invalid Parameters","invalidParams":["location"]}}`, ErrUpstreamRejected},
		{"unauthorized", 401, `{"error":{"status":401,"title":"Unauthorized"}}`, ErrUnauthorized},
		{"forbidden", 403, `{"error":{"status":403,"title":"Forbidden"}}`, ErrUnauthorized},
		{"quota", 429, `{"error":{"status":429,"title":"Too Many Requests"}}`, ErrQuotaExceeded},
		{"server error", 500, `{"error":{"status":500,"title":"Internal Server Error"}}`, ErrUpstreamUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs := newFakeServer(t, tc.body)
			fs.handler = func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}
			c := newTestClient(t, fs)
			loc := Location{Lat: 39.92, Lon: 116.41}

			_, err := c.Current(context.Background(), loc, WeatherOptions{})
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("error %v does not match %v", err, tc.want)
			}

			var ue *UpstreamError
			if !errors.As(err, &ue) {
				t.Fatalf("error is not *UpstreamError: %T", err)
			}
			if ue.StatusCode != tc.status {
				t.Errorf("status = %d, want %d", ue.StatusCode, tc.status)
			}
			if ue.Title == "" {
				t.Error("title should be parsed from upstream body")
			}
		})
	}
}

func TestUpstreamErrorParsesInvalidParams(t *testing.T) {
	body := `{"error":{"status":400,"title":"Invalid Parameters","invalidParams":["longitude","latitude"]}}`
	fs := newFakeServer(t, body)
	fs.handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(body))
	}
	c := newTestClient(t, fs)

	_, err := c.Current(context.Background(), Location{Lat: 39.92, Lon: 116.41}, WeatherOptions{})
	var ue *UpstreamError
	if !errors.As(err, &ue) {
		t.Fatalf("error is not *UpstreamError: %T", err)
	}
	if len(ue.InvalidParams) != 2 {
		t.Errorf("invalidParams = %v, want 2 entries", ue.InvalidParams)
	}
	if !strings.Contains(ue.Error(), "longitude") {
		t.Errorf("error message should mention invalid params: %s", ue.Error())
	}
}

func TestLegacyErrorBodyParsing(t *testing.T) {
	// v7 的错误体：{"code":"400","error":"Invalid Parameters"}
	body := `{"code":"400","error":"Invalid Parameters"}`
	fs := newFakeServer(t, body)
	fs.handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(body))
	}
	c := newTestClient(t, fs)

	_, err := c.Warnings(context.Background(), Location{ID: "101010100"}, "")
	var ue *UpstreamError
	if !errors.As(err, &ue) {
		t.Fatalf("error is not *UpstreamError: %T", err)
	}
	if ue.Title != "Invalid Parameters" {
		t.Errorf("title = %q, want Invalid Parameters", ue.Title)
	}
	if ue.Code != "400" {
		t.Errorf("code = %q, want 400", ue.Code)
	}
}
