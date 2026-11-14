package param

import (
	"errors"
	"testing"

	"github.com/KanoCifer/qwx"
)

func TestValidateDate(t *testing.T) {
	cases := []struct {
		name string
		date string
		ok   bool
	}{
		{"真实日期", "20260101", true},
		{"闰年2月29日", "20240229", true},
		{"非闰年2月29日", "20250229", false},
		{"13月", "20261301", false},
		{"32日", "20260132", false},
		{"位数不对", "202601", false},
		{"含字母", "2026010a", false},
		{"空串", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDate(tc.date)
			if tc.ok {
				if err != nil {
					t.Fatalf("ValidateDate(%q) = %v, want nil", tc.date, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateDate(%q) = nil, want error", tc.date)
			}
			if !errors.Is(err, qwx.ErrInvalidParameter) {
				t.Errorf("ValidateDate(%q) error = %v, want ErrInvalidParameter", tc.date, err)
			}
		})
	}
}

func TestIsDigits(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"0123456789", true},
		{"12a", false},
		{" 12", false},
		{"1.2", false},
	}
	for _, tc := range cases {
		if got := IsDigits(tc.in); got != tc.want {
			t.Errorf("IsDigits(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
