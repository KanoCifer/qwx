package historical

import (
	"time"

	"github.com/KanoCifer/qwx"
)

// 历史天气端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// Endpoint 是历史天气端点。
	Endpoint = qwx.Endpoint("v7/historical/weather")
)

// 历史天气接口的默认缓存时长：某一天的数据过去之后就不再变。
const cacheTTL = 24 * time.Hour

// Options 是历史天气端点的可选参数。
type Options struct {
	// Lang 是语言，如 [qwx.LangZH]。留空用上游默认。
	Lang qwx.Lang
}

// Result 是历史天气查询的结果。
//
// 上游这个端点的响应里没有 updateTime 与 fxLink 之外的元信息块，本类型
// 也没有对应的字段。
type Result struct {
	// Code 是上游状态码，"200" 为成功。
	Code string `json:"code"`
	// FXLink 是官网链接。
	FXLink string `json:"fxLink"`
	// Daily 是当天的天气汇总。
	//
	// 上游用对象给这一天的汇总数据，不是数组。上游没给这一块时是 nil。
	Daily *Daily `json:"weatherDaily"`
	// Hourly 是当天的逐小时历史天气。
	//
	// 列表长度不假设：上游可能给少于 24 条。
	Hourly []Hourly `json:"weatherHourly"`
	// Refer 是数据来源与许可信息。
	//
	// 上游在结果里带 refer 对象，展示这些数据时需要一并展示。
	Refer qwx.Refer `json:"refer"`
}

// Daily 是一天的历史天气汇总。
//
// 上游把数值都包成字符串，这里保持字符串 —— 小数位数是上游的表达精度，
// 转成 float64 会丢掉它。
type Daily struct {
	// Date 是日期，"YYYY-MM-DD"。
	Date string `json:"date"`
	// Sunrise 是日出时间。
	//
	// 高纬度地区上游可能不给，用指针区分"没有日出"与"空串"。
	Sunrise *string `json:"sunrise"`
	// Sunset 是日落时间。
	//
	// 同 [Daily.Sunrise]：极昼极夜地区可能不给。
	Sunset *string `json:"sunset"`
	// Moonrise 是月升时间。上游可能不给。
	Moonrise *string `json:"moonrise"`
	// Moonset 是月落时间。上游可能不给。
	Moonset *string `json:"moonset"`
	// MoonPhase 是月相名称，如 "峨眉月"。
	//
	// 名称随语言变，上游的取值也不止一种，这里只透传不写完备列表。
	MoonPhase string `json:"moonPhase"`
	// TempMax 是当天最高温度。
	TempMax string `json:"tempMax"`
	// TempMin 是当天最低温度。
	TempMin string `json:"tempMin"`
	// Humidity 是当天相对湿度，百分比数值。
	Humidity string `json:"humidity"`
	// Precip 是当天总降水量，默认单位毫米。
	//
	// 空串表示"无此数据"，与 "0.0"（真的没下雨）不是一回事。
	Precip string `json:"precip"`
	// Pressure 是站点气压，默认单位百帕。
	Pressure string `json:"pressure"`
}

// Hourly 是一小时的历史天气。
//
// 上游把数值都包成字符串，这里保持字符串 —— 小数位数是上游的表达精度，
// 转成 float64 会丢掉它。
//
// 上游个别小时可能不带风向与风速字段，缺失时是空串。
type Hourly struct {
	// Time 是时刻，"YYYY-MM-DD HH:mm"。
	Time string `json:"time"`
	// Temp 是该小时的温度，默认单位摄氏度。
	Temp string `json:"temp"`
	// Icon 是天气状况的图标代码。
	Icon string `json:"icon"`
	// Text 是天气状况的文字描述，如 "晴"。
	//
	// 描述随语言变，上游的取值也不止一种，这里只透传不写完备列表。
	Text string `json:"text"`
	// Wind360 是风向 360 度角度。
	Wind360 string `json:"wind360"`
	// WindDir 是风向，如 "西南风"。
	WindDir string `json:"windDir"`
	// WindScale 是风力等级。
	WindScale string `json:"windScale"`
	// WindSpeed 是风速，默认单位公里/小时。
	WindSpeed string `json:"windSpeed"`
	// Humidity 是该小时的相对湿度，百分比数值。
	Humidity string `json:"humidity"`
	// Precip 是该小时的降水量，默认单位毫米。
	//
	// 空串表示"无此数据"，与 "0.0"（真的没下雨）不是一回事。
	Precip string `json:"precip"`
	// Pressure 是站点气压，默认单位百帕。
	Pressure string `json:"pressure"`
}
