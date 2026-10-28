// Package main 演示 qwx 的基本用法。
//
// 运行前需要设置环境变量：
//
//	QWEATHER_API_KEY   你的 API KEY（或改用 JWT，见 README）
//	QWEATHER_API_HOST  控制台设置页里的专属 API Host（必填）
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/KanoCifer/qwx"
	"github.com/KanoCifer/qwx/geo"
	"github.com/KanoCifer/qwx/v1/weather"
	"github.com/KanoCifer/qwx/v7"
)

func main() {
	key := os.Getenv("QWEATHER_API_KEY")
	if key == "" {
		log.Fatal("请设置 QWEATHER_API_KEY")
	}
	host := os.Getenv("QWEATHER_API_HOST")
	if host == "" {
		log.Fatal("请设置 QWEATHER_API_HOST（控制台设置页里的专属 API Host）")
	}

	auth, err := qwx.NewAPIKeyAuth(key)
	if err != nil {
		log.Fatalf("认证初始化失败: %v", err)
	}

	// 生产环境建议注入 Redis 等共享缓存；开发时用内置的内存缓存即可。
	base, err := qwx.NewClient(auth, host, qwx.WithCache(qwx.NewMemoryCache()))
	if err != nil {
		log.Fatalf("客户端初始化失败: %v", err)
	}

	// 各接口族用基础客户端构造自己的服务客户端。
	w := weather.New(base)
	g := geo.New(base)
	old := v7.New(base)

	ctx := context.Background()

	// 1. 用坐标查实时天气。
	loc, err := qwx.ParseLocation("116.41,39.92")
	if err != nil {
		log.Fatalf("位置解析失败: %v", err)
	}

	current, err := w.Current(ctx, loc, weather.Options{Lang: "zh", LocalTime: true})
	if err != nil {
		log.Fatalf("实时天气查询失败: %v", err)
	}
	fmt.Printf("实时天气：%s %v%s，湿度 %.0f%%\n",
		current.Condition.Text,
		current.Temperature.Value,
		current.Temperature.Unit,
		current.Humidity*100,
	)

	// 2. 逐小时预报。
	hourly, err := w.Hourly(ctx, loc, 24, weather.Options{Lang: "zh", LocalTime: true})
	if err != nil {
		log.Fatalf("逐小时预报查询失败: %v", err)
	}
	fmt.Printf("逐小时预报 %d 条\n", len(hourly.Hours))

	// 3. 逐日预报。
	daily, err := w.Daily(ctx, loc, 3, weather.Options{Lang: "zh", LocalTime: true})
	if err != nil {
		log.Fatalf("逐日预报查询失败: %v", err)
	}
	for _, d := range daily.Days {
		date := d.ForecastStartTime
		if len(date) >= 10 {
			date = date[:10]
		}
		fmt.Printf("  %s 最高 %v%s / 最低 %v%s\n",
			date,
			d.TemperatureMax.Value, d.TemperatureMax.Unit,
			d.TemperatureMin.Value, d.TemperatureMin.Unit,
		)
	}

	// 4. 城市检索（认地名）。
	cities, err := g.LookupCity(ctx, qwx.Location{Name: "杭州"}, geo.Options{Lang: "zh"})
	if err != nil {
		log.Fatalf("城市检索失败: %v", err)
	}
	for _, city := range cities.Locations {
		fmt.Printf("城市：%s（%s）\n", city.DisplayName(), city.ID)
	}

	// 5. 钓鱼指数（旧一代接口）。
	indices, err := old.Indices(ctx, loc, v7.IndicesTypeFishing, 1, "zh")
	if err != nil {
		log.Fatalf("生活指数查询失败: %v", err)
	}
	for _, idx := range indices.Daily {
		fmt.Printf("  %s %s：%s\n", idx.Name, idx.Category, idx.Text)
	}
}
