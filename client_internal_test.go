package qwx

import (
	"errors"
	"testing"
)

func TestNewClientRequiresAPIHost(t *testing.T) {
	for _, host := range []string{"", "   "} {
		_, err := NewClient(&APIKeyAuth{key: "k"}, host)
		if err == nil {
			t.Fatalf("NewClient(host=%q) should fail", host)
		}
		if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("error = %v, want ErrInvalidConfig", err)
		}
	}
}

func TestNewClientNormalizesAPIHost(t *testing.T) {
	cases := []struct {
		name string
		host string
		want string
	}{
		{"host only", "abcxyz.qweatherapi.com", "https://abcxyz.qweatherapi.com"},
		{"with scheme", "https://abcxyz.qweatherapi.com", "https://abcxyz.qweatherapi.com"},
		{"trailing slash", "https://abcxyz.qweatherapi.com/", "https://abcxyz.qweatherapi.com"},
		{"trailing slashes", "abcxyz.qweatherapi.com///", "https://abcxyz.qweatherapi.com"},
		{"surrounding spaces", "  abcxyz.qweatherapi.com  ", "https://abcxyz.qweatherapi.com"},
		{"plain http kept", "http://127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"path kept", "https://gateway.example.com/qwx", "https://gateway.example.com/qwx"},
		// 尾随问号必须去掉：留着会让整个请求路径被吞进查询串。
		{"trailing question mark", "abcxyz.qweatherapi.com?", "https://abcxyz.qweatherapi.com"},
		{"query dropped", "abcxyz.qweatherapi.com?x=1", "https://abcxyz.qweatherapi.com"},
		{"fragment dropped", "abcxyz.qweatherapi.com#frag", "https://abcxyz.qweatherapi.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := NewClient(&APIKeyAuth{key: "k"}, tc.host)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if c.apiHost != tc.want {
				t.Errorf("apiHost = %q, want %q", c.apiHost, tc.want)
			}
		})
	}
}

func TestNewClientRejectsMalformedAPIHost(t *testing.T) {
	for _, host := range []string{"https://", "://abc", "https://a b.com"} {
		if _, err := NewClient(&APIKeyAuth{key: "k"}, host); err == nil {
			t.Errorf("NewClient(host=%q) should fail", host)
		} else if !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("error = %v, want ErrInvalidConfig", err)
		}
	}
}
