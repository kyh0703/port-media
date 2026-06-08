package cache

import (
	"fmt"

	"github.com/kyh0703/portfoilo-media/configs"
	"github.com/redis/go-redis/v9"
)

func NewRedisClient(cfg *configs.Config) (*redis.Client, error) {
	options, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return redis.NewClient(options), nil
}
