package coord

import "testing"

func TestFormat(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{39.92, "39.92"},
		{116.41876, "116.42"},
		{0, "0.00"},
		{-1.005, "-1.00"},
	}
	for _, tc := range cases {
		if got := Format(tc.in); got != tc.want {
			t.Errorf("Format(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatNormalizesNegativeZero(t *testing.T) {
	// -0 不能进缓存键，否则同一坐标会出两个键。
	if got := Format(-0.0); got != "0.00" {
		t.Errorf("Format(-0.0) = %q, want 0.00", got)
	}
}

func TestRound(t *testing.T) {
	lat, lon := Round(39.91755), Round(116.41876)
	if lat != 39.92 || lon != 116.42 {
		t.Errorf("Round = %v/%v, want 39.92/116.42", lat, lon)
	}
}
