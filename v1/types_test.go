package v1_test

import (
	"encoding/json"
	"testing"

	"github.com/KanoCifer/qwx/v1"
)

func TestMetadata(t *testing.T) {
	var m v1.Metadata
	if err := json.Unmarshal([]byte(`{
		"tag": "abc",
		"attributions": ["https://developer.qweather.com/attribution.html"],
		"zeroResult": true
	}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if m.Tag != "abc" {
		t.Errorf("tag = %q, want abc", m.Tag)
	}
	if len(m.Attributions) != 1 {
		t.Errorf("attributions = %v, want 1 item", m.Attributions)
	}
	// zeroResult 是指针：缺失与 false 可区分。
	if m.ZeroResult == nil {
		t.Fatal("zeroResult = nil, want pointer to true")
	}
	if !*m.ZeroResult {
		t.Error("zeroResult = false, want true")
	}
}

func TestMetadataZeroResultMissing(t *testing.T) {
	// 老于该字段的接口版本没有 zeroResult —— 此时必须是 nil，
	// 不能与「查了但结果为空」混淆。
	var m v1.Metadata
	if err := json.Unmarshal([]byte(`{"tag": "abc"}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.ZeroResult != nil {
		t.Errorf("zeroResult = %v, want nil", *m.ZeroResult)
	}
}

func TestValue(t *testing.T) {
	var v v1.Value
	if err := json.Unmarshal([]byte(`{"value": 31.71, "unit": "°C"}`), &v); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if v.Value != 31.71 {
		t.Errorf("value = %v, want 31.71", v.Value)
	}
	if v.Unit != "°C" {
		t.Errorf("unit = %q, want °C", v.Unit)
	}
}
