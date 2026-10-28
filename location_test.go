package qwx_test

import (
	"testing"

	"github.com/KanoCifer/qwx"
)

func TestParseLocation(t *testing.T) {
	t.Run("coordinate pair", func(t *testing.T) {
		loc, err := qwx.ParseLocation("116.41,39.92")
		if err != nil {
			t.Fatalf("ParseLocation: %v", err)
		}
		if loc.Lat != 39.92 || loc.Lon != 116.41 {
			t.Errorf("lat/lon = %v/%v", loc.Lat, loc.Lon)
		}
		// v7/GeoAPI 的 location 参数是"经度,纬度"。
		if got := loc.Query(); got != "116.41,39.92" {
			t.Errorf("Query = %q, want 116.41,39.92", got)
		}
		// 新一代接口的路径是"纬度,经度"。
		lat, lon, err := loc.Path()
		if err != nil {
			t.Fatalf("Path: %v", err)
		}
		if lat != "39.92" || lon != "116.41" {
			t.Errorf("Path = %q/%q, want 39.92/116.41", lat, lon)
		}
	})

	t.Run("location id", func(t *testing.T) {
		loc, err := qwx.ParseLocation("101010100")
		if err != nil {
			t.Fatalf("ParseLocation: %v", err)
		}
		if loc.ID != "101010100" {
			t.Errorf("ID = %q", loc.ID)
		}
		if loc.HasCoordinates() {
			t.Error("LocationID should not have coordinates")
		}
		if got := loc.Query(); got != "101010100" {
			t.Errorf("Query = %q", got)
		}
	})

	t.Run("poi id with letter suffix", func(t *testing.T) {
		loc, err := qwx.ParseLocation("10101010012A")
		if err != nil {
			t.Fatalf("ParseLocation: %v", err)
		}
		if loc.ID != "10101010012A" {
			t.Errorf("ID = %q", loc.ID)
		}
	})

	t.Run("tide station id", func(t *testing.T) {
		loc, err := qwx.ParseLocation("P2352")
		if err != nil {
			t.Fatalf("ParseLocation: %v", err)
		}
		if loc.ID != "P2352" {
			t.Errorf("ID = %q", loc.ID)
		}
	})

	t.Run("bare longitude is rejected", func(t *testing.T) {
		// 裸经度三种形态都不沾，上游直接 400。
		if _, err := qwx.ParseLocation("121.47"); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("out of range", func(t *testing.T) {
		for _, s := range []string{"181,39.92", "116.41,91", "200,100"} {
			if _, err := qwx.ParseLocation(s); err == nil {
				t.Errorf("%q: expected error", s)
			}
		}
	})

	t.Run("NaN and Inf are rejected", func(t *testing.T) {
		for _, s := range []string{"NaN,39.92", "116.41,NaN", "Inf,0", "0,Inf"} {
			if _, err := qwx.ParseLocation(s); err == nil {
				t.Errorf("%q: expected error", s)
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, err := qwx.ParseLocation("   "); err == nil {
			t.Error("expected error")
		}
	})
}

func TestLocationPathRejectsMissingCoordinates(t *testing.T) {
	loc := qwx.Location{ID: "101010100"}
	if _, _, err := loc.Path(); err == nil {
		t.Error("expected error")
	}
}

func TestLocationQueryPrefersCoordinates(t *testing.T) {
	loc := qwx.Location{ID: "101010100", Lat: 39.92, Lon: 116.41}
	if got := loc.Query(); got != "116.41,39.92" {
		t.Errorf("Query = %q, want coordinates", got)
	}
}

func TestLocationQueryFallsBackToName(t *testing.T) {
	// 地名只有 GeoAPI 认，Query 原样透传。
	loc := qwx.Location{Name: "杭州"}
	if got := loc.Query(); got != "杭州" {
		t.Errorf("Query = %q, want 杭州", got)
	}
}

func TestLocationIsZero(t *testing.T) {
	cases := []struct {
		name string
		loc  qwx.Location
		want bool
	}{
		{"empty", qwx.Location{}, true},
		{"id", qwx.Location{ID: "101010100"}, false},
		{"name", qwx.Location{Name: "杭州"}, false},
		{"coordinates", qwx.Location{Lat: 39.92, Lon: 116.41}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.loc.IsZero(); got != tc.want {
				t.Errorf("IsZero = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLocationCoordinatesRoundsToTwoDecimals(t *testing.T) {
	loc := qwx.Location{Lat: 39.923456, Lon: 116.414567}
	lat, lon, err := loc.Coordinates()
	if err != nil {
		t.Fatalf("Coordinates: %v", err)
	}
	if lat != 39.92 || lon != 116.41 {
		t.Errorf("coordinates = %v/%v, want 39.92/116.41", lat, lon)
	}
}

func TestLocationCoordinatesRejectsMissingCoordinates(t *testing.T) {
	loc := qwx.Location{ID: "101010100"}
	if _, _, err := loc.Coordinates(); err == nil {
		t.Error("expected error")
	}
}
