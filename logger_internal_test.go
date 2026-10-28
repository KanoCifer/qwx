package qwx

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestNewClientDefaultLoggerIsSilent(t *testing.T) {
	// 默认不注入 logger：一个字节都不该输出。
	c, err := NewClient(&APIKeyAuth{key: "k"}, "abcxyz.qweatherapi.com")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.logger == nil {
		t.Fatal("logger should not be nil")
	}
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		if c.logger.Enabled(context.Background(), level) {
			t.Errorf("default logger should not enable %v", level)
		}
	}
}

func TestWithLoggerNilKeepsDefault(t *testing.T) {
	c, err := NewClient(&APIKeyAuth{key: "k"}, "abcxyz.qweatherapi.com", WithLogger(nil))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if c.logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("logger should stay silent")
	}
}

func TestWithLoggerReplacesDefault(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	c, err := NewClient(&APIKeyAuth{key: "k"}, "abcxyz.qweatherapi.com", WithLogger(logger))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if !c.logger.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("injected logger should be enabled")
	}
}

func TestRequestMethodDefaultsToGet(t *testing.T) {
	if got := requestMethod(Request{}); got != "GET" {
		t.Errorf("requestMethod = %q, want GET", got)
	}
	if got := requestMethod(Request{Method: "POST"}); got != "POST" {
		t.Errorf("requestMethod = %q, want POST", got)
	}
}

func TestResponseStatusHandlesNilResponse(t *testing.T) {
	if got := responseStatus(nil); got != 0 {
		t.Errorf("responseStatus = %d, want 0", got)
	}
}
