package consumer

import (
	"time"

	"github.com/kthomas/go-redisutil"
)

type IDatabase interface {
	Get(key string) (*string, error)
	Increment(key string) (*int64, error)
	Set(key string, val interface{}, ttl *time.Duration) error
	Setup()
}

type Redis struct {
}

func (r *Redis) Get(key string) (*string, error) {
	return redisutil.Get(key)
}

func (r *Redis) Increment(key string) (*int64, error) {
	return redisutil.Increment(key)
}

func (r *Redis) Set(key string, val interface{}, ttl *time.Duration) error {
	return redisutil.Set(key, val, ttl)
}

func (r *Redis) Setup() {
	redisutil.RequireRedis()
}
