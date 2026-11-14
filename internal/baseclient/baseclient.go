// Package baseclient 收敛端点包构造服务客户端时的共同校验。
//
// 十三个端点包各有一份 "base 不能为 nil，否则 panic" 的样板，panic 文案
// 逐字不同。放进来只有一处真相：漏传客户端时 panic，程序错误不静默。
package baseclient

import "github.com/KanoCifer/qwx"

// Require 在 doer 为 nil 时 panic。
//
// base 为 nil 是编程错误，不是运行时错误：调用方漏传了基础客户端。
// pkg 是端点包名，出现在 panic 文案里，便于定位是哪个端点族。
func Require(doer qwx.Doer, pkg string) {
	if doer == nil {
		panic(pkg + ": nil base client")
	}
}
