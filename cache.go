package qwx

import (
	"context"
	"sync"
	"time"
)

// Cache 是响应缓存接口，由调用方注入。
//
// 本包不绑定任何具体的缓存实现（Redis、内存、文件均可）。未注入缓存时
// 每次请求都会打到上游。
//
// 实现约定：
//
//   - Get 命中时返回 (data, true, nil)；未命中返回 (nil, false, nil)。
//   - 读写失败返回非 nil error。缓存故障不应中断业务，SDK 会降级为
//     直接请求上游，因此实现方可以放心地返回错误。
//   - Set 的 ttl 是相对时长。
type Cache interface {
	// Get 读取缓存。
	Get(ctx context.Context, key string) (data []byte, ok bool, err error)
	// Set 写入缓存，ttl 为过期时长。
	Set(ctx context.Context, key string, data []byte, ttl time.Duration) error
}

// MemoryCache 是一个进程内的 [Cache] 实现。
//
// 适合单实例部署或开发调试。它会定期清理过期条目，避免无界增长。
// 多实例部署请换成 Redis 等共享缓存，否则各实例的缓存命中率会很低。
//
// 内部有锁，可并发使用。
type MemoryCache struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
	// now 可注入，便于测试。
	now func() time.Time
}

type memoryEntry struct {
	data []byte
	exp  time.Time
}

// NewMemoryCache 构造 [MemoryCache]。
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		entries: make(map[string]memoryEntry),
		now:     time.Now,
	}
}

// Get 读取缓存。
func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[key]
	if !ok {
		return nil, false, nil
	}
	if c.now().After(e.exp) {
		delete(c.entries, key)
		return nil, false, nil
	}
	return e.data, true, nil
}

// Set 写入缓存。
func (c *MemoryCache) Set(_ context.Context, key string, data []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = memoryEntry{data: data, exp: c.now().Add(ttl)}
	return nil
}

// 编译期断言。
var _ Cache = (*MemoryCache)(nil)
