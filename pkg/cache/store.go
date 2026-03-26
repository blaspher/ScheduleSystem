package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultCacheTTL = 2 * time.Minute

var ErrCacheDecode = errors.New("cache decode failed")

type Store struct {
	client *redis.Client
	ttl    time.Duration
}

func NewStore(client *redis.Client, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	return &Store{
		client: client,
		ttl:    ttl,
	}
}

func (s *Store) GetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	raw, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return false, fmt.Errorf("%w: %v", ErrCacheDecode, err)
	}
	return true, nil
}

func (s *Store) SetJSON(ctx context.Context, key string, val interface{}) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, data, s.ttl).Err()
}

func (s *Store) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return s.client.Del(ctx, keys...).Err()
}

func (s *Store) DeleteByPattern(ctx context.Context, pattern string) error {
	// MVP/demo-scale strategy: SCAN + DEL for pattern invalidation.
	// This is intentionally simple and not intended as a high-scale final strategy.
	iter := s.client.Scan(ctx, 0, pattern, 100).Iterator()
	keys := make([]string, 0, 100)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		if len(keys) >= 100 {
			if err := s.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
			keys = keys[:0]
		}
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		if err := s.client.Del(ctx, keys...).Err(); err != nil {
			return err
		}
	}
	return nil
}
