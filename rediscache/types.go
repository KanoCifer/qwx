package rediscache

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/redis/go-redis/v9"
)

// Redis 客户端的默认超时。
const (
	// DefaultDialTimeout 是建立连接的超时。
	DefaultDialTimeout = 5 * time.Second
	// DefaultReadTimeout 是读操作的超时。
	DefaultReadTimeout = 3 * time.Second
	// DefaultWriteTimeout 是写操作的超时。
	DefaultWriteTimeout = 3 * time.Second
)

// Options 是 Redis 客户端的配置。
//
// 零值不可用，Addr 必填。其余字段留空时用默认值。
//
// 调用方 ctx 上的截止时间与取消会传到 Redis 操作：ctx 先到期就先返回，
// 不必等满 ReadTimeout / WriteTimeout。
type Options struct {
	// Addr 是 Redis 地址，形如 "127.0.0.1:6379"。必填。
	Addr string
	// Username 是 ACL 用户名（Redis 6+）。留空用默认用户。
	Username string
	// Password 是认证密码。留空表示不认证。
	Password string
	// DB 是数据库编号，默认 0。
	DB int
	// TLSConfig 是 TLS 配置。非 nil 时走 TLS 连接。
	//
	// 云托管的 Redis 常强制 TLS，这时填一个带 ServerName 的配置。
	TLSConfig *tls.Config
	// DialTimeout 是建立连接的超时。小于等于 0 时用 [DefaultDialTimeout]。
	DialTimeout time.Duration
	// ReadTimeout 是读操作的超时。小于等于 0 时用 [DefaultReadTimeout]。
	ReadTimeout time.Duration
	// WriteTimeout 是写操作的超时。小于等于 0 时用 [DefaultWriteTimeout]。
	WriteTimeout time.Duration
	// PoolSize 是连接池大小。0 时用 go-redis 的默认值，负数无效 ——
	// [New] 返回 [qwx.ErrInvalidConfig]。
	PoolSize int
	// KeyPrefix 是键前缀，原样拼在 SDK 生成的键前面。
	//
	// 留空时键就是 SDK 生成的样子。多个应用共用一个 Redis 实例时用它隔离，
	// 前缀自取，比如 "myapp:"。
	KeyPrefix string
}

// validate 校验配置。
func (o Options) validate() error {
	if strings.TrimSpace(o.Addr) == "" {
		return fmt.Errorf("%w: redis addr is required", qwx.ErrInvalidConfig)
	}
	if o.DB < 0 {
		return fmt.Errorf("%w: redis db must not be negative", qwx.ErrInvalidConfig)
	}
	if o.PoolSize < 0 {
		return fmt.Errorf("%w: redis pool size must not be negative", qwx.ErrInvalidConfig)
	}
	return nil
}

// toRedis 转成 go-redis 的配置。
func (o Options) toRedis() *redis.Options {
	dial := o.DialTimeout
	if dial <= 0 {
		dial = DefaultDialTimeout
	}
	read := o.ReadTimeout
	if read <= 0 {
		read = DefaultReadTimeout
	}
	write := o.WriteTimeout
	if write <= 0 {
		write = DefaultWriteTimeout
	}
	return &redis.Options{
		Addr:         o.Addr,
		Username:     o.Username,
		Password:     o.Password,
		DB:           o.DB,
		TLSConfig:    o.TLSConfig,
		DialTimeout:  dial,
		ReadTimeout:  read,
		WriteTimeout: write,
		PoolSize:     o.PoolSize,
		// 让调用方 ctx 的截止时间与取消参与 Redis I/O：ctx 先到期就先返回，
		// 不必等满 ReadTimeout / WriteTimeout。
		ContextTimeoutEnabled: true,
	}
}
