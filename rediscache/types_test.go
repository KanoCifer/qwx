package rediscache

import (
	"errors"
	"testing"
	"time"

	"github.com/KanoCifer/qwx"
)

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name string
		opts Options
		want error
	}{
		{"空地址", Options{}, qwx.ErrInvalidConfig},
		{"空白地址", Options{Addr: "  "}, qwx.ErrInvalidConfig},
		{"负数 DB", Options{Addr: "127.0.0.1:6379", DB: -1}, qwx.ErrInvalidConfig},
		{"正常", Options{Addr: "127.0.0.1:6379"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.validate()
			if tt.want == nil {
				if err != nil {
					t.Fatalf("validate = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("validate = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestNewInvalidOptions(t *testing.T) {
	if _, err := New(Options{}); !errors.Is(err, qwx.ErrInvalidConfig) {
		t.Errorf("New with empty addr = %v, want ErrInvalidConfig", err)
	}
}

func TestOptionsToRedisDefaults(t *testing.T) {
	// 留空的超时用默认值。
	r := Options{Addr: "127.0.0.1:6379"}.toRedis()
	if r.DialTimeout != DefaultDialTimeout {
		t.Errorf("DialTimeout = %s, want %s", r.DialTimeout, DefaultDialTimeout)
	}
	if r.ReadTimeout != DefaultReadTimeout {
		t.Errorf("ReadTimeout = %s, want %s", r.ReadTimeout, DefaultReadTimeout)
	}
	if r.WriteTimeout != DefaultWriteTimeout {
		t.Errorf("WriteTimeout = %s, want %s", r.WriteTimeout, DefaultWriteTimeout)
	}

	// 显式给的值要保留。
	r = Options{
		Addr:         "127.0.0.1:6379",
		DialTimeout:  time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     7,
		Username:     "u",
		Password:     "p",
		DB:           2,
	}.toRedis()
	if r.DialTimeout != time.Second || r.ReadTimeout != 2*time.Second || r.WriteTimeout != 3*time.Second {
		t.Errorf("timeouts = %s/%s/%s, want 1s/2s/3s", r.DialTimeout, r.ReadTimeout, r.WriteTimeout)
	}
	if r.PoolSize != 7 || r.Username != "u" || r.Password != "p" || r.DB != 2 {
		t.Errorf("options not carried over: %+v", r)
	}
}
