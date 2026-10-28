package storm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v7/storm"
)

func TestList(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2020-12-31T16:00+00:00",
	  "fxLink": "https://www.qweather.com",
	  "storm": [
	    {"id": "NP2018", "name": "莫拉菲", "basin": "NP", "year": "2020", "isActive": "0"}
	  ],
	  "refer": {
	    "sources": ["https://developer.qweather.com/attribution.html"],
	    "license": ["QWeather Developers License"]
	  }
	}`
	fs := testutil.NewFakeServer(t, body)
	c := storm.New(testutil.NewClient(t, fs))

	got, err := c.List(context.Background(), storm.BasinNP, 2020)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	fs.AssertPath(t, "/v7/tropical/storm-list")
	fs.AssertQuery(t, "basin", "NP")
	fs.AssertQuery(t, "year", "2020")

	if len(got.Storms) != 1 {
		t.Fatalf("storms = %d, want 1", len(got.Storms))
	}
	if got.Storms[0].ID != "NP2018" {
		t.Errorf("id = %q, want NP2018", got.Storms[0].ID)
	}
	if got.Storms[0].IsActive != "0" {
		t.Errorf("isActive = %q, want \"0\"", got.Storms[0].IsActive)
	}
	// 展示上游数据时要一并展示来源与许可声明。
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

func TestForecast(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2021-07-27T03:00+00:00",
	  "fxLink": "https://www.qweather.com",
	  "forecast": [
	    {
	      "fxTime": "2021-07-27T20:00+08:00",
	      "lat": "31.7",
	      "lon": "118.4",
	      "type": "TS",
	      "pressure": "990",
	      "windSpeed": "18",
	      "moveSpeed": "",
	      "moveDir": "",
	      "move360": ""
	    }
	  ],
	  "refer": {
	    "sources": ["https://developer.qweather.com/attribution.html"],
	    "license": ["QWeather Developers License"]
	  }
	}`
	fs := testutil.NewFakeServer(t, body)
	c := storm.New(testutil.NewClient(t, fs))

	got, err := c.Forecast(context.Background(), "NP2018")
	if err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	fs.AssertPath(t, "/v7/tropical/storm-forecast")
	fs.AssertQuery(t, "stormid", "NP2018")

	if len(got.Forecast) != 1 {
		t.Fatalf("forecast = %d, want 1", len(got.Forecast))
	}
	if got.Forecast[0].FXTime != "2021-07-27T20:00+08:00" {
		t.Errorf("fxTime = %q", got.Forecast[0].FXTime)
	}
	// 数值保持字符串，不转 float64。
	if got.Forecast[0].Lat != "31.7" {
		t.Errorf("lat = %q, want 31.7", got.Forecast[0].Lat)
	}
	if got.Forecast[0].Pressure != "990" {
		t.Errorf("pressure = %q, want 990", got.Forecast[0].Pressure)
	}
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

func TestTrack(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2024-05-30T06:11+00:00",
	  "fxLink": "https://www.qweather.com",
	  "isActive": "1",
	  "now": {
	    "pubTime": "2024-05-30T05:00+08:00",
	    "lat": "27.7",
	    "lon": "134.5",
	    "type": "STS",
	    "pressure": "980",
	    "windSpeed": "30",
	    "moveSpeed": "21",
	    "moveDir": "NE",
	    "move360": "",
	    "windRadius30": {"neRadius": "120", "seRadius": "150", "swRadius": "120", "nwRadius": "100"}
	  },
	  "track": [
	    {
	      "time": "2024-05-29T05:00+08:00",
	      "lat": "25.3",
	      "lon": "136.1",
	      "type": "STS",
	      "pressure": "980",
	      "windSpeed": "30",
	      "moveSpeed": "21",
	      "moveDir": "NE",
	      "move360": ""
	    }
	  ],
	  "refer": {
	    "sources": ["https://developer.qweather.com/attribution.html"],
	    "license": ["QWeather Developers License"]
	  }
	}`
	fs := testutil.NewFakeServer(t, body)
	c := storm.New(testutil.NewClient(t, fs))

	got, err := c.Track(context.Background(), "NP2018")
	if err != nil {
		t.Fatalf("Track: %v", err)
	}
	fs.AssertPath(t, "/v7/tropical/storm-track")
	fs.AssertQuery(t, "stormid", "NP2018")

	if got.IsActive != "1" {
		t.Errorf("isActive = %q, want \"1\"", got.IsActive)
	}
	if got.Now == nil {
		t.Fatal("now = nil, want point")
	}
	if got.Now.PubTime != "2024-05-30T05:00+08:00" {
		t.Errorf("pubTime = %q", got.Now.PubTime)
	}
	if got.Now.WindRadius30 == nil || got.Now.WindRadius30.NERadius != "120" {
		t.Errorf("windRadius30 = %+v", got.Now.WindRadius30)
	}
	if len(got.Track) != 1 {
		t.Fatalf("track = %d, want 1", len(got.Track))
	}
	if got.Track[0].Time != "2024-05-29T05:00+08:00" {
		t.Errorf("time = %q", got.Track[0].Time)
	}
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

// TestTrackNowIsNullWhenStormEnded 台风已结束时上游给 now=null。
func TestTrackNowIsNullWhenStormEnded(t *testing.T) {
	body := `{"code": "200", "isActive": "0", "now": null, "track": []}`
	fs := testutil.NewFakeServer(t, body)
	c := storm.New(testutil.NewClient(t, fs))

	got, err := c.Track(context.Background(), "NP2018")
	if err != nil {
		t.Fatalf("Track: %v", err)
	}
	if got.Now != nil {
		t.Errorf("now = %+v, want nil", got.Now)
	}
}

// TestListRejectsUnsupportedBasin 上游只支持 NP，其余流域代码客户端就挡掉。
func TestListRejectsUnsupportedBasin(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := storm.New(testutil.NewClient(t, fs))

	// 上游枚举还有 AL、EP、SP、NI、SI，但目前都不支持。
	for _, basin := range []string{"AL", "EP", "SP", "NI", "SI", "", "np"} {
		_, err := c.List(context.Background(), basin, 2020)
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("basin=%q: err = %v, want ErrInvalidParameter", basin, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

// TestListRejectsNonPositiveYear 年份必须是正整数。
func TestListRejectsNonPositiveYear(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := storm.New(testutil.NewClient(t, fs))

	for _, year := range []int{0, -1, -2020} {
		_, err := c.List(context.Background(), storm.BasinNP, year)
		if !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("year=%d: err = %v, want ErrInvalidParameter", year, err)
		}
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

// TestForecastRejectsEmptyStormID 台风 ID 必填。
func TestForecastRejectsEmptyStormID(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := storm.New(testutil.NewClient(t, fs))

	_, err := c.Forecast(context.Background(), "")
	if !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Fatalf("err = %v, want ErrInvalidParameter", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

// TestTrackRejectsEmptyStormID 台风 ID 必填。
func TestTrackRejectsEmptyStormID(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := storm.New(testutil.NewClient(t, fs))

	_, err := c.Track(context.Background(), "")
	if !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Fatalf("err = %v, want ErrInvalidParameter", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}

func TestListReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := storm.New(d)

	if _, err := c.List(context.Background(), storm.BasinNP, 2020); err != nil {
		t.Fatalf("List: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != storm.EndpointList {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, storm.EndpointList)
	}
	if req.TTL != 24*time.Hour {
		t.Errorf("ttl = %v, want 24h", req.TTL)
	}
}

func TestForecastReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := storm.New(d)

	if _, err := c.Forecast(context.Background(), "NP2018"); err != nil {
		t.Fatalf("Forecast: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != storm.EndpointForecast {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, storm.EndpointForecast)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

func TestTrackReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := storm.New(d)

	if _, err := c.Track(context.Background(), "NP2018"); err != nil {
		t.Fatalf("Track: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != storm.EndpointTrack {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, storm.EndpointTrack)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
//
// 改了会让已写入的缓存条目全部失效。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := storm.New(d)

	if _, err := c.List(context.Background(), storm.BasinNP, 2020); err != nil {
		t.Fatalf("List: %v", err)
	}
	if want := "qwx:storm:list:NP:2020"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Forecast(context.Background(), "NP2018"); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	if want := "qwx:storm:forecast:NP2018"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Track(context.Background(), "NP2018"); err != nil {
		t.Fatalf("Track: %v", err)
	}
	if want := "qwx:storm:track:NP2018"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}

// TestCacheKeysDistinguishParameters 影响结果的参数都要进键。
//
// 共键就是串味：不同流域、年份、台风 ID 必须是不同的键。
func TestCacheKeysDistinguishParameters(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := storm.New(d)

	keys := map[string]bool{}
	record := func(key string) {
		t.Helper()
		if keys[key] {
			t.Errorf("duplicate cacheKey %q", key)
		}
		keys[key] = true
	}

	if _, err := c.List(context.Background(), storm.BasinNP, 2020); err != nil {
		t.Fatalf("List: %v", err)
	}
	record(d.Last(t).CacheKey)

	if _, err := c.List(context.Background(), storm.BasinNP, 2021); err != nil {
		t.Fatalf("List: %v", err)
	}
	record(d.Last(t).CacheKey)

	if _, err := c.Forecast(context.Background(), "NP2018"); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	record(d.Last(t).CacheKey)

	if _, err := c.Forecast(context.Background(), "NP2020"); err != nil {
		t.Fatalf("Forecast: %v", err)
	}
	record(d.Last(t).CacheKey)

	if _, err := c.Track(context.Background(), "NP2018"); err != nil {
		t.Fatalf("Track: %v", err)
	}
	record(d.Last(t).CacheKey)

	if _, err := c.Track(context.Background(), "NP2020"); err != nil {
		t.Fatalf("Track: %v", err)
	}
	record(d.Last(t).CacheKey)

	if len(keys) != 6 {
		t.Errorf("keys = %d, want 6", len(keys))
	}
}

// TestEndpointIdentifiersAreFrozen 端点标识的值发布后不再变更。
//
// TTLPolicy 按值匹配，改值会让用户的 TTL 配置静默失效。
func TestEndpointIdentifiersAreFrozen(t *testing.T) {
	for _, c := range []struct {
		endpoint qwx.Endpoint
		want     string
	}{
		{storm.EndpointList, "v7/tropical/storm-list"},
		{storm.EndpointForecast, "v7/tropical/storm-forecast"},
		{storm.EndpointTrack, "v7/tropical/storm-track"},
	} {
		if string(c.endpoint) != c.want {
			t.Errorf("endpoint = %q, want %q", c.endpoint, c.want)
		}
	}
}
