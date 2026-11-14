package baseclient

import (
	"context"
	"strings"
	"testing"

	"github.com/KanoCifer/qwx"
)

// stubDoer 是个什么都不做的 Doer。
type stubDoer struct{}

func (stubDoer) Do(_ context.Context, _ qwx.Request, _ any) error { return nil }

func TestRequireAcceptsDoer(t *testing.T) {
	Require(stubDoer{}, "weather")
}

func TestRequirePanicsOnNil(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("nil base client 应该 panic")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic 值 = %T, want string", r)
		}
		if !strings.Contains(msg, "weather") {
			t.Errorf("panic 文案 %q 里没有包名", msg)
		}
	}()
	Require(nil, "weather")
}
