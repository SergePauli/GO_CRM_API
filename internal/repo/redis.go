package repo

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func NewRedis(addr string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})
	return rdb
}

func PingRedis(rdb *redis.Client) error {
	return rdb.Ping(context.Background()).Err()
}