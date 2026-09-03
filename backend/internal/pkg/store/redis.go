package store

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	c *redis.Client
}

func NewRedis(addr, password string) *Redis {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	return &Redis{c: rdb}
}

func (r *Redis) Client() *redis.Client { return r.c }

func (r *Redis) GetString(ctx context.Context, key string) (string, error) {
	v, err := r.c.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

func (r *Redis) GetInt64(ctx context.Context, key string) (int64, bool, error) {
	v, err := r.GetString(ctx, key)
	if err != nil {
		return 0, false, err
	}
	n, pErr := strconv.ParseInt(v, 10, 64)
	return n, pErr == nil, pErr
}

func (r *Redis) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.c.Set(ctx, key, value, ttl).Err()
}

func (r *Redis) SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	return r.c.SetNX(ctx, key, value, ttl).Result()
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.c.Del(ctx, keys...).Err()
}

func (r *Redis) Keys(ctx context.Context, pattern string) ([]string, error) {
	var cursor uint64
	var keys []string
	for {
		var ks []string
		var err error
		ks, cursor, err = r.c.Scan(ctx, cursor, pattern, 0).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, ks...)
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func (r *Redis) Close() error {
	if err := r.c.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("redis ping: %w", err)
	}
	return r.c.Close()
}
