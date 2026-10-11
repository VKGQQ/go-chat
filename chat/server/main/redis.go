package main

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/logging"
)

var redisClient *redis.Client

// 启动阶段等待 Redis 就绪的重试参数
const (
	redisMaxRetries  = 5               // Ping 最大尝试次数
	redisRetryGap    = 2 * time.Second // 两次尝试之间的等待时间
	redisPingTimeout = 3 * time.Second // 单次 Ping 的超时时间
)

func initRedis(address string, password string, db int, poolSize int) error {
	// 关闭 go-redis 内部日志：连接失败时它会按自己的策略反复重试并刷屏，
	// 错误统一由我们的外层重试逻辑记录，避免两层重试日志叠加。
	logging.Disable()
	redisClient = redis.NewClient(&redis.Options{
		Addr:            address,
		Password:        password,
		DB:              db,
		PoolSize:        poolSize, // 连接池最大连接数
		MinIdleConns:    3,        // 最小空闲连接数
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		ConnMaxIdleTime: 5 * time.Minute,
	})

	// 容忍 Redis 比应用晚就绪的部署时序（如容器编排中依赖先启动）：
	// 最多重试 redisMaxRetries 次，每次间隔 redisRetryGap；
	// 全部失败则返回错误，由 main 决定 fail fast 退出。
	var lastErr error
	for i := 1; i <= redisMaxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), redisPingTimeout)
		err := redisClient.Ping(ctx).Err()
		cancel()
		if err == nil {
			fmt.Println("redis 连接成功")
			return nil
		}
		lastErr = err
		if i < redisMaxRetries {
			fmt.Printf("redis 连接失败（第 %d/%d 次），%v 后重试: %v\n",
				i, redisMaxRetries, redisRetryGap, err)
			time.Sleep(redisRetryGap)
		}
	}
	return fmt.Errorf("redis 连续 %d 次连接失败: %w", redisMaxRetries, lastErr)
}
