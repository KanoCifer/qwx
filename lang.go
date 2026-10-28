package qwx

import (
	"fmt"
	"regexp"
	"strings"
)

// Lang 是一种语言。
//
// 上游用 lang 查询参数切换返回的语言。Lang 给出语言名称与对应的 API 代码，
// 端点的语言参数用这个类型。
//
// 零值表示不指定语言 —— 端点不传 lang 参数，用上游默认（所在国家或地区的
// 官方语言，没有官方语言时回退到英语）。
//
// Lang 的字段都是字符串，可以用 == 比较。
type Lang struct {
	// Name 是语言名称（中文），如 "简体中文"。
	Name string
	// API 是上游 lang 参数的语言代码，如 "zh"。
	//
	// 这是实际发出去的代码，也是进缓存键的代码。官方给简体中文列了两个
	// 等价代码（"zh" 与 "zh-hans"），这里只存首选的 "zh" —— 旧一代接口的
	// 文档明示 zh 是中文的默认值。等价代码用 [LookupLang] 查得到。
	API string
}

// 官方「多语言」页列出的语言。
//
// 变量名取 API 代码的大写形式。
var (
	// LangZH 是简体中文。
	LangZH = Lang{Name: "简体中文", API: "zh"}
	// LangZHHant 是繁体中文。
	LangZHHant = Lang{Name: "繁体中文", API: "zh-hant"}
	// LangEN 是英文。
	LangEN = Lang{Name: "英文", API: "en"}
	// LangDE 是德语。
	LangDE = Lang{Name: "德语", API: "de"}
	// LangES 是西班牙语。
	LangES = Lang{Name: "西班牙语", API: "es"}
	// LangFR 是法语。
	LangFR = Lang{Name: "法语", API: "fr"}
	// LangIT 是意大利语。
	LangIT = Lang{Name: "意大利语", API: "it"}
	// LangJA 是日语。
	LangJA = Lang{Name: "日语", API: "ja"}
	// LangKO 是韩语。
	LangKO = Lang{Name: "韩语", API: "ko"}
	// LangRU 是俄语。
	LangRU = Lang{Name: "俄语", API: "ru"}
	// LangHI 是印地语。
	LangHI = Lang{Name: "印地语", API: "hi"}
	// LangTH 是泰语。
	LangTH = Lang{Name: "泰语", API: "th"}
	// LangAR 是阿拉伯语。
	LangAR = Lang{Name: "阿拉伯语", API: "ar"}
	// LangPT 是葡萄牙语。
	LangPT = Lang{Name: "葡萄牙语", API: "pt"}
	// LangBN 是孟加拉语。
	LangBN = Lang{Name: "孟加拉语", API: "bn"}
	// LangMS 是马来语。
	LangMS = Lang{Name: "马来语", API: "ms"}
	// LangNL 是荷兰语。
	LangNL = Lang{Name: "荷兰语", API: "nl"}
	// LangEL 是希腊语。
	LangEL = Lang{Name: "希腊语", API: "el"}
	// LangLA 是拉丁语。
	LangLA = Lang{Name: "拉丁语", API: "la"}
	// LangSV 是瑞典语。
	LangSV = Lang{Name: "瑞典语", API: "sv"}
	// LangID 是印尼语。
	LangID = Lang{Name: "印尼语", API: "id"}
	// LangPL 是波兰语。
	LangPL = Lang{Name: "波兰语", API: "pl"}
	// LangTR 是土耳其语。
	LangTR = Lang{Name: "土耳其语", API: "tr"}
	// LangCS 是捷克语。
	LangCS = Lang{Name: "捷克语", API: "cs"}
	// LangET 是爱沙尼亚语。
	LangET = Lang{Name: "爱沙尼亚语", API: "et"}
	// LangVI 是越南语。
	LangVI = Lang{Name: "越南语", API: "vi"}
	// LangFIL 是菲律宾语。
	LangFIL = Lang{Name: "菲律宾语", API: "fil"}
	// LangFI 是芬兰语。
	LangFI = Lang{Name: "芬兰语", API: "fi"}
	// LangHE 是希伯来语。
	LangHE = Lang{Name: "希伯来语", API: "he"}
	// LangIS 是冰岛语。
	LangIS = Lang{Name: "冰岛语", API: "is"}
	// LangNB 是挪威语。
	LangNB = Lang{Name: "挪威语", API: "nb"}
)

// Langs 是官方「多语言」页列出的全部语言，顺序与该页一致。
var Langs = []Lang{
	LangZH,
	LangZHHant,
	LangEN,
	LangDE,
	LangES,
	LangFR,
	LangIT,
	LangJA,
	LangKO,
	LangRU,
	LangHI,
	LangTH,
	LangAR,
	LangPT,
	LangBN,
	LangMS,
	LangNL,
	LangEL,
	LangLA,
	LangSV,
	LangID,
	LangPL,
	LangTR,
	LangCS,
	LangET,
	LangVI,
	LangFIL,
	LangFI,
	LangHE,
	LangIS,
	LangNB,
}

// langAliases 是官方给出的等价代码别名。
//
// 官方给简体中文列了两个等价代码（"zh" 与 "zh-hans"）。语言表只存首选的
// "zh"，别名在这里登记 —— 查得到，但不发出去。
var langAliases = map[string]Lang{
	"zh-hans": LangZH,
}

// String 返回 API 语言代码。
//
// 这样打印与拼接语言时拿到的是上游认的代码，不是整个结构体。
func (l Lang) String() string {
	return l.API
}

// IsZero 报告该语言是否未指定。
func (l Lang) IsZero() bool {
	return l.API == ""
}

// Validate 校验语言代码的形态。
//
// 不按官方列表做白名单：官方会新增语言，白名单会把新代码挡在门外。
// 这里只挡笔误 —— 形态不对的代码上游一律 400。
//
// 未指定语言（零值）是合法的：端点不传 lang，用上游默认。
func (l Lang) Validate() error {
	if l.API == "" {
		return nil
	}
	if len(l.API) > maxLangCodeLen || !langCodeRe.MatchString(l.API) {
		return fmt.Errorf("%w: lang %q is not a valid language code", ErrInvalidParameter, l.API)
	}
	return nil
}

// LookupLang 按代码查找语言。
//
// 先按 API 代码匹配，再按等价别名匹配，都不区分大小写。找不到返回 false。
func LookupLang(code string) (Lang, bool) {
	c := strings.ToLower(strings.TrimSpace(code))
	if c == "" {
		return Lang{}, false
	}
	for _, l := range Langs {
		if strings.ToLower(l.API) == c {
			return l, true
		}
	}
	if l, ok := langAliases[c]; ok {
		return l, true
	}
	return Lang{}, false
}

// langCodeRe 语言代码的形态。
//
// 官方代码是小写字母与数字，用连字符分段（如 "zh-hant"）。这里放宽到
// 允许大写与多段 —— 形态校验只挡笔误，不替官方划定代码全集。
var langCodeRe = regexp.MustCompile(`^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$`)

// maxLangCodeLen 语言代码的长度上限。
const maxLangCodeLen = 32
