package quick_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/quick"
)

// TestEndpointErrorUnwrap 失败原因能按类别识别，错误文本能定位到具体端点。
func TestEndpointErrorUnwrap(t *testing.T) {
	err := quick.EndpointError{
		Part:     quick.PartAlerts,
		Endpoint: qwx.Endpoint("weatheralert/v1/current"),
		Err:      qwx.ErrUnauthorized,
	}
	if !errors.Is(err, qwx.ErrUnauthorized) {
		t.Error("errors.Is 未识别出 ErrUnauthorized")
	}
	text := err.Error()
	if !strings.Contains(text, string(quick.PartAlerts)) {
		t.Errorf("错误文本 %q 里没有数据块 %q", text, quick.PartAlerts)
	}
	if !strings.Contains(text, "weatheralert/v1/current") {
		t.Errorf("错误文本 %q 里没有端点标识", text)
	}
}
