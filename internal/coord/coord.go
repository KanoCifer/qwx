// Package coord 提供坐标的格式化与规整。
//
// 内部包：坐标只在 SDK 内流转，不对外暴露。
package coord

import "strconv"

// Decimals 是坐标保留的小数位数。
//
// 官方文档写明"最多支持小数点后两位"，实测给更多小数上游也按 1km 网格取值。
const Decimals = 2

// Format 格式化坐标分量，保留两位小数，-0 归一成 0。
//
// 归一 -0 是为了不让 "-0.00" 这种怪值进缓存键。
func Format(v float64) string {
	s := strconv.FormatFloat(v, 'f', Decimals, 64)
	if s == "-0.00" {
		return "0.00"
	}
	return s
}

// Round 把坐标规整到两位小数。
func Round(v float64) float64 {
	f, err := strconv.ParseFloat(Format(v), 64)
	if err != nil {
		return v
	}
	return f
}
