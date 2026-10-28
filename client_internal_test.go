package qwx

import (
	"errors"
	"net/http"
	"testing"
	"time"
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

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name  string
		value string
		want  time.Duration
		ok    bool
	}{
		{"seconds", "120", 2 * time.Minute, true},
		{"http date", now.Add(90 * time.Second).Format(http.TimeFormat), 90 * time.Second, true},
		{"past date", now.Add(-time.Minute).Format(http.TimeFormat), 0, true},
		{"empty", "", 0, false},
		{"garbage", "later", 0, false},
		{"negative", "-5", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseRetryAfter(tc.value, now)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("delay = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNextRetryDelay(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := []struct {
		name       string
		attempt    int
		retryAfter string
		want       time.Duration
	}{
		{"backoff", 1, "", time.Second},
		{"backoff grows", 3, "", 3 * time.Second},
		{"retry-after raises the floor", 1, "5", 5 * time.Second},
		{"retry-after does not shorten the backoff", 3, "1", 3 * time.Second},
		{"retry-after is capped", 1, "3600", maxRetryDelay},
		{"invalid retry-after ignored", 2, "soon", 2 * time.Second},
		{"negative retry-after ignored", 2, "-5", 2 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextRetryDelay(tc.attempt, tc.retryAfter, now); got != tc.want {
				t.Errorf("nextRetryDelay = %v, want %v", got, tc.want)
			}
		})
	}
}
