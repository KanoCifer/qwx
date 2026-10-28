package geo_test

import (
	"context"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/geo"
	"github.com/KanoCifer/qwx/internal/testutil"
)

func TestEndpointIdentifiersAreDistinct(t *testing.T) {
	seen := map[qwx.Endpoint]bool{}
	for _, ep := range []qwx.Endpoint{
		geo.EndpointCity,
		geo.EndpointPOI,
		geo.EndpointPOIRange,
		geo.EndpointTopCities,
	} {
		if seen[ep] {
			t.Errorf("duplicate endpoint identifier %q", ep)
		}
		seen[ep] = true
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
