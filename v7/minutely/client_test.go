package minutely_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7/minutely"
)

func TestMinutely(t *testing.T) {
	body := `{
  "code": "200",
  "updateTime": "2021-12-16T18:55+08:00",
  "summary": "95分钟后雨就停了",
  "minutely": [
    {"fxTime": "2021-12-16T18:55+08:00", "precip": "0.00", "type": "rain"},
    {"fxTime": "2021-12-16T19:00+08:00", "precip": "0.15", "type": "rain"}
  ],
  "refer": {
    "sources": ["https://developer.qweather.com/attribution.html"],
    "license": ["QWeather Developers License"]
  }
}`
	fs := testutil.NewFakeServer(t, body)
	c := minutely.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	got, err := c.Minutely(context.Background(), loc, "")
	if err != nil {
		t.Fatalf("Minutely: %v", err)
	}
	fs.AssertPath(t, "/v7/minutely/5m")
	fs.AssertQuery(t, "location", "116.41,39.92")

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
	// 展示上游数据时要一并展示来源与许可声明。
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

func TestMinutelyRequiresCoordinates(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := minutely.New(testutil.NewClient(t, fs))

	// 上游只认坐标，LocationID 与地名都不接受。
	_, err := c.Minutely(context.Background(), qwx.Location{ID: "101010100"}, "")
	if !errors.Is(err, qwx.ErrInvalidLocation) {
		t.Fatalf("err = %v, want ErrInvalidLocation", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestMinutelyReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := minutely.New(d)

	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		t.Fatalf("ParseLocation: %v", err)
	}
	if _, err := c.Minutely(context.Background(), loc, "zh"); err != nil {
		t.Fatalf("Minutely: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != minutely.Endpoint {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, minutely.Endpoint)
	}
	if req.TTL != 5*time.Minute {
		t.Errorf("ttl = %v, want 5m", req.TTL)
	}
	// 缓存键字面量在已发布版本里定死，改了会让旧缓存条目全部失效。
	if want := "qwx:minutely:116.41,39.92:lang=zh"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}
}
