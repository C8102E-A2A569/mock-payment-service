package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	balanceTTL     time.Duration
	idempotencyTTL time.Duration
	client         *redis.Client
}

func NewRedisCache(addr, password string, balanceTTL, idempotencyTTL time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisCache{
		client:         client,
		balanceTTL:     balanceTTL,
		idempotencyTTL: idempotencyTTL,
	}, nil
}

func (c *RedisCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *RedisCache) GetBalance(ctx context.Context, accountID uuid.UUID) (balance int64, ok bool, err error) {
	key := "balance:" + accountID.String()
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, false, nil
		}
		return 0, false, err
	}
	b, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return b, true, nil
}

func (c *RedisCache) SetBalance(ctx context.Context, accountID uuid.UUID, balance int64) error {
	key := "balance:" + accountID.String()
	return c.client.Set(ctx, key, balance, c.balanceTTL).Err()
}

func (c *RedisCache) InvalidateBalance(ctx context.Context, accountID uuid.UUID) error {
	key := "balance:" + accountID.String()
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) GetIdempotency(ctx context.Context, prefix, idemKey string) ([]byte, bool, error) {
	if idemKey == "" {
		return nil, false, nil
	}
	k := "idem:" + prefix + ":" + idemKey
	val, err := c.client.Get(ctx, k).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}
	return []byte(val), true, nil
}

func (c *RedisCache) SetIdempotency(ctx context.Context, prefix, idemKey string, value []byte) error {
	if idemKey == "" {
		return nil
	}
	k := "idem:" + prefix + ":" + idemKey
	return c.client.Set(ctx, k, value, c.idempotencyTTL).Err()
}
