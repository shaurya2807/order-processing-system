package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const ttl = 5 * time.Minute

type Cache struct {
	client *redis.Client
}

func New(host, port string) *Cache {
	return &Cache{
		client: redis.NewClient(&redis.Options{
			Addr: host + ":" + port,
		}),
	}
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (c *Cache) Set(ctx context.Context, key, value string) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
