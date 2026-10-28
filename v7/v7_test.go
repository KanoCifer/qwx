package v7_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7"
)

func TestIndices(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2021-12-16T18:35+08:00",
	  "daily": [{
	    "date": "2021-12-16", "type": "1", "name": "运动指数",
	    "level": "3", "category": "较不宜", "text": "天气较好，但考虑天气寒冷。"
	  }]
	}`
	fs := testutil.NewFakeServer(t, body)
	c := v7.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 39.92, Lon: 116.41}

	got, err := c.Indices(context.Background(), loc, v7.IndicesTypeFishing, 1, "")
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
}

func TestIndicesRejectsUnsupportedDays(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := v7.New(testutil.NewClient(t, fs))
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
	fs := testutil.NewFakeServer(t, body)
	c := v7.New(testutil.NewClient(t, fs))
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
}

func TestMinutelyRequiresCoordinates(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := v7.New(testutil.NewClient(t, fs))

	// 上游只认坐标，LocationID 与地名都不接受。
	if _, err := c.Minutely(context.Background(), qwx.Location{ID: "101010100"}, ""); err == nil {
		t.Fatal("expected error")
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestTide(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2021-02-04T05:02+08:00",
	  "tideTable": [{"fxTime": "2021-02-06T03:48+08:00", "height": "2.17", "type": "H"}],
	  "tideHourly": [{"fxTime": "2021-02-06T00:00+08:00", "height": "1.02"}]
	}`
	fs := testutil.NewFakeServer(t, body)
	c := v7.New(testutil.NewClient(t, fs))

	got, err := c.Tide(context.Background(), "P2352", "20210206")
	if err != nil {
		t.Fatalf("Tide: %v", err)
	}
	fs.AssertPath(t, "/v7/ocean/tide")
	fs.AssertQuery(t, "location", "P2352")
	fs.AssertQuery(t, "date", "20210206")

	if len(got.TideTable) != 1 || got.TideTable[0].Type != "H" {
		t.Errorf("tideTable = %+v", got.TideTable)
	}
	// 高度保持字符串，保留上游的小数位表达。
	if got.TideTable[0].Height != "2.17" {
		t.Errorf("height = %q, want 2.17", got.TideTable[0].Height)
	}
}

func TestTideRejectsBadDate(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := v7.New(testutil.NewClient(t, fs))

	for _, date := range []string{"", "2021-02-06", "2021020", "202102061"} {
		if _, err := c.Tide(context.Background(), "P2352", date); err == nil {
			t.Errorf("date=%q: expected error", date)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestWarnings(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2021-12-16T18:35+08:00",
	  "warning": [{
	    "id": "20211216010133", "sender": "北京市气象台", "pubTime": "2021-12-16T01:01+08:00",
	    "title": "北京市大风蓝色预警", "startTime": "2021-12-16T01:00+08:00",
	    "endTime": "2021-12-17T01:00+08:00", "status": "active", "level": "蓝色",
	    "severity": "Minor", "type": "10002", "typeName": "大风"
	  }]
	}`
	fs := testutil.NewFakeServer(t, body)
	c := v7.New(testutil.NewClient(t, fs))

	got, err := c.Warnings(context.Background(), qwx.Location{ID: "101010100"}, "zh")
	if err != nil {
		t.Fatalf("Warnings: %v", err)
	}
	fs.AssertPath(t, "/v7/warning/now")
	fs.AssertQuery(t, "location", "101010100")

	if len(got.Warnings) != 1 || got.Warnings[0].TypeName != "大风" {
		t.Errorf("warnings = %+v", got.Warnings)
	}
}

func TestLegacyErrorBodyParsing(t *testing.T) {
	// v7 的错误体：{"code":"400","error":"Invalid Parameters"}
	body := `{"code":"400","error":"Invalid Parameters"}`
	fs := testutil.NewFakeServer(t, body)
	fs.Handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(body))
	}
	c := v7.New(testutil.NewClient(t, fs))

	_, err := c.Warnings(context.Background(), qwx.Location{ID: "101010100"}, "")
	var ue *qwx.UpstreamError
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
