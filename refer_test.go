package qwx_test

import (
	"encoding/json"
	"testing"

	"github.com/KanoCifer/qwx"
)

func TestReferUnmarshal(t *testing.T) {
	body := `{
  "sources": ["https://developer.qweather.com/attribution.html"],
  "license": ["QWeather Developers License"]
}`
	var got qwx.Refer
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got.Sources) != 1 || got.Sources[0] != "https://developer.qweather.com/attribution.html" {
		t.Errorf("sources = %v", got.Sources)
	}
	if len(got.License) != 1 || got.License[0] != "QWeather Developers License" {
		t.Errorf("license = %v", got.License)
	}
}
