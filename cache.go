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
// **不建议用于生产环境。** 它没有后台清理：过期条目只在被 [MemoryCache.Get]
// 命中时顺手删除，没被再次访问的过期条目会一直留在 map 里，长期运行的进程
// 内存占用只增不减。
//
// 生产环境用共享缓存实现 [Cache]：独立模块 `rediscache`
// 提供 Redis 实现，也可自行实现带清理的缓存。多实例部署同样需要共享缓存，
// 否则各实例的缓存命中率互相摊薄。
//
// 它只适合测试替身与单实例的临时缓存。
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
