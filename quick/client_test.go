package quick_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/quick"
	"github.com/KanoCifer/qwx/v1/alert"
)

// upstream 是按路径应答的假上游。
//
// 并发安全：快速查询会同时发多个请求，记录请求时必须加锁。
type upstream struct {
	*httptest.Server

	mu      sync.Mutex
	paths   []string
	queries []url.Values

	// respond 按路径给出响应。返回的 status 为 0 时用 200。
	respond func(path string) (int, string)
}

func newUpstream(t *testing.T, respond func(path string) (int, string)) *upstream {
	t.Helper()
	u := &upstream{respond: respond}
	u.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.mu.Lock()
		u.paths = append(u.paths, r.URL.Path)
		u.queries = append(u.queries, r.URL.Query())
		u.mu.Unlock()

		status, body := 200, ""
		if u.respond != nil {
			status, body = u.respond(r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(u.Close)
	return u
}

// qwxClient 构造指向假上游的基础客户端。
func (u *upstream) qwxClient(t *testing.T, opts ...qwx.Option) *qwx.Client {
	t.Helper()
	auth, err := qwx.NewAPIKeyAuth("test-key")
	if err != nil {
		t.Fatalf("NewAPIKeyAuth: %v", err)
	}
	c, err := qwx.NewClient(auth, u.URL, opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func (u *upstream) client(t *testing.T) *quick.Client {
	t.Helper()
	return quick.New(u.qwxClient(t))
}

func (u *upstream) hitPaths() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	out := make([]string, len(u.paths))
	copy(out, u.paths)
	return out
}

func (u *upstream) countPath(prefix string) int {
	n := 0
	for _, p := range u.hitPaths() {
		if strings.HasPrefix(p, prefix) {
			n++
		}
	}
	return n
}

// defaultRespond 是各端点的正常响应。
func defaultRespond(path string) (int, string) {
	switch {
	case path == "/geo/v2/city/lookup":
		return 200, "{\"code\":\"200\",\"location\":[{\"id\":\"101010100\",\"name\":\"东城\",\"lat\":\"39.92\",\"lon\":\"116.41\",\"adm1\":\"北京市\",\"adm2\":\"北京\",\"country\":\"中国\"}]}"
	case path == "/weather/v1/current/39.92/116.41":
		return 200, "{\"metadata\":{\"tag\":\"t\"},\"condition\":{\"text\":\"晴\",\"code\":\"100\"},\"temperature\":{\"value\":30.5,\"unit\":\"°C\"}}"
	case path == "/weather/v1/hourly/39.92/116.41":
		return 200, "{\"metadata\":{\"tag\":\"t\"},\"hours\":[{\"forecastTime\":\"2026-01-01T00:00Z\",\"temperature\":{\"value\":29,\"unit\":\"°C\"}}]}"
	case path == "/weather/v1/daily/39.92/116.41":
		return 200, "{\"metadata\":{\"tag\":\"t\"},\"days\":[{\"temperatureMax\":{\"value\":31,\"unit\":\"°C\"}}]}"
	case path == "/airquality/v1/current/39.92/116.41":
		return 200, "{\"metadata\":{\"tag\":\"t\"},\"indexes\":[{\"code\":\"us-epa\",\"aqi\":52}]}"
	case path == "/weatheralert/v1/current/39.92/116.41":
		return 200, "{\"metadata\":{\"tag\":\"t\"},\"alerts\":[]}"
	default:
		return 200, "{}"
	}
}

func TestQueryByCoordinates(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	report, err := c.Query(context.Background(), "116.41,39.92", quick.Options{
		Lang:      qwx.LangZH,
		LocalTime: true,
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	// 坐标直接可用，不必查 GeoAPI。
	if n := u.countPath("/geo/v2/"); n != 0 {
		t.Errorf("geo requests = %d, want 0", n)
	}
	if report.Current == nil || report.Current.Temperature.Value != 30.5 {
		t.Errorf("current = %+v, want 30.5°C", report.Current)
	}
	if report.Hourly == nil || len(report.Hourly.Hours) != 1 {
		t.Errorf("hourly = %+v, want 1 条", report.Hourly)
	}
	if report.Daily == nil || len(report.Daily.Days) != 1 {
		t.Errorf("daily = %+v, want 1 条", report.Daily)
	}
	if report.AirQuality == nil || len(report.AirQuality.Indexes) != 1 {
		t.Errorf("airquality = %+v, want 1 个指数", report.AirQuality)
	}
	// 查得到但没有预警：空列表，不是 nil。
	if report.Alerts == nil || report.Alerts.Alerts == nil {
		t.Errorf("alerts = %+v, want 空列表", report.Alerts)
	}
	if report.Place != nil {
		t.Errorf("place = %+v, want nil", report.Place)
	}
	if len(report.Errors) != 0 {
		t.Errorf("errors = %+v, want 空", report.Errors)
	}
	if report.Location.Lat != 39.92 || report.Location.Lon != 116.41 {
		t.Errorf("location = %+v, want 39.92,116.41", report.Location)
	}
}

func TestQueryByCityName(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	report, err := c.Query(context.Background(), "北京", quick.Options{Lang: qwx.LangZH})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	// 地名先查 GeoAPI 换坐标。
	if n := u.countPath("/geo/v2/city/lookup"); n != 1 {
		t.Errorf("geo requests = %d, want 1", n)
	}
	if report.Place == nil {
		t.Fatal("place = nil, want 解析出的地点")
	}
	if got := report.Place.DisplayName(); got != "北京东城" {
		t.Errorf("place = %q, want 北京东城", got)
	}
	if report.Location.Lat != 39.92 || report.Location.Lon != 116.41 {
		t.Errorf("location = %+v, want 39.92,116.41", report.Location)
	}
	if report.Current == nil {
		t.Error("current = nil, want 实时天气")
	}
}

func TestQueryByLocationID(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	report, err := c.Query(context.Background(), "101010100", quick.Options{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if n := u.countPath("/geo/v2/city/lookup"); n != 1 {
		t.Errorf("geo requests = %d, want 1", n)
	}
	if report.Location.Lat != 39.92 {
		t.Errorf("location = %+v, want 39.92", report.Location)
	}
}

func TestQueryWithLocationStruct(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	report, err := c.Query(context.Background(), qwx.Location{Lat: 39.92, Lon: 116.41}, quick.Options{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if n := u.countPath("/geo/v2/"); n != 0 {
		t.Errorf("geo requests = %d, want 0", n)
	}
	if report.Current == nil {
		t.Error("current = nil, want 实时天气")
	}
}

func TestQueryWithNamedLocationStruct(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	report, err := c.Query(context.Background(), qwx.Location{Name: "北京"}, quick.Options{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if n := u.countPath("/geo/v2/city/lookup"); n != 1 {
		t.Errorf("geo requests = %d, want 1", n)
	}
	if report.Place == nil {
		t.Error("place = nil, want 解析出的地点")
	}
}

func TestQuerySkipPart(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	report, err := c.Query(context.Background(), "116.41,39.92", quick.Options{
		Skip: []quick.Part{quick.PartAlerts, quick.PartAirQuality},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if report.Alerts != nil {
		t.Errorf("alerts = %+v, want nil", report.Alerts)
	}
	if report.AirQuality != nil {
		t.Errorf("airquality = %+v, want nil", report.AirQuality)
	}
	if n := u.countPath("/weatheralert/"); n != 0 {
		t.Errorf("alert requests = %d, want 0", n)
	}
	if n := u.countPath("/airquality/"); n != 0 {
		t.Errorf("aq requests = %d, want 0", n)
	}
	if report.Current == nil {
		t.Error("current = nil, want 实时天气")
	}
}

func TestQueryPartialFailureIsTolerated(t *testing.T) {
	u := newUpstream(t, func(path string) (int, string) {
		if strings.HasPrefix(path, "/weatheralert/") {
			// 账号没开通预警权限。
			return 403, "{\"error\":{\"status\":403,\"title\":\"Forbidden\"}}"
		}
		return defaultRespond(path)
	})
	c := u.client(t)

	report, err := c.Query(context.Background(), "116.41,39.92", quick.Options{})
	if err != nil {
		t.Fatalf("Query: %v, want 宽松模式下不失败", err)
	}
	if len(report.Errors) != 1 {
		t.Fatalf("errors = %+v, want 1 条", report.Errors)
	}
	if !errors.Is(report.Errors[0].Err, qwx.ErrUnauthorized) {
		t.Errorf("error = %v, want ErrUnauthorized", report.Errors[0].Err)
	}
	if report.Errors[0].Part != quick.PartAlerts {
		t.Errorf("part = %q, want alerts", report.Errors[0].Part)
	}
	// 其余照常返回。
	if report.Current == nil {
		t.Error("current = nil, want 实时天气")
	}
}

func TestQueryStrictFailsOnPartialError(t *testing.T) {
	u := newUpstream(t, func(path string) (int, string) {
		if strings.HasPrefix(path, "/weatheralert/") {
			return 403, "{\"error\":{\"status\":403,\"title\":\"Forbidden\"}}"
		}
		return defaultRespond(path)
	})
	c := u.client(t)

	_, err := c.Query(context.Background(), "116.41,39.92", quick.Options{Strict: true})
	if err == nil {
		t.Fatal("expected error in strict mode")
	}
	if !errors.Is(err, qwx.ErrUnauthorized) {
		t.Errorf("error = %v, want ErrUnauthorized", err)
	}
	// 错误要带上失败的数据块与端点，调用方才知道是哪一块挂了。
	var partErr quick.EndpointError
	if !errors.As(err, &partErr) {
		t.Fatalf("error = %v, want quick.EndpointError", err)
	}
	if partErr.Part != quick.PartAlerts {
		t.Errorf("part = %q, want alerts", partErr.Part)
	}
	if partErr.Endpoint != alert.Endpoint {
		t.Errorf("endpoint = %q, want %q", partErr.Endpoint, alert.Endpoint)
	}
}

func TestQueryAllPartsFailed(t *testing.T) {
	u := newUpstream(t, func(string) (int, string) {
		return 500, "{\"error\":{\"status\":500,\"title\":\"Internal Server Error\"}}"
	})
	c := u.client(t)

	_, err := c.Query(context.Background(), "116.41,39.92", quick.Options{})
	if err == nil {
		t.Fatal("expected error when every part failed")
	}
}

func TestQueryInvalidLocation(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	for _, loc := range []any{nil, "", 42} {
		_, err := c.Query(context.Background(), loc, quick.Options{})
		if !errors.Is(err, qwx.ErrInvalidLocation) {
			t.Errorf("loc=%v: error = %v, want ErrInvalidLocation", loc, err)
		}
	}
	if n := len(u.hitPaths()); n != 0 {
		t.Errorf("requests = %d, want 0 (invalid location must not reach upstream)", n)
	}
}

func TestQueryInvalidHoursAndDays(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	if _, err := c.Query(context.Background(), "116.41,39.92", quick.Options{Hours: 241}); !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Errorf("hours: error = %v, want ErrInvalidParameter", err)
	}
	if _, err := c.Query(context.Background(), "116.41,39.92", quick.Options{Days: 11}); !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Errorf("days: error = %v, want ErrInvalidParameter", err)
	}
	if n := len(u.hitPaths()); n != 0 {
		t.Errorf("requests = %d, want 0", n)
	}
}

func TestQueryUnknownPart(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	_, err := c.Query(context.Background(), "116.41,39.92", quick.Options{Skip: []quick.Part{"moon"}})
	if !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Errorf("error = %v, want ErrInvalidParameter", err)
	}
}

func TestQueryInvalidLang(t *testing.T) {
	u := newUpstream(t, defaultRespond)
	c := u.client(t)

	_, err := c.Query(context.Background(), "116.41,39.92", quick.Options{Lang: qwx.Lang{API: "zh hans"}})
	if !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Errorf("error = %v, want ErrInvalidParameter", err)
	}
	if n := len(u.hitPaths()); n != 0 {
		t.Errorf("requests = %d, want 0", n)
	}
}

func TestQueryCityNotFound(t *testing.T) {
	u := newUpstream(t, func(path string) (int, string) {
		if path == "/geo/v2/city/lookup" {
			return 200, "{\"code\":\"200\",\"location\":[]}"
		}
		return defaultRespond(path)
	})
	c := u.client(t)

	_, err := c.Query(context.Background(), "不存在的地方", quick.Options{})
	if !errors.Is(err, qwx.ErrInvalidLocation) {
		t.Errorf("error = %v, want ErrInvalidLocation", err)
	}
	if n := u.countPath("/weather/"); n != 0 {
		t.Errorf("weather requests = %d, want 0", n)
	}
}

func TestQueryUsesEndpointCacheKeys(t *testing.T) {
	// 各块数据的缓存键由各自的端点包生成，组合层不加自己的键。
	cache := testutil.NewRecordingCache()
	u := newUpstream(t, defaultRespond)
	c := quick.New(u.qwxClient(t, qwx.WithCache(cache)))

	if _, err := c.Query(context.Background(), "116.41,39.92", quick.Options{Lang: qwx.LangZH}); err != nil {
		t.Fatalf("Query: %v", err)
	}

	for _, want := range []string{
		"qwx:v1:current:39.92,116.41:lang=zh",
		"qwx:v1:hourly:39.92,116.41:h24:lang=zh",
		"qwx:v1:daily:39.92,116.41:d3:lang=zh",
		"qwx:aq:v1:current:39.92,116.41:lang=zh",
		"qwx:alert:v1:39.92,116.41:lang=zh",
	} {
		if _, ok := cache.TTL(want); !ok {
			t.Errorf("cache key %q not written", want)
		}
	}
}
