package qwx_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/KanoCifer/qwx"
)

// TestLangAPI API 代码与打印形态。
func TestLangAPI(t *testing.T) {
	if got := qwx.LangZH.API; got != "zh" {
		t.Errorf("LangZH.API = %q, want %q", got, "zh")
	}
	if got := qwx.LangZH.String(); got != "zh" {
		t.Errorf("LangZH.String() = %q, want %q", got, "zh")
	}
	if got := (qwx.Lang{}).API; got != "" {
		t.Errorf("zero Lang.API = %q, want empty", got)
	}
	if got := qwx.LangZHHant.API; got != "zh-hant" {
		t.Errorf("LangZHHant.API = %q, want %q", got, "zh-hant")
	}
}

// TestLangIsZero 零值语言表示不指定语言。
func TestLangIsZero(t *testing.T) {
	if !(qwx.Lang{}).IsZero() {
		t.Error("zero Lang should be zero")
	}
	if qwx.LangZH.IsZero() {
		t.Error("LangZH should not be zero")
	}
}

// TestLangCompare 语言可以直接用 == 比较。
func TestLangCompare(t *testing.T) {
	if qwx.LangZH != qwx.LangZH {
		t.Error("LangZH should equal itself")
	}
	if qwx.LangZH == qwx.LangEN {
		t.Error("LangZH should not equal LangEN")
	}
	if qwx.LangZH == (qwx.Lang{}) {
		t.Error("LangZH should not equal zero Lang")
	}
}

// TestLangValidate 形态校验。
func TestLangValidate(t *testing.T) {
	tests := []struct {
		name string
		lang qwx.Lang
		ok   bool
	}{
		{"未指定语言", qwx.Lang{}, true},
		{"官方语言", qwx.LangZH, true},
		{"连字符代码", qwx.LangZHHant, true},
		{"三字母代码", qwx.LangFIL, true},
		{"自定义代码", qwx.Lang{API: "xx-yyyy"}, true},
		{"含空格", qwx.Lang{API: "zh hans"}, false},
		{"下划线", qwx.Lang{API: "zh_hans"}, false},
		{"中文", qwx.Lang{API: "中文"}, false},
		{"含斜杠", qwx.Lang{API: "zh/hans"}, false},
		{"尾部连字符", qwx.Lang{API: "zh-"}, false},
		{"超长", qwx.Lang{API: strings.Repeat("a", 33)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lang.Validate()
			if tt.ok {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() = nil, want error")
			}
			if !errors.Is(err, qwx.ErrInvalidParameter) {
				t.Errorf("Validate() error = %v, want ErrInvalidParameter", err)
			}
		})
	}
}

// TestLookupLang 按代码查找语言。
func TestLookupLang(t *testing.T) {
	tests := []struct {
		code string
		want qwx.Lang
		ok   bool
	}{
		{"en", qwx.LangEN, true},
		{"EN", qwx.LangEN, true},
		{"zh", qwx.LangZH, true},
		{"zh-hans", qwx.LangZH, true},
		{"fil", qwx.LangFIL, true},
		{"nb", qwx.LangNB, true},
		{"  en  ", qwx.LangEN, true},
		{"klingon", qwx.Lang{}, false},
		{"", qwx.Lang{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got, ok := qwx.LookupLang(tt.code)
			if ok != tt.ok {
				t.Fatalf("LookupLang(%q) ok = %v, want %v", tt.code, ok, tt.ok)
			}
			if !tt.ok {
				return
			}
			if got != tt.want {
				t.Errorf("LookupLang(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

// TestLangsCatalogue 语言目录的完整性。
func TestLangsCatalogue(t *testing.T) {
	if len(qwx.Langs) != 31 {
		t.Errorf("len(Langs) = %d, want 31", len(qwx.Langs))
	}
	seen := make(map[string]string, len(qwx.Langs))
	for _, l := range qwx.Langs {
		if l.Name == "" {
			t.Errorf("language %q has no name", l.API)
		}
		if err := l.Validate(); err != nil {
			t.Errorf("language %q is invalid: %v", l.API, err)
		}
		if prev, dup := seen[l.API]; dup {
			t.Errorf("API code %q is used by both %q and %q", l.API, prev, l.Name)
			continue
		}
		seen[l.API] = l.Name
	}
}

// TestLangsMatchVars 目录里的每种语言都有对应的导出变量。
func TestLangsMatchVars(t *testing.T) {
	vars := []qwx.Lang{
		qwx.LangZH, qwx.LangZHHant, qwx.LangEN, qwx.LangDE, qwx.LangES,
		qwx.LangFR, qwx.LangIT, qwx.LangJA, qwx.LangKO, qwx.LangRU,
		qwx.LangHI, qwx.LangTH, qwx.LangAR, qwx.LangPT, qwx.LangBN,
		qwx.LangMS, qwx.LangNL, qwx.LangEL, qwx.LangLA, qwx.LangSV,
		qwx.LangID, qwx.LangPL, qwx.LangTR, qwx.LangCS, qwx.LangET,
		qwx.LangVI, qwx.LangFIL, qwx.LangFI, qwx.LangHE, qwx.LangIS,
		qwx.LangNB,
	}
	if len(vars) != len(qwx.Langs) {
		t.Fatalf("exported vars = %d, Langs = %d", len(vars), len(qwx.Langs))
	}
	inList := make(map[string]bool, len(qwx.Langs))
	for _, l := range qwx.Langs {
		inList[l.API] = true
	}
	for _, l := range vars {
		if !inList[l.API] {
			t.Errorf("language %q is not in Langs", l.API)
		}
	}
}

// TestLangZHAlias 简体中文的等价代码查得到同一个语言。
func TestLangZHAlias(t *testing.T) {
	got, ok := qwx.LookupLang("zh-hans")
	if !ok {
		t.Fatal(`LookupLang("zh-hans") not found`)
	}
	if got != qwx.LangZH {
		t.Errorf(`LookupLang("zh-hans") = %v, want %v`, got, qwx.LangZH)
	}
}
