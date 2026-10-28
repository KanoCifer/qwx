package geo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/geo"
	"github.com/KanoCifer/qwx/internal/testutil"
)

func TestLookupCity(t *testing.T) {
	body := `{
	  "code": "200",
	  "location": [{
	    "name": "东城", "id": "101011600", "lat": "39.91755", "lon": "116.41876",
	    "adm2": "北京", "adm1": "北京市", "country": "中国",
	    "tz": "Asia/Shanghai", "utcOffset": "+08:00", "isDst": "0",
	    "type": "city", "rank": "35",
	    "fxLink": "https://www.qweather.com/weather/dongcheng-101011600.html"
	  }],
	  "refer": {
	    "sources": ["https://developer.qweather.com/attribution.html"],
	    "license": ["QWeather Developers License"]
	  }
	}`
	fs := testutil.NewFakeServer(t, body)
	c := geo.New(testutil.NewClient(t, fs))

	loc, err := qwx.ParseLocation("101011600")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	got, err := c.LookupCity(context.Background(), loc, geo.Options{Lang: qwx.LangZH})
	if err != nil {
		t.Fatalf("LookupCity: %v", err)
	}

	fs.AssertPath(t, "/geo/v2/city/lookup")
	fs.AssertQuery(t, "location", "101011600")
	fs.AssertQuery(t, "lang", "zh")

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
	// 展示 GeoAPI 数据时要一并展示来源与许可声明。
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

func TestDisplayNameDeduplicatesCity(t *testing.T) {
	// 城市级条目 name 与 adm2 同名，不能拼成"北京北京"。
	l := geo.Place{Name: "北京", Adm2: "北京"}
	if got := l.DisplayName(); got != "北京" {
		t.Errorf("DisplayName = %q, want 北京", got)
	}
}

func TestCoordinatesFromPlace(t *testing.T) {
	l := geo.Place{Lat: "39.91755", Lon: "116.41876"}
	lat, lon, err := l.Coordinates()
	if err != nil {
		t.Fatalf("Coordinates: %v", err)
	}
	if lat != 39.91755 || lon != 116.41876 {
		t.Errorf("lat/lon = %v/%v, want 39.91755/116.41876", lat, lon)
	}
}

func TestLookupPOI(t *testing.T) {
	body := `{
	  "code": "200",
	  "poi": [{"name": "景山公园", "id": "10101010012A", "lat": "39.91999", "lon": "116.38999",
	           "adm2": "北京", "adm1": "北京", "type": "scenic", "rank": "671"}]
	}`
	fs := testutil.NewFakeServer(t, body)
	c := geo.New(testutil.NewClient(t, fs))

	got, err := c.LookupPOI(context.Background(), qwx.Location{Lat: 39.92, Lon: 116.41}, geo.POIScenic, geo.Options{})
	if err != nil {
		t.Fatalf("LookupPOI: %v", err)
	}
	fs.AssertPath(t, "/geo/v2/poi/lookup")
	fs.AssertQuery(t, "type", "scenic")

	if len(got.POIs) != 1 || got.POIs[0].Name != "景山公园" {
		t.Errorf("pois = %+v", got.POIs)
	}
}

func TestLookupPOIRejectsBadType(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := geo.New(testutil.NewClient(t, fs))

	if _, err := c.LookupPOI(context.Background(), qwx.Location{Lat: 39.92, Lon: 116.41}, geo.POIType("bad"), geo.Options{}); err == nil {
		t.Error("expected error")
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestPOIRangeRequiresCoordinates(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := geo.New(testutil.NewClient(t, fs))

	if _, err := c.POIRange(context.Background(), qwx.Location{ID: "101010100"}, geo.POIScenic, geo.POIRangeOptions{}); err == nil {
		t.Error("expected error")
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestTopCities(t *testing.T) {
	body := `{"code": "200", "topCityList": [{"name": "北京", "id": "101010100"}]}`
	fs := testutil.NewFakeServer(t, body)
	c := geo.New(testutil.NewClient(t, fs))

	got, err := c.TopCities(context.Background(), geo.Options{Lang: qwx.LangZH, Number: 5})
	if err != nil {
		t.Fatalf("TopCities: %v", err)
	}
	fs.AssertPath(t, "/geo/v2/city/top")
	fs.AssertQuery(t, "number", "5")

	if len(got.Cities) != 1 || got.Cities[0].ID != "101010100" {
		t.Errorf("cities = %+v", got.Cities)
	}
}

func TestLookupCityReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.LookupCity(context.Background(), loc, geo.Options{}); err != nil {
		t.Fatalf("LookupCity: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != geo.EndpointCity {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, geo.EndpointCity)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

func TestLookupPOIReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.LookupPOI(context.Background(), loc, geo.POIScenic, geo.Options{}); err != nil {
		t.Fatalf("LookupPOI: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != geo.EndpointPOI {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, geo.EndpointPOI)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

func TestPOIRangeReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.POIRange(context.Background(), loc, geo.POIScenic, geo.POIRangeOptions{}); err != nil {
		t.Fatalf("POIRange: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != geo.EndpointPOIRange {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, geo.EndpointPOIRange)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

func TestTopCitiesReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	if _, err := c.TopCities(context.Background(), geo.Options{}); err != nil {
		t.Fatalf("TopCities: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != geo.EndpointTopCities {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, geo.EndpointTopCities)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

// TestOptionalParamsReachQueryAndKey 行政区划、搜索范围与限定城市
//
// 既要发给上游，也要进缓存键 —— 漏进键就是一次串味。
func TestOptionalParamsReachQueryAndKey(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}
	ctx := context.Background()

	if _, err := c.LookupCity(ctx, loc, geo.Options{Lang: qwx.LangZH, Adm: "101010100", Range: "cn"}); err != nil {
		t.Fatalf("LookupCity: %v", err)
	}
	req := d.Last(t)
	if got := req.Query.Get("adm"); got != "101010100" {
		t.Errorf("query adm = %q, want 101010100", got)
	}
	if got := req.Query.Get("range"); got != "cn" {
		t.Errorf("query range = %q, want cn", got)
	}
	if want := "qwx:geo:city:116.41,39.92:lang=zh:adm=101010100:range=cn"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}

	if _, err := c.LookupPOI(ctx, loc, geo.POIScenic, geo.Options{City: "101010100"}); err != nil {
		t.Fatalf("LookupPOI: %v", err)
	}
	req = d.Last(t)
	if got := req.Query.Get("city"); got != "101010100" {
		t.Errorf("query city = %q, want 101010100", got)
	}
	if want := "qwx:geo:poi:scenic:116.41,39.92:city=101010100"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}

	if _, err := c.TopCities(ctx, geo.Options{Range: "world"}); err != nil {
		t.Fatalf("TopCities: %v", err)
	}
	req = d.Last(t)
	if want := "qwx:geo:topcity:range=world"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
//
// 改了会让已写入的缓存条目全部失效。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)

	idLoc, err := qwx.ParseLocation("101011600")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.LookupCity(context.Background(), idLoc, geo.Options{Lang: qwx.LangZH}); err != nil {
		t.Fatalf("LookupCity: %v", err)
	}
	if want := "qwx:geo:city:101011600:lang=zh"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	coordLoc := qwx.Location{Lat: 39.92, Lon: 116.41}
	if _, err := c.LookupPOI(context.Background(), coordLoc, geo.POIScenic, geo.Options{}); err != nil {
		t.Fatalf("LookupPOI: %v", err)
	}
	if want := "qwx:geo:poi:scenic:116.41,39.92"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.POIRange(context.Background(), coordLoc, geo.POIScenic, geo.POIRangeOptions{Lang: qwx.LangZH, Number: 5, Radius: 20}); err != nil {
		t.Fatalf("POIRange: %v", err)
	}
	if want := "qwx:geo:poirange:scenic:116.41,39.92:lang=zh:n=5:r=20"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	// 半径为 0 时键里保留 :r=0，请求里不发 radius —— 既有键字面量的约定。
	if _, err := c.POIRange(context.Background(), coordLoc, geo.POIScenic, geo.POIRangeOptions{}); err != nil {
		t.Fatalf("POIRange: %v", err)
	}
	if want := "qwx:geo:poirange:scenic:116.41,39.92:r=0"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.TopCities(context.Background(), geo.Options{Lang: qwx.LangZH, Number: 5}); err != nil {
		t.Fatalf("TopCities: %v", err)
	}
	if want := "qwx:geo:topcity:lang=zh:n=5"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}

// TestInvalidLangRejects 非法语言代码在客户端就被挡下，不打上游。
func TestInvalidLangRejects(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := geo.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	_, err := c.LookupCity(context.Background(), loc, geo.Options{Lang: qwx.Lang{API: "zh_hans"}})
	if !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Fatalf("LookupCity error = %v, want ErrInvalidParameter", err)
	}
	if len(d.Requests) != 0 {
		t.Errorf("requests = %d, want 0 (invalid lang must not reach upstream)", len(d.Requests))
	}
}
