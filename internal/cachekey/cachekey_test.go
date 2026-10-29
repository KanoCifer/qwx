package cachekey

import "testing"

// TestFamilyPrefixesAreFrozen 族前缀在已发布版本里定死。
//
// 改了前缀，命中不到旧缓存条目，还要让新旧两代共键。
func TestFamilyPrefixesAreFrozen(t *testing.T) {
	want := map[Family]string{
		FamilyWeather:    "qwx:v1:",
		FamilyAirQuality: "qwx:aq:v1:",
		FamilyAlert:      "qwx:alert:v1:",
		FamilyGeo:        "qwx:geo:",
		FamilyTide:       "qwx:tide:",
		FamilyIndices:    "qwx:indices:",
		FamilyMinutely:   "qwx:minutely:",
		FamilyConsole:    "qwx:console:",
	}
	for family, w := range want {
		if string(family) != w {
			t.Errorf("family %q = %q, want %q", family, string(family), w)
		}
	}
}

// TestFamiliesAreDistinct 两代与 GeoAPI 的前缀不能互相嵌套。
//
// 前缀嵌套会让一个族的键落进另一个族的前缀空间。
func TestFamiliesAreDistinct(t *testing.T) {
	all := []Family{
		FamilyWeather, FamilyAirQuality, FamilyAlert,
		FamilyGeo, FamilyTide, FamilyIndices, FamilyMinutely,
		FamilyConsole,
	}
	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}
			if a == b {
				t.Errorf("duplicate family %q", a)
			}
			if len(b) > len(a) && string(b[:len(a)]) == string(a) {
				t.Errorf("family %q is a prefix of %q", a, b)
			}
		}
	}
}

func TestKeyShape(t *testing.T) {
	radius := 5
	tests := []struct {
		name string
		key  string
		want string
	}{
		{
			name: "新一代实况",
			key: New(FamilyWeather).Path("current").PathCoord(39.92, 116.41).
				Params(Params{Lang: "zh"}).String(),
			want: "qwx:v1:current:39.92,116.41:lang=zh",
		},
		{
			name: "计数段排在参数段之前",
			key: New(FamilyWeather).Path("hourly").PathCoord(39.92, 116.41).
				Count("h", 24).
				Params(Params{Lang: "zh", LocalTime: true}).String(),
			want: "qwx:v1:hourly:39.92,116.41:h24:lang=zh:localTime=1",
		},
		{
			name: "GeoAPI 位置段",
			key: New(FamilyGeo).Path("city", "101011600").
				Params(Params{Lang: "zh", Number: 5}).String(),
			want: "qwx:geo:city:101011600:lang=zh:n=5",
		},
		{
			name: "半径进键",
			key: New(FamilyGeo).Path("poirange", "scenic", "116.41,39.92").
				Params(Params{Lang: "zh", Number: 5, Radius: &radius}).String(),
			want: "qwx:geo:poirange:scenic:116.41,39.92:lang=zh:n=5:r=5",
		},
		{
			name: "旧一代潮汐没有参数段",
			key:  New(FamilyTide).Path("P2352", "20260101").String(),
			want: "qwx:tide:P2352:20260101",
		},
		{
			name: "空位置段保留形状",
			key:  New(FamilyGeo).Path("city", "").String(),
			want: "qwx:geo:city:",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.key != tt.want {
				t.Errorf("key = %q, want %q", tt.key, tt.want)
			}
		})
	}
}

// TestPathCoordNormalizesNegativeZero 负零不进键。
func TestPathCoordNormalizesNegativeZero(t *testing.T) {
	got := New(FamilyWeather).Path("current").PathCoord(-0.0001, 0).String()
	want := "qwx:v1:current:0.00,0.00"
	if got != want {
		t.Errorf("key = %q, want %q", got, want)
	}
}

// TestParamsChangeKey 每个影响结果的参数都必须改变键。
//
// 少了任何一个，两个不同的结果就共用一条缓存条目。
func TestParamsChangeKey(t *testing.T) {
	base := func() string {
		return New(FamilyGeo).Path("city", "101011600").String()
	}
	radius := 5
	other := 10
	cases := []struct {
		name string
		a    Params
		b    Params
	}{
		{"语言", Params{Lang: "zh"}, Params{Lang: "en"}},
		{"本地时间", Params{LocalTime: true}, Params{}},
		{"条数", Params{Number: 5}, Params{Number: 10}},
		{"半径", Params{Radius: &radius}, Params{Radius: &other}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			a := New(FamilyGeo).Path("city", "101011600").Params(tt.a).String()
			b := New(FamilyGeo).Path("city", "101011600").Params(tt.b).String()
			if a == b {
				t.Errorf("参数 %q 变了但键没变：%q", tt.name, a)
			}
			if a == base() && tt.a != (Params{}) {
				t.Errorf("参数 %q 没有进键：%q", tt.name, a)
			}
		})
	}
}
