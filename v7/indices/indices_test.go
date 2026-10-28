package indices_test

import (
	"context"
	"errors"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7/indices"
)

func TestIndices(t *testing.T) {
	body := `{
  "code": "200",
  "updateTime": "2021-12-16T18:35+08:00",
  "daily": [{
    "date": "2021-12-16", "type": "1", "name": "运动指数",
    "level": "3", "category": "较不宜", "text": "天气较好，但考虑天气寒冷。"
  }],
  "refer": {
    "sources": ["https://developer.qweather.com/attribution.html"],
    "license": ["QWeather Developers License"]
  }
}`
	fs := testutil.NewFakeServer(t, body)
	c := indices.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	got, err := c.Indices(context.Background(), loc, indices.TypeFishing, 1, "")
	if err != nil {
		t.Fatalf("Indices: %v", err)
	}
	fs.AssertPath(t, "/v7/indices/1d")
	fs.AssertQuery(t, "type", "4")
	fs.AssertQuery(t, "location", "116.41,39.92")

	if len(got.Daily) != 1 {
		t.Fatalf("daily = %d, want 1", len(got.Daily))
	}
	if got.Daily[0].Level != "3" {
		t.Errorf("level = %q, want 3", got.Daily[0].Level)
	}
	// 展示上游数据时要一并展示来源与许可声明。
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

func TestIndicesRejectsUnsupportedDays(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := indices.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	// 上游只认 1d 与 3d，其余天数码 404。
	for _, days := range []int{2, 7, 10, 30} {
		if _, err := c.Indices(context.Background(), loc, 0, days, ""); err == nil {
			t.Errorf("days=%d: expected error", days)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestIndicesRejectsOutOfRangeType(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := indices.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	for _, typ := range []int{-1, indices.MaxType + 1} {
		_, err := c.Indices(context.Background(), loc, typ, 1, "")
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("type=%d: err = %v, want ErrInvalidParameter", typ, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestIndicesRequiresCoordinates(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := indices.New(testutil.NewClient(t, fs))

	_, err := c.Indices(context.Background(), qwx.Location{ID: "101010100"}, 0, 1, "")
	if !errors.Is(err, qwx.ErrInvalidLocation) {
		t.Fatalf("err = %v, want ErrInvalidLocation", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}
