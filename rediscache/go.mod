module github.com/KanoCifer/qwx/rediscache

go 1.24

require (
	github.com/KanoCifer/qwx v0.8.0
	github.com/alicebob/miniredis/v2 v2.40.0
	github.com/redis/go-redis/v9 v9.22.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

// 本地开发时指向仓库根模块，不依赖已发布版本。
// replace 只在主模块生效：消费者构建本模块时该行被忽略，按上面的 require 解析。
replace github.com/KanoCifer/qwx => ../
