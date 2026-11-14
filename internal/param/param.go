// Package param 校验跨端点族共用的参数格式。
//
// 参数名在 internal/upstream，键在 internal/cachekey，这里只管"这个值
// 的形态对不对"。
//
// 旧一代三个端点族的日期参数是同一个约定，各自写一份校验，改一处就会
// 漏掉另外两处。
package param

import (
	"fmt"
	"time"

	"github.com/KanoCifer/qwx"
)

// dateLayout 是日期参数的格式。
const dateLayout = "20060102"

// ValidateDate 校验日期参数：8 位数字且是真实日历日期。
//
// 只校验格式，不校验范围 —— 上游对日期范围各有规定（天文最多未来 60 天、
// 历史天气只给最近 10 天），拿"今天"比要引入时钟依赖，超范围的请求交给
// 上游拒绝。
func ValidateDate(date string) error {
	if len(date) != 8 || !IsDigits(date) {
		return fmt.Errorf("%w: date %q must be in yyyyMMdd format", qwx.ErrInvalidParameter, date)
	}
	if _, err := time.Parse(dateLayout, date); err != nil {
		return fmt.Errorf("%w: date %q is not a valid calendar date", qwx.ErrInvalidParameter, date)
	}
	return nil
}

// IsDigits 报告 s 是否全是 ASCII 数字。
func IsDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
