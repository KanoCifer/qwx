package qwx

import (
	"errors"
	"strings"
	"testing"
)

func TestNewUpstreamErrorParsesNewGenerationBody(t *testing.T) {
	// 新一代错误体：error 是对象，带 title 与 invalidParams。
	body := []byte(`{"error":{"status":400,"title":"Invalid Parameters","invalidParams":["longitude","latitude"]}}`)
	e := newUpstreamError(400, body)

	if e.StatusCode != 400 {
		t.Errorf("status = %d, want 400", e.StatusCode)
	}
	if e.Title != "Invalid Parameters" {
		t.Errorf("title = %q, want Invalid Parameters", e.Title)
	}
	if len(e.InvalidParams) != 2 {
		t.Errorf("invalidParams = %v, want 2 entries", e.InvalidParams)
	}
	if e.InvalidParams[0] != "longitude" || e.InvalidParams[1] != "latitude" {
		t.Errorf("invalidParams = %v", e.InvalidParams)
	}
}

func TestNewUpstreamErrorParsesLegacyBody(t *testing.T) {
	// 旧一代错误体：error 是字符串，code 在顶层。
	body := []byte(`{"code":"400","error":"Invalid Parameters"}`)
	e := newUpstreamError(400, body)

	if e.Code != "400" {
		t.Errorf("code = %q, want 400", e.Code)
	}
	if e.Title != "Invalid Parameters" {
		t.Errorf("title = %q, want Invalid Parameters", e.Title)
	}
	// 旧一代不给 invalidParams。
	if len(e.InvalidParams) != 0 {
		t.Errorf("invalidParams = %v, want empty", e.InvalidParams)
	}
}

func TestNewUpstreamErrorKeepsNonJSONBody(t *testing.T) {
	// 网关错误页不是 JSON：解析不出细节，但原始 body 要留住。
	body := []byte(`<html>502 Bad Gateway</html>`)
	e := newUpstreamError(502, body)

	if e.Title != "" {
		t.Errorf("title = %q, want empty", e.Title)
	}
	if e.Body != string(body) {
		t.Errorf("body = %q, want raw body", e.Body)
	}
}

func TestNewUpstreamErrorTruncatesLongBody(t *testing.T) {
	// 超过上限的 body 截断，避免整个响应体进内存与日志。
	body := []byte(strings.Repeat("x", upstreamBodyLimit+100))
	e := newUpstreamError(500, body)

	if !strings.HasSuffix(e.Body, "...(truncated)") {
		t.Errorf("body should be truncated, got %d bytes", len(e.Body))
	}
	if len(e.Body) > upstreamBodyLimit+len("...(truncated)") {
		t.Errorf("body = %d bytes, want at most %d", len(e.Body), upstreamBodyLimit+len("...(truncated)"))
	}
}

func TestUpstreamErrorUnwrapClassification(t *testing.T) {
	cases := []struct {
		name   string
		status int
		want   error
	}{
		{"bad request", 400, ErrUpstreamRejected},
		{"not found", 404, ErrUpstreamRejected},
		{"unauthorized", 401, ErrUnauthorized},
		{"forbidden", 403, ErrUnauthorized},
		{"quota", 429, ErrQuotaExceeded},
		{"server error", 500, ErrUpstreamUnavailable},
		{"bad gateway", 502, ErrUpstreamUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := newUpstreamError(tc.status, []byte(`{}`))
			if !errors.Is(err, tc.want) {
				t.Errorf("status %d: error %v does not match %v", tc.status, err, tc.want)
			}
		})
	}
}

func TestUpstreamErrorMessage(t *testing.T) {
	e := &UpstreamError{
		StatusCode:    400,
		Title:         "Invalid Parameters",
		InvalidParams: []string{"longitude", "latitude"},
	}
	got := e.Error()
	for _, want := range []string{"qwx: upstream error", "(status 400)", "Invalid Parameters", "[invalid: longitude, latitude]"} {
		if !strings.Contains(got, want) {
			t.Errorf("message %q should contain %q", got, want)
		}
	}
}
