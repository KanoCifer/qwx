package qwx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// SDK 自身的错误。用 errors.Is 判断，不要直接比字符串。
var (
	// ErrInvalidConfig 表示构造参数有问题（缺认证、空私钥等）。
	ErrInvalidConfig = errors.New("qwx: invalid config")
	// ErrInvalidLocation 表示 location 参数的形态上游不认。
	ErrInvalidLocation = errors.New("qwx: invalid location")
	// ErrInvalidParameter 表示端点参数取值越界。
	ErrInvalidParameter = errors.New("qwx: invalid parameter")
	// ErrInvalidResponse 表示上游返回的不是合法 JSON。
	ErrInvalidResponse = errors.New("qwx: invalid response")
	// ErrUpstreamRejected 表示上游 4xx：请求本身被判为非法。
	// 重试没有意义，调用方应修正请求。
	ErrUpstreamRejected = errors.New("qwx: upstream rejected the request")
	// ErrQuotaExceeded 表示上游 429：配额或频率限制，稍后重试有意义。
	ErrQuotaExceeded = errors.New("qwx: quota exceeded")
	// ErrUnauthorized 表示上游 401/403：认证凭据无效或权限不足。
	ErrUnauthorized = errors.New("qwx: unauthorized")
	// ErrUpstreamUnavailable 表示上游 5xx 或网络错误。
	ErrUpstreamUnavailable = errors.New("qwx: upstream unavailable")
)

// UpstreamError 是上游返回错误时的具体错误。
//
// 通过 errors.Is 判断它归哪一类（[ErrUpstreamRejected] / [ErrQuotaExceeded] /
// [ErrUnauthorized] / [ErrUpstreamUnavailable]），通过字段读上游给的细节。
type UpstreamError struct {
	// StatusCode 是上游的 HTTP 状态码。
	StatusCode int
	// Code 是上游返回的业务状态码（如 "400"）。
	Code string
	// Title 是上游给的一句话原因（如 "Invalid Parameters"）。
	Title string
	// InvalidParams 是上游指出的非法参数名，新一代接口会给出。
	InvalidParams []string
	// Body 是上游的原始响应体（截断到 2KB）。
	Body string
}

// Error 实现 error 接口。
func (e *UpstreamError) Error() string {
	var b strings.Builder
	b.WriteString("qwx: upstream error")
	if e.StatusCode != 0 {
		fmt.Fprintf(&b, " (status %d)", e.StatusCode)
	}
	if e.Title != "" {
		b.WriteString(": ")
		b.WriteString(e.Title)
	}
	if len(e.InvalidParams) > 0 {
		fmt.Fprintf(&b, " [invalid: %s]", strings.Join(e.InvalidParams, ", "))
	}
	return b.String()
}

// Unwrap 让 errors.Is 能识别错误的类别。
func (e *UpstreamError) Unwrap() error {
	switch {
	case e.StatusCode == http.StatusTooManyRequests:
		return ErrQuotaExceeded
	case e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden:
		return ErrUnauthorized
	case e.StatusCode >= 400 && e.StatusCode < 500:
		return ErrUpstreamRejected
	default:
		return ErrUpstreamUnavailable
	}
}

// upstreamBodyLimit 是记录到错误里的响应体上限。
const upstreamBodyLimit = 2048

// newUpstreamError 从上游响应构造 [UpstreamError]。
func newUpstreamError(status int, body []byte) *UpstreamError {
	e := &UpstreamError{StatusCode: status}
	if len(body) > upstreamBodyLimit {
		e.Body = string(body[:upstreamBodyLimit]) + "...(truncated)"
	} else {
		e.Body = string(body)
	}

	// 上游有两代错误体，都见过：
	//
	//	新一代：{"error":{"status":400,"title":"Invalid Parameters","invalidParams":["location"]}}
	//	旧一代：{"code":"400","error":"Invalid Parameters"}
	var probe struct {
		Code  string          `json:"code"`
		Error json.RawMessage `json:"error"`
		Title string          `json:"title"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return e
	}

	e.Code = probe.Code

	// 新一代：error 是对象。
	var inner struct {
		Title         string   `json:"title"`
		InvalidParams []string `json:"invalidParams"`
	}
	if len(probe.Error) > 0 && json.Unmarshal(probe.Error, &inner) == nil {
		if inner.Title != "" {
			e.Title = inner.Title
			e.InvalidParams = inner.InvalidParams
			return e
		}
	}

	// 旧一代：error 是字符串。
	var legacy string
	if len(probe.Error) > 0 && json.Unmarshal(probe.Error, &legacy) == nil && legacy != "" {
		e.Title = legacy
		return e
	}

	if probe.Title != "" {
		e.Title = probe.Title
	}
	return e
}
