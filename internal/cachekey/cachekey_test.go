package cachekey

import "testing"

func TestLang(t *testing.T) {
	if got := Lang("zh"); got != ":lang=zh" {
		t.Errorf("Lang(zh) = %q, want :lang=zh", got)
	}
	// 空语言不进键，也不留分隔符。
	if got := Lang(""); got != "" {
		t.Errorf("Lang(empty) = %q, want empty", got)
	}
}
