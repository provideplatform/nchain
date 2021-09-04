package consumer

import (
	"time"

	"github.com/kthomas/go-redisutil"
)

// IDatabase abstracts out the storage for Fragments
type IDatabase interface {
	Get(key string) (*string, error)
	Increment(key string) (*int64, error)
	Set(key string, val interface{}, ttl *time.Duration) error
	Setup()
}

// Redis wrapper for IDatabase
type Redis struct {
}

// Get returns the value for the given key
func (r *Redis) Get(key string) (*string, error) {
	return redisutil.Get(key)
}

// Increment atomically increments the given key's value
func (r *Redis) Increment(key string) (*int64, error) {
	return redisutil.Increment(key)
}

// Set stores the value to the given key with an optional time to live (ttl)
func (r *Redis) Set(key string, val interface{}, ttl *time.Duration) error {
	return redisutil.Set(key, val, ttl)
}

// Setup will initialise the Redis connection
func (r *Redis) Setup() {
	redisutil.RequireRedis()
}
