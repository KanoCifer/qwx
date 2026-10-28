package astronomy_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/v7/astronomy"
)

// TestEndpointValuesAreFrozen 端点标识的值在已发布版本里定死。
//
// [qwx.TTLPolicy] 按值匹配，改值会让用户的 TTL 配置静默失效。
func TestEndpointValuesAreFrozen(t *testing.T) {
	cases := []struct {
		got  qwx.Endpoint
		want string
	}{
		{astronomy.EndpointSun, "v7/astronomy/sun"},
		{astronomy.EndpointMoon, "v7/astronomy/moon"},
		{astronomy.EndpointAngle, "v7/astronomy/solar-elevation-angle"},
	}
	seen := map[qwx.Endpoint]bool{}
	for _, c := range cases {
		if string(c.got) != c.want {
			t.Errorf("endpoint = %q, want %q", c.got, c.want)
		}
		if seen[c.got] {
			t.Errorf("endpoint %q is used twice", c.got)
		}
		seen[c.got] = true
	}
}

// TestResultJSONTags 字段名与上游一致。
//
// 改了会让上游的数据解析不出来。
func TestResultJSONTags(t *testing.T) {
	cases := []struct {
		name string
		typ  reflect.Type
		want map[string]string
	}{
		{"SunResult", reflect.TypeOf(astronomy.SunResult{}), map[string]string{
			"Code":       "code",
			"UpdateTime": "updateTime",
			"FXLink":     "fxLink",
			"Sunrise":    "sunrise",
			"Sunset":     "sunset",
			"Refer":      "refer",
		}},
		{"MoonResult", reflect.TypeOf(astronomy.MoonResult{}), map[string]string{
			"Code":       "code",
			"UpdateTime": "updateTime",
			"FXLink":     "fxLink",
			"Moonrise":   "moonrise",
			"Moonset":    "moonset",
			"MoonPhase":  "moonPhase",
			"Refer":      "refer",
		}},
		{"MoonPhase", reflect.TypeOf(astronomy.MoonPhase{}), map[string]string{
			"FXTime":       "fxTime",
			"Value":        "value",
			"Name":         "name",
			"Illumination": "illumination",
			"Icon":         "icon",
		}},
		{"AngleResult", reflect.TypeOf(astronomy.AngleResult{}), map[string]string{
			"Code":                "code",
			"SolarElevationAngle": "solarElevationAngle",
			"SolarAzimuthAngle":   "solarAzimuthAngle",
			"SolarHour":           "solarHour",
			"HourAngle":           "hourAngle",
			"Refer":               "refer",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.typ.NumField() != len(c.want) {
				t.Fatalf("fields = %d, want %d", c.typ.NumField(), len(c.want))
			}
			for i := 0; i < c.typ.NumField(); i++ {
				f := c.typ.Field(i)
				want, ok := c.want[f.Name]
				if !ok {
					t.Errorf("unexpected field %q", f.Name)
					continue
				}
				if got := f.Tag.Get("json"); got != want {
					t.Errorf("%s tag = %q, want %q", f.Name, got, want)
				}
			}
		})
	}
}

// TestAngleResultHasNoUpdateTimeOrFXLink 上游这个端点的响应里没有这两个字段。
func TestAngleResultHasNoUpdateTimeOrFXLink(t *testing.T) {
	typ := reflect.TypeOf(astronomy.AngleResult{})
	for _, name := range []string{"UpdateTime", "FXLink"} {
		if _, ok := typ.FieldByName(name); ok {
			t.Errorf("AngleResult should not have field %q", name)
		}
	}
}

// TestSunResultNullableTimes 上游在高纬度地区可能给 null。
func TestSunResultNullableTimes(t *testing.T) {
	var got astronomy.SunResult
	body := `{"code":"200","sunrise":null,"sunset":"19:24"}`
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Sunrise != nil {
		t.Errorf("sunrise = %q, want nil", *got.Sunrise)
	}
	if got.Sunset == nil || *got.Sunset != "19:24" {
		t.Errorf("sunset = %v, want 19:24", got.Sunset)
	}
}

// TestMoonResultNullableTimes 上游说月升月落可能为空。
func TestMoonResultNullableTimes(t *testing.T) {
	var got astronomy.MoonResult
	body := `{"code":"200","moonrise":null,"moonset":"20:11"}`
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Moonrise != nil {
		t.Errorf("moonrise = %q, want nil", *got.Moonrise)
	}
	if got.Moonset == nil || *got.Moonset != "20:11" {
		t.Errorf("moonset = %v, want 20:11", got.Moonset)
	}
}

// TestMoonResultPhaseList 月相列表长度不假设，只透传上游给的。
func TestMoonResultPhaseList(t *testing.T) {
	var got astronomy.MoonResult
	body := `{"code":"200","moonPhase":[
	  {"fxTime":"2026-05-31T20:00+08:00","value":"1","name":"新月","illumination":"0.5","icon":"01"},
	  {"fxTime":"2026-05-31T21:00+08:00","value":"1","name":"峨眉月","illumination":"3.2","icon":"02"}
	]}`
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.MoonPhase) != 2 {
		t.Fatalf("moonPhase = %d, want 2", len(got.MoonPhase))
	}
	if got.MoonPhase[0].Name != "新月" {
		t.Errorf("name = %q, want 新月", got.MoonPhase[0].Name)
	}
	// 照明度保持字符串，保留上游的表达精度。
	if got.MoonPhase[0].Illumination != "0.5" {
		t.Errorf("illumination = %q, want 0.5", got.MoonPhase[0].Illumination)
	}
	if got.MoonPhase[1].Icon != "02" {
		t.Errorf("icon = %q, want 02", got.MoonPhase[1].Icon)
	}
}

// TestAngleResultKeepsStringPrecision 角度保持字符串，不转成浮点。
func TestAngleResultKeepsStringPrecision(t *testing.T) {
	var got astronomy.AngleResult
	body := `{"code":"200","solarElevationAngle":"35.42","solarAzimuthAngle":"196.31","solarHour":"1102","hourAngle":"-12.55"}`
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.SolarElevationAngle != "35.42" {
		t.Errorf("solarElevationAngle = %q, want 35.42", got.SolarElevationAngle)
	}
	if got.SolarAzimuthAngle != "196.31" {
		t.Errorf("solarAzimuthAngle = %q, want 196.31", got.SolarAzimuthAngle)
	}
	if got.SolarHour != "1102" {
		t.Errorf("solarHour = %q, want 1102", got.SolarHour)
	}
	if got.HourAngle != "-12.55" {
		t.Errorf("hourAngle = %q, want -12.55", got.HourAngle)
	}
}
