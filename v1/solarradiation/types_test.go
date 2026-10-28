package solarradiation_test

import (
	"encoding/json"
	"testing"

	"github.com/KanoCifer/qwx/v1/solarradiation"
)

// upstreamBody 是官方文档给的返回样例（截取两条）。
const upstreamBody = `{
  "metadata": {"tag": "c4ca4238a0b923820dcc509a6f75849b"},
  "forecasts": [
    {
      "forecastTime": "2023-10-15T11:30Z",
      "solarAngle": {"azimuth": 184, "elevation": 40},
      "dni": {"value": 25.16, "unit": "W/m²"},
      "dhi": {"value": 136.29, "unit": "W/m²"},
      "ghi": {"value": 152.57, "unit": "W/m²"},
      "weather": {
        "temperature": {"value": 18.6, "unit": "°C"},
        "windSpeed": {"value": 2.78, "unit": "m/s"},
        "humidity": 76
      },
      "poa": {
        "global": {"value": 134.39, "unit": "W/m²"},
        "direct": {"value": 9.35, "unit": "W/m²"},
        "diffuse": {"value": 125.04, "unit": "W/m²"},
        "reflected": {"value": 1.52, "unit": "W/m²"}
      }
    },
    {
      "forecastTime": "2023-10-15T11:45Z",
      "solarAngle": {"azimuth": 189, "elevation": 40},
      "dni": {"value": 20.71, "unit": "W/m²"},
      "dhi": {"value": 132.14, "unit": "W/m²"},
      "ghi": {"value": 145.47, "unit": "W/m²"}
    }
  ]
}`

// TestResultJSONShape 结果类型按上游 JSON 形态解析。
func TestResultJSONShape(t *testing.T) {
	var got solarradiation.Result
	if err := json.Unmarshal([]byte(upstreamBody), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Metadata.Tag != "c4ca4238a0b923820dcc509a6f75849b" {
		t.Errorf("metadata tag = %q", got.Metadata.Tag)
	}
	if len(got.Forecasts) != 2 {
		t.Fatalf("forecasts = %d, want 2", len(got.Forecasts))
	}

	f := got.Forecasts[0]
	if f.ForecastTime != "2023-10-15T11:30Z" {
		t.Errorf("forecastTime = %q", f.ForecastTime)
	}
	if f.SolarAngle.Azimuth != 184 || f.SolarAngle.Elevation != 40 {
		t.Errorf("solarAngle = %+v", f.SolarAngle)
	}
	if f.DNI.Value != 25.16 || f.DNI.Unit != "W/m²" {
		t.Errorf("dni = %+v", f.DNI)
	}
	if f.DHI.Value != 136.29 {
		t.Errorf("dhi = %v, want 136.29", f.DHI.Value)
	}
	if f.GHI.Value != 152.57 {
		t.Errorf("ghi = %v, want 152.57", f.GHI.Value)
	}

	if f.Weather == nil {
		t.Fatal("weather = nil, want the extra=weather block")
	}
	if f.Weather.Temperature.Value != 18.6 || f.Weather.Temperature.Unit != "°C" {
		t.Errorf("temperature = %+v", f.Weather.Temperature)
	}
	if f.Weather.WindSpeed.Value != 2.78 || f.Weather.WindSpeed.Unit != "m/s" {
		t.Errorf("windSpeed = %+v", f.Weather.WindSpeed)
	}
	// 这一族的湿度是百分数（0~100），不是新一代天气端点的 0~1 比例。
	if f.Weather.Humidity != 76 {
		t.Errorf("humidity = %v, want 76", f.Weather.Humidity)
	}

	if f.POA == nil {
		t.Fatal("poa = nil, want the extra=poa block")
	}
	if f.POA.Global.Value != 134.39 {
		t.Errorf("poa global = %v, want 134.39", f.POA.Global.Value)
	}
	if f.POA.Direct.Value != 9.35 {
		t.Errorf("poa direct = %v, want 9.35", f.POA.Direct.Value)
	}
	if f.POA.Diffuse.Value != 125.04 {
		t.Errorf("poa diffuse = %v, want 125.04", f.POA.Diffuse.Value)
	}
	if f.POA.Reflected.Value != 1.52 {
		t.Errorf("poa reflected = %v, want 1.52", f.POA.Reflected.Value)
	}
}

// TestOptionalBlocksAreNilWhenAbsent 没请求 extra 时上游不给这两个对象。
func TestOptionalBlocksAreNilWhenAbsent(t *testing.T) {
	var got solarradiation.Result
	if err := json.Unmarshal([]byte(upstreamBody), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	f := got.Forecasts[1]
	if f.Weather != nil {
		t.Errorf("weather = %+v, want nil", f.Weather)
	}
	if f.POA != nil {
		t.Errorf("poa = %+v, want nil", f.POA)
	}
}

// TestEndpointIdentifier 端点标识是公开契约，取值发布后不再变更。
func TestEndpointIdentifier(t *testing.T) {
	if got := string(solarradiation.Endpoint); got != "solarradiation/v1/forecast" {
		t.Errorf("endpoint = %q, want %q", got, "solarradiation/v1/forecast")
	}
}

// TestMaxForecastHours 上限取官方描述的 72，与 OpenAPI schema 的 60 不一致。
func TestMaxForecastHours(t *testing.T) {
	if solarradiation.MaxForecastHours != 72 {
		t.Errorf("MaxForecastHours = %d, want 72", solarradiation.MaxForecastHours)
	}
}
