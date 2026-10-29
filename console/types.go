// Package console 提供控制台 API（/finance/v1/summary、/metrics/v1/stats）。
//
// 控制台 API 查的是帐号自己的财务与请求量，不是天气数据。凭据默认没有
// 权限访问：帐号所有者要在控制台的「项目管理 - 凭据 - 控制台权限」里
// 单独开通对应的权限，未开通时上游直接拒绝。
//
// 上游把这两个端点归在同一页文档下，共享同一套响应骨架（元数据 + asOf），
// 所以放在一个包里。
//
// 两个端点返回的数据都截止至上一个小时或更早，与控制台网页上显示的可能有
// 1 小时或更长的延迟，以响应里的 AsOf 为准。
package console

import (
	"encoding/json"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/cachekey"
)

// 控制台端点的标识，用于 [qwx.TTLPolicy] 按端点匹配缓存时长。
const (
	// EndpointFinance 是财务汇总端点。
	EndpointFinance = qwx.Endpoint("finance/v1/summary")
	// EndpointStats 是请求量统计端点。
	EndpointStats = qwx.Endpoint("metrics/v1/stats")
)

// cacheTTL 是控制台端点的默认缓存时长。
//
// 上游的数据截止至上一个小时或更早，缓存一小时刚好对齐它的更新节奏。
const cacheTTL = time.Hour

// FinanceResult 是财务汇总的结果。
type FinanceResult struct {
	// Metadata 是元数据。
	Metadata qwx.MetaData `json:"metadata"`
	// AsOf 是数据的截止时间，如 "2024-04-03T17:13Z"。
	//
	// 上游给的是 UTC，原样透传。
	AsOf string `json:"asOf"`
	// Currency 是货币代码，如 "CNY"、"USD"。
	Currency string `json:"currency"`
	// Balance 是可用额度。负数表示欠费。
	Balance float64 `json:"balance"`
	// AccruedCharges 是应计费用。
	AccruedCharges AccruedCharges `json:"accruedCharges"`
	// PendingBills 是待支付的账单列表。
	PendingBills []PendingBill `json:"pendingBills"`
	// AvailableSavingsPlans 是可用的节省计划列表。
	AvailableSavingsPlans []SavingsPlan `json:"availableSavingsPlans"`
	// AvailableResourcePlans 是可用的资源包列表。
	AvailableResourcePlans []ResourcePlan `json:"availableResourcePlans"`
}

// AccruedCharges 是应计费用。
type AccruedCharges struct {
	// PreviousDay 是前一天应计费用总额。
	PreviousDay float64 `json:"previousDay"`
	// ThisMonth 是本月应计费用总额。
	ThisMonth float64 `json:"thisMonth"`
	// SinceLastBill 是从上次出账以来的应计费用总额。
	SinceLastBill float64 `json:"sinceLastBill"`
}

// PendingBill 是一张待支付的账单。
type PendingBill struct {
	// Number 是账单号。
	Number string `json:"number"`
	// Date 是出账日期，如 "2024-04-01T10:00Z"。
	//
	// 上游给的是 UTC，原样透传。
	Date string `json:"date"`
	// Type 是账单类型，如 "PayAsYouGo"、"SavingsPlans"。
	//
	// 官方会新增类型，不要依赖枚举的完备性。
	Type string `json:"type"`
	// Status 是账单状态，如 "unpaid"、"overdue"。
	//
	// 官方会新增状态，不要依赖枚举的完备性。
	Status string `json:"status"`
	// Amount 是账单总金额。
	Amount float64 `json:"amount"`
	// AmountDue 是剩余应付金额。
	AmountDue float64 `json:"amountDue"`
	// DueDate 是应付日期，如 "2024-04-10T23:59Z"。
	//
	// 上游给的是 UTC，原样透传。
	DueDate string `json:"dueDate"`
}

// SavingsPlan 是一个可用的节省计划。
type SavingsPlan struct {
	// BillNumber 是账单号。
	BillNumber string `json:"billNumber"`
	// Status 是计划状态，如 "pending"、"active"。
	//
	// 官方会新增状态，不要依赖枚举的完备性。
	Status string `json:"status"`
	// Term 是承诺期限，如 "1"。
	//
	// 上游给的是字符串，原样透传。
	Term string `json:"term"`
	// Commitments 是承诺金额。
	Commitments float64 `json:"commitments"`
	// Utilized 是已用承诺金额。
	Utilized float64 `json:"utilized"`
	// EffectiveTime 是生效时间，如 "2024-04-30T13:00Z"。
	//
	// 上游给的是 UTC，原样透传。
	EffectiveTime string `json:"effectiveTime"`
}

// ResourcePlan 是一个可用的资源包。
type ResourcePlan struct {
	// BillNumber 是账单号。
	BillNumber string `json:"billNumber"`
	// Status 是资源包状态，如 "pending"、"active"。
	//
	// 官方会新增状态，不要依赖枚举的完备性。
	Status string `json:"status"`
	// Requests 是总请求量。
	//
	// 上游的返回样例给的是字符串（"1000000"），官方字段表标的是数字 ——
	// 两种形态都用 [json.Number] 收，调用方按数值用：
	//
	//	n, err := plan.Requests.Int64()
	Requests json.Number `json:"requests"`
	// Commitments 是承诺金额。
	Commitments float64 `json:"commitments"`
	// Utilized 是已使用的请求数量。
	Utilized float64 `json:"utilized"`
	// EffectiveTime 是生效时间，如 "2024-04-03T17:00Z"。
	//
	// 上游给的是 UTC，原样透传。
	EffectiveTime string `json:"effectiveTime"`
}

// StatsResult 是请求量统计的结果。
type StatsResult struct {
	// Metadata 是元数据。
	Metadata qwx.MetaData `json:"metadata"`
	// AsOf 是数据的截止时间，如 "2025-05-12T02:59Z"。
	//
	// 上游给的是 UTC，原样透传。
	AsOf string `json:"asOf"`
	// Success 是各 API 的成功请求量。
	Success []APICount `json:"success"`
	// Errors 是各 API 的错误请求量。
	Errors []APICount `json:"errors"`
}

// APICount 是一个 API 的逐小时请求量。
type APICount struct {
	// API 是 API 名称，如 "Weather"、"Geo"、"AirQuality"。
	//
	// 官方会新增名称，不要依赖枚举的完备性。
	API string `json:"api"`
	// Hours 是最近 24 小时的逐小时请求量。
	//
	// 在 [StatsResult.Success] 里是成功请求量，在 [StatsResult.Errors] 里是
	// 错误请求量。末位对应 AsOf 所在的那一小时：AsOf=2025-03-20T09:59Z 时，
	// 末位是 09:00~09:59（UTC）的请求量。
	//
	// 上游可能给少于 24 条，不要假设长度。
	Hours []int `json:"hours"`
}

// StatsOptions 是请求量统计的可选参数。
//
// 项目与凭据两个过滤条件互斥，都留空时统计帐号下所有项目与凭据
// （含已删除的 —— 已删除的项目或凭据无法单独查询，但请求量仍计入汇总）。
type StatsOptions struct {
	// Project 是项目 ID，只统计这个项目的请求量。
	//
	// 与 Credential 互斥。
	Project string
	// Credential 是凭据 ID，只统计这个凭据的请求量。
	//
	// 与 Project 互斥。
	Credential string
}

// params 是影响结果的查询参数。
//
// 同一份参数既生成上游查询串，也生成缓存键后缀 —— 新增参数时不会漏进键。
func (o StatsOptions) params() cachekey.Params {
	return cachekey.Params{Project: o.Project, Credential: o.Credential}
}
