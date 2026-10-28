package storm_test

import (
	"encoding/json"
	"testing"

	"github.com/KanoCifer/qwx/v7/storm"
)

// TestListResultDecodesUpstreamShape 列表结果按上游 JSON 解码。
//
// 上游的 "storm" 是数组，不是对象 —— 这条容易写错成 map。
func TestListResultDecodesUpstreamShape(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2020-12-31T16:00+00:00",
	  "fxLink": "https://www.qweather.com",
	  "storm": [
	    {"id": "NP2018", "name": "莫拉菲", "basin": "NP", "year": "2020", "isActive": "0"},
	    {"id": "NP2022", "name": "环高", "basin": "NP", "year": "2020", "isActive": "1"}
	  ],
	  "refer": {
	    "sources": ["https://developer.qweather.com/attribution.html"],
	    "license": ["QWeather Developers License"]
	  }
	}`

	var got storm.ListResult
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Code != "200" {
		t.Errorf("code = %q, want 200", got.Code)
	}
	if got.UpdateTime != "2020-12-31T16:00+00:00" {
		t.Errorf("updateTime = %q", got.UpdateTime)
	}
	if got.FXLink != "https://www.qweather.com" {
		t.Errorf("fxLink = %q", got.FXLink)
	}
	if len(got.Storms) != 2 {
		t.Fatalf("storms = %d, want 2", len(got.Storms))
	}
	if got.Storms[0].ID != "NP2018" || got.Storms[0].Name != "莫拉菲" {
		t.Errorf("storm[0] = %+v", got.Storms[0])
	}
	if got.Storms[0].Basin != "NP" || got.Storms[0].Year != "2020" {
		t.Errorf("storm[0] = %+v", got.Storms[0])
	}
	// isActive 上游是字符串 "1"/"0"，不是 bool。
	if got.Storms[1].IsActive != "1" {
		t.Errorf("isActive = %q, want \"1\"", got.Storms[1].IsActive)
	}
	// 展示上游数据时要一并展示来源与许可声明。
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

// TestForecastResultDecodesUpstreamShape 预报结果按上游 JSON 解码。
//
// 数值都是字符串：lat 是 "31.7"，pressure 是 "990"，转成 float64 会丢掉精度。
func TestForecastResultDecodesUpstreamShape(t *testing.T) {
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

	var got storm.ForecastResult
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.Forecast) != 1 {
		t.Fatalf("forecast = %d, want 1", len(got.Forecast))
	}
	f := got.Forecast[0]
	if f.FXTime != "2021-07-27T20:00+08:00" {
		t.Errorf("fxTime = %q", f.FXTime)
	}
	if f.Lat != "31.7" || f.Lon != "118.4" {
		t.Errorf("lat/lon = %q/%q, want 31.7/118.4", f.Lat, f.Lon)
	}
	if f.Type != "TS" {
		t.Errorf("type = %q, want TS", f.Type)
	}
	if f.Pressure != "990" {
		t.Errorf("pressure = %q, want 990", f.Pressure)
	}
	if f.WindSpeed != "18" {
		t.Errorf("windSpeed = %q, want 18", f.WindSpeed)
	}
	// 上游可能给空串，不是缺字段。
	if f.MoveSpeed != "" || f.MoveDir != "" || f.Move360 != "" {
		t.Errorf("move = %q/%q/%q, want empty", f.MoveSpeed, f.MoveDir, f.Move360)
	}
	// 上游没给风圈时是 nil，不是零值半径。
	if f.WindRadius30 != nil || f.WindRadius50 != nil || f.WindRadius64 != nil {
		t.Errorf("windRadius = %+v/%+v/%+v, want nil", f.WindRadius30, f.WindRadius50, f.WindRadius64)
	}
}

// TestForecastWindRadiusDecodes 风圈存在时按上游 JSON 解码。
func TestForecastWindRadiusDecodes(t *testing.T) {
	body := `{
	  "forecast": [{
	    "fxTime": "2024-05-30T13:00+08:00",
	    "windRadius30": {"neRadius": "120", "seRadius": "150", "swRadius": "120", "nwRadius": "100"},
	    "windRadius50": {"neRadius": "60", "seRadius": "60", "swRadius": "60", "nwRadius": "50"}
	  }]
	}`

	var got storm.ForecastResult
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.Forecast) != 1 {
		t.Fatalf("forecast = %d, want 1", len(got.Forecast))
	}
	r30 := got.Forecast[0].WindRadius30
	if r30 == nil {
		t.Fatal("windRadius30 = nil, want radius")
	}
	if r30.NERadius != "120" || r30.SERadius != "150" || r30.SWRadius != "120" || r30.NWRadius != "100" {
		t.Errorf("windRadius30 = %+v", *r30)
	}
	r50 := got.Forecast[0].WindRadius50
	if r50 == nil {
		t.Fatal("windRadius50 = nil, want radius")
	}
	if r50.NWRadius != "50" {
		t.Errorf("windRadius50 = %+v", *r50)
	}
	// 12 级风圈上游没给，仍是 nil。
	if got.Forecast[0].WindRadius64 != nil {
		t.Errorf("windRadius64 = %+v, want nil", got.Forecast[0].WindRadius64)
	}
}

// TestTrackResultDecodesUpstreamShape 实况与路径按上游 JSON 解码。
//
// now 与 track 数组元素平铺同一组台风字段。
func TestTrackResultDecodesUpstreamShape(t *testing.T) {
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

	var got storm.TrackResult
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.IsActive != "1" {
		t.Errorf("isActive = %q, want \"1\"", got.IsActive)
	}
	if got.Now == nil {
		t.Fatal("now = nil, want point")
	}
	if got.Now.PubTime != "2024-05-30T05:00+08:00" {
		t.Errorf("pubTime = %q", got.Now.PubTime)
	}
	if got.Now.Lat != "27.7" || got.Now.Lon != "134.5" {
		t.Errorf("lat/lon = %q/%q, want 27.7/134.5", got.Now.Lat, got.Now.Lon)
	}
	if got.Now.Type != "STS" || got.Now.Pressure != "980" || got.Now.WindSpeed != "30" {
		t.Errorf("now = %+v", *got.Now)
	}
	if got.Now.MoveDir != "NE" {
		t.Errorf("moveDir = %q, want NE", got.Now.MoveDir)
	}
	if got.Now.WindRadius30 == nil || got.Now.WindRadius30.NERadius != "120" {
		t.Errorf("windRadius30 = %+v", got.Now.WindRadius30)
	}
	// 10 级与 12 级风圈上游没给。
	if got.Now.WindRadius50 != nil || got.Now.WindRadius64 != nil {
		t.Errorf("windRadius = %+v/%+v, want nil", got.Now.WindRadius50, got.Now.WindRadius64)
	}
	if len(got.Track) != 1 {
		t.Fatalf("track = %d, want 1", len(got.Track))
	}
	if got.Track[0].Time != "2024-05-29T05:00+08:00" {
		t.Errorf("time = %q", got.Track[0].Time)
	}
	if got.Track[0].Lat != "25.3" {
		t.Errorf("lat = %q, want 25.3", got.Track[0].Lat)
	}
	if len(got.Refer.Sources) != 1 || len(got.Refer.License) != 1 {
		t.Errorf("refer = %+v, want 1 source and 1 license", got.Refer)
	}
}

// TestTrackResultNowIsNullWhenStormEnded 台风结束时上游给 now=null。
//
// 用指针承接 null，nil 表示"没有当前实况"，不是零值坐标。
func TestTrackResultNowIsNullWhenStormEnded(t *testing.T) {
	body := `{
	  "code": "200",
	  "isActive": "0",
	  "now": null,
	  "track": []
	}`

	var got storm.TrackResult
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Now != nil {
		t.Errorf("now = %+v, want nil", got.Now)
	}
	if got.IsActive != "0" {
		t.Errorf("isActive = %q, want \"0\"", got.IsActive)
	}
	if len(got.Track) != 0 {
		t.Errorf("track = %d, want 0", len(got.Track))
	}
}

// TestForecastResultEmptyWhenStormEnded 台风结束时预报列表为空。
//
// 空列表不是错误：上游回 200，forecast 是空数组。
func TestForecastResultEmptyWhenStormEnded(t *testing.T) {
	body := `{
	  "code": "200",
	  "updateTime": "2021-07-27T03:00+00:00",
	  "forecast": []
	}`

	var got storm.ForecastResult
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.Forecast) != 0 {
		t.Errorf("forecast = %d, want 0", len(got.Forecast))
	}
}
