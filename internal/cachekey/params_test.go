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
