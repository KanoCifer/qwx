// Package cachekey 拼装缓存键的公共片段。
//
// 内部包：缓存键格式由各端点包自行拼装，这里只放跨包复用的片段。
package cachekey

// Lang 是语言参数对缓存键的影响片段。
//
// 语言不同是不同的数据，必须进键。
func Lang(lang string) string {
	if lang == "" {
		return ""
	}
	return ":lang=" + lang
}
