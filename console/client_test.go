package console_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/console"
	"github.com/KanoCifer/qwx/internal/testutil"
)

// financeBody 是上游文档给的财务汇总返回样例。
const financeBody = `{
  "metadata": {"tag": "2a9873efd0a6c75665dc4f6116b19bdf"},
  "asOf": "2024-04-03T17:13Z",
  "currency": "CNY",
  "balance": -17.54,
  "accruedCharges": {
    "previousDay": 28.92,
    "thisMonth": 113.12,
    "sinceLastBill": 113.12
  },
  "pendingBills": [
    {
      "number": "20681CYX",
      "date": "2024-04-01T10:00Z",
      "type": "PayAsYouGo",
      "status": "unpaid",
      "amount": 535.1,
      "amountDue": 17.54,
      "dueDate": "2024-04-10T23:59Z"
    },
    {
      "number": "605D0FYX",
      "date": "2024-04-02T13:34Z",
      "type": "SavingsPlans",
      "status": "unpaid",
      "amount": 2000,
      "amountDue": 2000,
      "dueDate": "2024-04-30T12:59Z"
    }
  ],
  "availableSavingsPlans": [
    {
      "billNumber": "605D0FYX",
      "status": "pending",
      "term": "1",
      "commitments": 2000,
      "utilized": 0,
      "effectiveTime": "2024-04-30T13:00Z"
    }
  ],
  "availableResourcePlans": [
    {
      "billNumber": "613D1FYX",
      "status": "active",
      "requests": "1000000",
      "commitments": 2000,
      "utilized": 12,
      "effectiveTime": "2024-04-03T17:00Z"
    }
  ]
}`

// statsBody 是上游文档给的请求量统计返回样例。
const statsBody = `{
  "metadata": {"tag": "dcd6183920ebf9f5aa3fd066ed68e61b"},
  "asOf": "2025-05-12T02:59Z",
  "success": [
    {"api": "Weather", "hours": [482, 475, 520]},
    {"api": "Geo", "hours": [20, 16, 0]}
  ],
  "errors": [
    {"api": "Weather", "hours": [3, 1, 4]}
  ]
}`

func TestFinance(t *testing.T) {
	fs := testutil.NewFakeServer(t, financeBody)
	c := console.New(testutil.NewClient(t, fs))

	got, err := c.Finance(context.Background())
	if err != nil {
		t.Fatalf("Finance: %v", err)
	}
	fs.AssertPath(t, "/finance/v1/summary")

	if got.Metadata.Tag != "2a9873efd0a6c75665dc4f6116b19bdf" {
		t.Errorf("tag = %q", got.Metadata.Tag)
	}
	if got.AsOf != "2024-04-03T17:13Z" {
		t.Errorf("asOf = %q", got.AsOf)
	}
	if got.Currency != "CNY" {
		t.Errorf("currency = %q, want CNY", got.Currency)
	}
	if got.Balance != -17.54 {
		t.Errorf("balance = %v, want -17.54", got.Balance)
	}
	if got.AccruedCharges.PreviousDay != 28.92 ||
		got.AccruedCharges.ThisMonth != 113.12 ||
		got.AccruedCharges.SinceLastBill != 113.12 {
		t.Errorf("accruedCharges = %+v", got.AccruedCharges)
	}

	if len(got.PendingBills) != 2 {
		t.Fatalf("pendingBills = %d, want 2", len(got.PendingBills))
	}
	b := got.PendingBills[0]
	if b.Number != "20681CYX" || b.Type != "PayAsYouGo" || b.Status != "unpaid" {
		t.Errorf("bill = %+v", b)
	}
	if b.Amount != 535.1 || b.AmountDue != 17.54 {
		t.Errorf("amount/amountDue = %v/%v, want 535.1/17.54", b.Amount, b.AmountDue)
	}
	if b.Date != "2024-04-01T10:00Z" || b.DueDate != "2024-04-10T23:59Z" {
		t.Errorf("date/dueDate = %q/%q", b.Date, b.DueDate)
	}

	if len(got.AvailableSavingsPlans) != 1 {
		t.Fatalf("savingsPlans = %d, want 1", len(got.AvailableSavingsPlans))
	}
	sp := got.AvailableSavingsPlans[0]
	if sp.BillNumber != "605D0FYX" || sp.Status != "pending" || sp.Term != "1" {
		t.Errorf("savingsPlan = %+v", sp)
	}
	if sp.Commitments != 2000 || sp.Utilized != 0 {
		t.Errorf("commitments/utilized = %v/%v", sp.Commitments, sp.Utilized)
	}

	if len(got.AvailableResourcePlans) != 1 {
		t.Fatalf("resourcePlans = %d, want 1", len(got.AvailableResourcePlans))
	}
	rp := got.AvailableResourcePlans[0]
	// 上游给的是字符串 "1000000"，按数值用。
	n, err := rp.Requests.Int64()
	if err != nil {
		t.Fatalf("requests: %v", err)
	}
	if n != 1000000 {
		t.Errorf("requests = %d, want 1000000", n)
	}
	if rp.Utilized != 12 || rp.EffectiveTime != "2024-04-03T17:00Z" {
		t.Errorf("resourcePlan = %+v", rp)
	}
}

func TestStats(t *testing.T) {
	fs := testutil.NewFakeServer(t, statsBody)
	c := console.New(testutil.NewClient(t, fs))

	got, err := c.Stats(context.Background(), console.StatsOptions{})
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	fs.AssertPath(t, "/metrics/v1/stats")

	if got.Metadata.Tag != "dcd6183920ebf9f5aa3fd066ed68e61b" {
		t.Errorf("tag = %q", got.Metadata.Tag)
	}
	if got.AsOf != "2025-05-12T02:59Z" {
		t.Errorf("asOf = %q", got.AsOf)
	}
	if len(got.Success) != 2 {
		t.Fatalf("success = %d, want 2", len(got.Success))
	}
	if got.Success[0].API != "Weather" {
		t.Errorf("success[0].api = %q, want Weather", got.Success[0].API)
	}
	if len(got.Success[0].Hours) != 3 || got.Success[0].Hours[0] != 482 {
		t.Errorf("success[0].hours = %v", got.Success[0].Hours)
	}
	if len(got.Errors) != 1 || got.Errors[0].API != "Weather" {
		t.Fatalf("errors = %+v", got.Errors)
	}
	if len(got.Errors[0].Hours) != 3 || got.Errors[0].Hours[2] != 4 {
		t.Errorf("errors[0].hours = %v", got.Errors[0].Hours)
	}
}

// TestStatsRejectsBothFilters 项目与凭据互斥，客户端就挡下，不打上游。
func TestStatsRejectsBothFilters(t *testing.T) {
	fs := testutil.NewFakeServer(t, statsBody)
	c := console.New(testutil.NewClient(t, fs))

	_, err := c.Stats(context.Background(), console.StatsOptions{
		Project:    "proj-1",
		Credential: "cred-1",
	})
	if !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Fatalf("Stats error = %v, want ErrInvalidParameter", err)
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0 (invalid options must not reach upstream)", fs.Hits)
	}
}

// TestFinanceReportsEndpointAndTTL 端点标识与默认时长要填进请求。
func TestFinanceReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := console.New(d)

	if _, err := c.Finance(context.Background()); err != nil {
		t.Fatalf("Finance: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != console.EndpointFinance {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, console.EndpointFinance)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
	if req.Path != "/finance/v1/summary" {
		t.Errorf("path = %q", req.Path)
	}
}

func TestStatsReportsEndpointAndTTL(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := console.New(d)

	if _, err := c.Stats(context.Background(), console.StatsOptions{}); err != nil {
		t.Fatalf("Stats: %v", err)
	}

	req := d.Last(t)
	if req.Endpoint != console.EndpointStats {
		t.Errorf("endpoint = %q, want %q", req.Endpoint, console.EndpointStats)
	}
	if req.TTL != time.Hour {
		t.Errorf("ttl = %v, want 1h", req.TTL)
	}
	if req.Path != "/metrics/v1/stats" {
		t.Errorf("path = %q", req.Path)
	}
}

// TestFiltersReachQueryAndKey 过滤条件既发给上游，也进缓存键。
//
// 漏进键就是一次串味：查项目的统计读到了查凭据的统计。
func TestFiltersReachQueryAndKey(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := console.New(d)
	ctx := context.Background()

	if _, err := c.Stats(ctx, console.StatsOptions{Project: "proj-1"}); err != nil {
		t.Fatalf("Stats: %v", err)
	}
	req := d.Last(t)
	if got := req.Query.Get("project"); got != "proj-1" {
		t.Errorf("query project = %q, want proj-1", got)
	}
	if got := req.Query.Get("credential"); got != "" {
		t.Errorf("query credential = %q, want 空", got)
	}
	if want := "qwx:console:stats:project=proj-1"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}

	if _, err := c.Stats(ctx, console.StatsOptions{Credential: "cred-1"}); err != nil {
		t.Fatalf("Stats: %v", err)
	}
	req = d.Last(t)
	if got := req.Query.Get("credential"); got != "cred-1" {
		t.Errorf("query credential = %q, want cred-1", got)
	}
	if got := req.Query.Get("project"); got != "" {
		t.Errorf("query project = %q, want 空", got)
	}
	if want := "qwx:console:stats:credential=cred-1"; req.CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", req.CacheKey, want)
	}
}

// TestCacheKeysAreFrozen 缓存键字面量在已发布版本里定死。
func TestCacheKeysAreFrozen(t *testing.T) {
	d := &testutil.RecordingDoer{}
	c := console.New(d)
	ctx := context.Background()

	if _, err := c.Finance(ctx); err != nil {
		t.Fatalf("Finance: %v", err)
	}
	if want := "qwx:console:finance"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}

	if _, err := c.Stats(ctx, console.StatsOptions{}); err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if want := "qwx:console:stats"; d.Last(t).CacheKey != want {
		t.Errorf("cacheKey = %q, want %q", d.Last(t).CacheKey, want)
	}
}

// TestPermissionDeniedIsUnauthorized 没开通控制台权限时上游回 403。
func TestPermissionDeniedIsUnauthorized(t *testing.T) {
	fs := testutil.NewFakeServer(t, `{"error":{"status":403,"title":"Forbidden"}}`)
	fs.Handler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"status":403,"title":"Forbidden"}}`))
	}
	c := console.New(testutil.NewClient(t, fs))

	_, err := c.Finance(context.Background())
	if !errors.Is(err, qwx.ErrUnauthorized) {
		t.Fatalf("Finance error = %v, want ErrUnauthorized", err)
	}
}
