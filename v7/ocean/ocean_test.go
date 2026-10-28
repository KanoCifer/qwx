package ocean_test

import (
	"context"
	"errors"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7/ocean"
)

func TestTide(t *testing.T) {
	body := `{
  "code": "200",
  "updateTime": "2021-02-04T05:02+08:00",
  "tideTable": [{"fxTime": "2021-02-06T03:48+08:00", "height": "2.17", "type": "H"}],
  "tideHourly": [{"fxTime": "2021-02-06T00:00+08:00", "height": "1.02"}],
  "refer": {
    "sources": ["https://developer.qweather.com/attribution.html"],
    "license": ["QWeather Developers License"]
  }
}`
	fs := testutil.NewFakeServer(t, body)
	c := ocean.New(testutil.NewClient(t, fs))

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
	// 展示上游数据时要一并展示来源与许可声明。
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

func TestTideRejectsBadDate(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := ocean.New(testutil.NewClient(t, fs))

	for _, date := range []string{"", "2021-02-06", "2021020", "202102061"} {
		if _, err := c.Tide(context.Background(), "P2352", date); err == nil {
			t.Errorf("date=%q: expected error", date)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestTideRejectsEmptyStationID(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := ocean.New(testutil.NewClient(t, fs))

	_, err := c.Tide(context.Background(), "", "20210206")
	if !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Fatalf("err = %v, want ErrInvalidParameter", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}
