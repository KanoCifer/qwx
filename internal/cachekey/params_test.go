package cachekey

import "testing"

func TestParamsQuery(t *testing.T) {
	radius := 5
	q := Params{Lang: "zh", LocalTime: true, Number: 3, Radius: &radius}.Query()

	want := map[string]string{
		"lang":      "zh",
		"localTime": "true",
		"number":    "3",
		"radius":    "5",
	}
	for k, w := range want {
		if got := q.Get(k); got != w {
			t.Errorf("query %q = %q, want %q", k, got, w)
		}
	}
	if len(q) != len(want) {
		t.Errorf("query = %v, want %d 项", q, len(want))
	}
}

// TestParamsQueryOmitsUnset 没设的参数不上报上游。
func TestParamsQueryOmitsUnset(t *testing.T) {
	zero := 0
	q := Params{Number: 0, Radius: &zero}.Query()
	if len(q) != 0 {
		t.Errorf("query = %v, want 空", q)
	}
}

// TestParamsSuffixOrder 后缀顺序固定：语言 → 本地时间 → 条数 → 半径。
//
// 已发布的键字面量按这个顺序排，改了会让旧缓存条目全部失效。
func TestParamsSuffixOrder(t *testing.T) {
	radius := 0
	got := Params{Lang: "zh", LocalTime: true, Number: 5, Radius: &radius}.suffix()
	want := ":lang=zh:localTime=1:n=5:r=0"
	if got != want {
		t.Errorf("suffix = %q, want %q", got, want)
	}
}

// TestParamsSuffixSkipsUnset 空参数不进键，也不留分隔符。
func TestParamsSuffixSkipsUnset(t *testing.T) {
	if got := (Params{}).suffix(); got != "" {
		t.Errorf("suffix = %q, want 空", got)
	}
	if got := (Params{Number: 0}).suffix(); got != "" {
		t.Errorf("suffix = %q, want 空", got)
	}
}

// TestParamsQueryIncludesFamilyParams 新加的参数也要上报上游。
func TestParamsQueryIncludesFamilyParams(t *testing.T) {
	tilt := 0
	azimuth := 180
	q := Params{
		Adm:        "101010100",
		Range:      "cn",
		City:       "101010100",
		Interval:   15,
		Tilt:       &tilt,
		Azimuth:    &azimuth,
		Extra:      "weather,poa",
		Project:    "proj-1",
		Credential: "cred-1",
	}.Query()

	want := map[string]string{
		"adm":        "101010100",
		"range":      "cn",
		"city":       "101010100",
		"interval":   "15",
		"tilt":       "0",
		"azimuth":    "180",
		"extra":      "weather,poa",
		"project":    "proj-1",
		"credential": "cred-1",
	}
	for k, w := range want {
		if got := q.Get(k); got != w {
			t.Errorf("query %q = %q, want %q", k, got, w)
		}
	}
	if len(q) != len(want) {
		t.Errorf("query = %v, want %d 项", q, len(want))
	}
}

// TestParamsSuffixIncludesFamilyParams 新加的参数进键，顺序接在已有参数之后。
//
// 老调用方不设这些字段，键的字节与发布时完全一致。
func TestParamsSuffixIncludesFamilyParams(t *testing.T) {
	tilt := 0
	azimuth := 180
	got := Params{
		Lang:       "zh",
		LocalTime:  true,
		Number:     5,
		Adm:        "101010100",
		Range:      "cn",
		City:       "101010100",
		Interval:   15,
		Tilt:       &tilt,
		Azimuth:    &azimuth,
		Extra:      "poa",
		Project:    "proj-1",
		Credential: "cred-1",
	}.suffix()
	want := ":lang=zh:localTime=1:n=5:adm=101010100:range=cn:city=101010100:interval=15:tilt=0:azimuth=180:extra=poa:project=proj-1:credential=cred-1"
	if got != want {
		t.Errorf("suffix = %q, want %q", got, want)
	}
}

// TestParamsTiltAndAzimuthZeroIsReported 倾斜角与方位角的 0 是合法取值。
//
// 与半径的语义相反：非 nil 时就要上报，不能当"没传"。
func TestParamsTiltAndAzimuthZeroIsReported(t *testing.T) {
	zero := 0
	q := Params{Tilt: &zero, Azimuth: &zero}.Query()
	if got := q.Get("tilt"); got != "0" {
		t.Errorf("query tilt = %q, want 0", got)
	}
	if got := q.Get("azimuth"); got != "0" {
		t.Errorf("query azimuth = %q, want 0", got)
	}
	if got := (Params{Tilt: &zero, Azimuth: &zero}).suffix(); got != ":tilt=0:azimuth=0" {
		t.Errorf("suffix = %q, want \":tilt=0:azimuth=0\"", got)
	}
}
