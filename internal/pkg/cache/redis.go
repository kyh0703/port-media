package cache

import (
	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(cfg *configs.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}
