package geo_test

import (
	"context"
	"testing"

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
	  }]
	}`
	fs := testutil.NewFakeServer(t, body)
	c := geo.New(testutil.NewClient(t, fs))

	loc, err := qwx.ParseLocation("101011600")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	got, err := c.LookupCity(context.Background(), loc, geo.Options{Lang: "zh"})
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

	got, err := c.TopCities(context.Background(), geo.Options{Lang: "zh", Number: 5})
	if err != nil {
		t.Fatalf("TopCities: %v", err)
	}
	fs.AssertPath(t, "/geo/v2/city/top")
	fs.AssertQuery(t, "number", "5")

	if len(got.Cities) != 1 || got.Cities[0].ID != "101010100" {
		t.Errorf("cities = %+v", got.Cities)
	}
}
