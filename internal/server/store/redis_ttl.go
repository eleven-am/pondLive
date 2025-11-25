//go:build redis

package store

import (
	context "context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/eleven-am/pondlive/go/internal/session"
)

// RedisTTLStore uses a Redis sorted set to track session expirations.
type RedisTTLStore struct {
	client *redis.Client
	key    string
}

// NewRedisTTLStore constructs a TTL store backed by Redis. The key parameter identifies the sorted-set name.
func NewRedisTTLStore(client *redis.Client, key string) *RedisTTLStore {
	if key == "" {
		key = "live:sessions"
	}
	return &RedisTTLStore{client: client, key: key}
}

func (s *RedisTTLStore) Touch(id session.SessionID, ttl time.Duration) error {
	ctx := context.Background()
	if ttl <= 0 {
		return s.client.ZRem(ctx, s.key, string(id)).Err()
	}
	expires := time.Now().Add(ttl).UnixMilli()
	z := &redis.Z{Score: float64(expires), Member: string(id)}
	return s.client.ZAdd(ctx, s.key, *z).Err()
}

func (s *RedisTTLStore) Remove(id session.SessionID) error {
	return s.client.ZRem(context.Background(), s.key, string(id)).Err()
}

func (s *RedisTTLStore) Expired(now time.Time) ([]session.SessionID, error) {
	ctx := context.Background()
	max := fmt.Sprintf("%d", now.UnixMilli())
	members, err := s.client.ZRangeByScore(ctx, s.key, &redis.ZRangeBy{Min: "-inf", Max: max}).Result()
	if err != nil {
		return nil, err
	}
	if len(members) > 0 {
		args := make([]interface{}, len(members))
		for i, m := range members {
			args[i] = m
		}
		if err := s.client.ZRem(ctx, s.key, args...).Err(); err != nil {
			return nil, err
		}
	}
	ids := make([]session.SessionID, len(members))
	for i, m := range members {
		ids[i] = session.SessionID(m)
	}
	return ids, nil
}
