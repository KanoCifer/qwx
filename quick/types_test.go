package quick

import (
	"errors"
	"testing"

	"github.com/KanoCifer/qwx"
)

func TestOptionsParts(t *testing.T) {
	opts := Options{}
	parts, err := opts.parts()
	if err != nil {
		t.Fatalf("parts: %v", err)
	}
	if len(parts) != 5 {
		t.Errorf("parts = %v, want 5 块", parts)
	}

	opts.Skip = []Part{PartCurrent, PartAlerts}
	parts, err = opts.parts()
	if err != nil {
		t.Fatalf("parts: %v", err)
	}
	if len(parts) != 3 {
		t.Errorf("parts = %v, want 3 块", parts)
	}
	for _, p := range parts {
		if p == PartCurrent || p == PartAlerts {
			t.Errorf("part %q 未被跳过", p)
		}
	}
}

func TestOptionsPartsUnknown(t *testing.T) {
	opts := Options{Skip: []Part{"moon"}}
	if _, err := opts.parts(); !errors.Is(err, qwx.ErrInvalidParameter) {
		t.Errorf("error = %v, want ErrInvalidParameter", err)
	}
}

func TestOptionsCounts(t *testing.T) {
	opts := Options{}
	hours, days, err := opts.counts()
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if hours != DefaultHours || days != DefaultDays {
		t.Errorf("hours=%d days=%d, want %d/%d", hours, days, DefaultHours, DefaultDays)
	}

	opts = Options{Hours: 48, Days: 7}
	hours, days, err = opts.counts()
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if hours != 48 || days != 7 {
		t.Errorf("hours=%d days=%d, want 48/7", hours, days)
	}
}

func TestOptionsCountsOutOfRange(t *testing.T) {
	for _, opts := range []Options{
		{Hours: 241},
		{Hours: -1},
		{Days: 11},
		{Days: -1},
	} {
		if _, _, err := opts.counts(); !errors.Is(err, qwx.ErrInvalidParameter) {
			t.Errorf("%+v: error = %v, want ErrInvalidParameter", opts, err)
		}
	}
}

func TestEndpointErrorUnwrap(t *testing.T) {
	err := EndpointError{
		Part:     PartAlerts,
		Endpoint: qwx.Endpoint("weatheralert/v1/current"),
		Err:      qwx.ErrUnauthorized,
	}
	if !errors.Is(err, qwx.ErrUnauthorized) {
		t.Error("errors.Is 未识别出 ErrUnauthorized")
	}
	if err.Error() == "" {
		t.Error("Error() 为空")
	}
}
