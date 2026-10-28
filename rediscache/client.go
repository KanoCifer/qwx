// Package rediscache 提供基于 Redis 的 qwx.Cache 实现。
//
// 它是独立模块（自带 go.mod，依赖 go-redis），根模块保持零第三方依赖。
// go-redis v9.23.0 起要求 Go 1.26，本模块的最低 Go 版本随之是 1.26。
// 需要 Redis 缓存时单独引入：
//
//	go get github.com/KanoCifer/qwx/rediscache
//
// 用法：
//
//	cache, err := rediscache.New(rediscache.Options{Addr: "127.0.0.1:6379"})
//	if err != nil {
//		return err
//	}
//	defer cache.Close()
//
//	base, err := qwx.NewClient(auth, "abcxyz.qweatherapi.com", qwx.WithCache(cache))
//
// 缓存键由 SDK 生成，本包只加可选前缀，不改键的形状。
package rediscache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KanoCifer/qwx"
	"github.com/redis/go-redis/v9"
)

// Cache 是基于 Redis 的 [qwx.Cache] 实现。
//
// 由 [New] 构造，内部有连接池，可并发共享。
type Cache struct {
	client *redis.Client
	prefix string
}

// New 构造 [Cache]。
//
// opts.Addr 必填，其余字段留空用默认值。构造时不连 Redis —— 连接在第一次
// 操作时建立，地址写错也要等 [Cache.Ping] 才暴露。
//
// 返回的 [Cache] 要用完 [Cache.Close] 关闭，释放连接池。
func New(opts Options) (*Cache, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}
	return &Cache{
		client: redis.NewClient(opts.toRedis()),
		prefix: opts.KeyPrefix,
	}, nil
}

// Get 读取缓存。
//
// 命中返回 (data, true, nil)，未命中返回 (nil, false, nil)。
// 读写失败返回非 nil error —— SDK 会吞掉缓存错误并降级为直连上游。
func (c *Cache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("rediscache: get %q: %w", key, err)
	}
	return data, true, nil
}

// Set 写入缓存，ttl 是相对时长。
//
// ttl 小于等于 0 返回 [qwx.ErrInvalidParameter] 且不写入：缓存条目不带
// 过期时间会一直留着，拿到的天气数据会一直是旧的。
func (c *Cache) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("%w: cache ttl must be positive, got %s", qwx.ErrInvalidParameter, ttl)
	}
	if err := c.client.Set(ctx, c.key(key), data, ttl).Err(); err != nil {
		return fmt.Errorf("rediscache: set %q: %w", key, err)
	}
	return nil
}

// Ping 检查 Redis 是否可用，用于健康检查与启动自检。
func (c *Cache) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("rediscache: ping: %w", err)
	}
	return nil
}

// Close 关闭连接池。
//
// 关闭后再调用读写方法会返回错误。Close 可重复调用，重复调用返回 nil。
func (c *Cache) Close() error {
	if err := c.client.Close(); err != nil && !errors.Is(err, redis.ErrClosed) {
		return err
	}
	return nil
}

// key 给 SDK 生成的键加前缀。
func (c *Cache) key(key string) string {
	return c.prefix + key
}

// 编译期断言。
var _ qwx.Cache = (*Cache)(nil)
