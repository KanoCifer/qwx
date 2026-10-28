package alert_test

import (
	"context"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/internal/testutil"
	"github.com/KanoCifer/qwx/v1/alert"
)

func TestCurrent(t *testing.T) {
	body := `{
	  "metadata": {"tag": "t", "zeroResult": false},
	  "alerts": [{
	    "id": "202510241119105837988676",
	    "senderName": "临桂区气象台",
	    "issuedTime": "2025-10-24T11:19+08:00",
	    "messageType": {"code": "update", "supersedes": ["202510181140100706230391"]},
	    "eventType": {"name": "大风", "code": "1006"},
	    "severity": "minor",
	    "color": {"code": "blue", "red": 30, "green": 50, "blue": 205, "alpha": 1},
	    "effectiveTime": "2025-10-24T11:19+08:00",
	    "headline": "大风蓝色预警信号"
	  }]
	}`
	fs := testutil.NewFakeServer(t, body)
	c := alert.New(testutil.NewClient(t, fs))
	loc := qwx.Location{Lat: 25.42, Lon: 110.29}

	got, err := c.Current(context.Background(), loc, alert.Options{Lang: "zh", LocalTime: true})
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	fs.AssertPath(t, "/weatheralert/v1/current/25.42/110.29")
	fs.AssertQuery(t, "lang", "zh")
	fs.AssertQuery(t, "localTime", "true")

	if len(got.Alerts) != 1 {
		t.Fatalf("alerts = %d, want 1", len(got.Alerts))
	}
	a := got.Alerts[0]
	if a.EventType.Code != "1006" || a.EventType.Name != "大风" {
		t.Errorf("eventType = %+v", a.EventType)
	}
	if a.Color.Code != "blue" {
		t.Errorf("color code = %q, want blue", a.Color.Code)
	}
	// severity 有值，urgency 缺失。
	if a.Severity == nil || *a.Severity != "minor" {
		t.Errorf("severity = %v", a.Severity)
	}
	if a.Urgency != nil {
		t.Errorf("urgency should be nil, got %v", *a.Urgency)
	}
}

func TestCurrentRejectsLocationID(t *testing.T) {
	fs := testutil.NewFakeServer(t, "{}")
	c := alert.New(testutil.NewClient(t, fs))

	if _, err := c.Current(context.Background(), qwx.Location{ID: "101010100"}, alert.Options{}); err == nil {
		t.Error("expected error")
	}
	if fs.Hits != 0 {
		t.Errorf("hits = %d, want 0", fs.Hits)
	}
}
