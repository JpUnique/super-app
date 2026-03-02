package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Store interface {
	AddRequest(ctx context.Context, key string, window time.Duration) (int, error)
}

type redisStore struct {
	client *redis.Client
}

func NewRedisStore(redisURL string) Store {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic(fmt.Sprintf("invalid redis url: %v", err))
	}

	return &redisStore{
		client: redis.NewClient(opt),
	}
}

func (s *redisStore) AddRequest(ctx context.Context, key string, window time.Duration) (int, error) {
	now := time.Now().UnixNano()

	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: now,
	})
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", now-int64(window.Nanoseconds())))
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, ErrRedisFailure
	}

	return int(countCmd.Val()), nil
}
