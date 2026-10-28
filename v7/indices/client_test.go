package indices_test

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestIndicesReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := indices.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Indices(context.Background(), loc, 0, 3, "zh"); err != nil {
		t.Fatalf("Indices: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != indices.Endpoint {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, indices.Endpoint)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
	// 缓存键字面量在已发布版本里定死，改了会让旧缓存条目全部失效。
	if want := "qwx:indices:116.41,39.92:t0:d3:lang=zh"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}
}

// TestIndicesMulti 一次请求拿多种指数：上游的 type 支持逗号分隔。
func TestIndicesMulti(t *testing.T) {
	body := `{
  "code": "200",
  "daily": [
    {"date": "2021-12-16", "type": "3", "name": "运动指数", "level": "3", "category": "较不宜", "text": ""},
    {"date": "2021-12-16", "type": "4", "name": "钓鱼指数", "level": "2", "category": "适宜", "text": ""}
  ]
}`
	fs := testutil.NewFakeServer(t, body)
	c := indices.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	got, err := c.IndicesMulti(context.Background(), loc, []int{3, 4}, 1, "zh")
	if err != nil {
		t.Fatalf("IndicesMulti: %v", err)
	}
	fs.AssertPath(t, "/v7/indices/1d")
	fs.AssertQuery(t, "type", "3,4")
	fs.AssertQuery(t, "location", "116.41,39.92")
	if len(got.Daily) != 2 {
		t.Fatalf("daily = %d, want 2", len(got.Daily))
	}
}

// TestIndicesMultiCacheKey 多种类型的键用逗号连接，单类型时与旧键一致。
func TestIndicesMultiCacheKey(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := indices.New(d)
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	if _, err := c.IndicesMulti(context.Background(), loc, []int{3, 4}, 1, "zh"); err != nil {
		t.Fatalf("IndicesMulti: %v", err)
	}
	if want := "qwx:indices:116.41,39.92:t3,4:d1:lang=zh"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	// 单类型的键与旧的单类型调用完全一致。
	if _, err := c.IndicesMulti(context.Background(), loc, []int{3}, 1, "zh"); err != nil {
		t.Fatalf("IndicesMulti: %v", err)
	}
	if want := "qwx:indices:116.41,39.92:t3:d1:lang=zh"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}

// TestIndicesMultiRejectsBadInput 非法入参在打上游之前就被挡住。
func TestIndicesMultiRejectsBadInput(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := indices.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}
	ctx := context.Background()

	// 类型列表不能为空。
	if _, err := c.IndicesMulti(ctx, loc, nil, 1, ""); !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Errorf("err = %v, want ErrInvalidParameter", err)
	}
	// 类型取值越界。
	if _, err := c.IndicesMulti(ctx, loc, []int{3, indices.MaxType + 1}, 1, ""); !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Errorf("err = %v, want ErrInvalidParameter", err)
	}
	// 取全部时不能与其他类型混用。
	if _, err := c.IndicesMulti(ctx, loc, []int{indices.TypeAll, 3}, 1, ""); !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Errorf("err = %v, want ErrInvalidParameter", err)
	}
	// 天数只支持 1 与 3。
	if _, err := c.IndicesMulti(ctx, loc, []int{3}, 7, ""); !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Errorf("err = %v, want ErrInvalidParameter", err)
	}
	// 位置必须是坐标对。
	if _, err := c.IndicesMulti(ctx, qwx.Location{ID: "101010100"}, []int{3}, 1, ""); !errors.Is(err, qwx.ErrInvalidLocation) {
		t.Errorf("err = %v, want ErrInvalidLocation", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}
