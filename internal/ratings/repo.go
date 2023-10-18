package ratings

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

// Repository is the interface that wraps the basic operations with the Redis store.
type Repository struct {
	client *redis.Client
}

// NewRepository creates a new repository. It will receive a context and the Redis connection string.
func NewRepository(ctx context.Context, connStr string) (*Repository, error) {
	options, err := redis.ParseURL(connStr)
	if err != nil {
		return nil, err
	}

	cli := redis.NewClient(options)

	pong, err := cli.Ping(ctx).Result()
	if err != nil {
		// You probably want to retry here
		return nil, err
	}

	if pong != "PONG" {
		// You probably want to retry here
		return nil, err
	}

	return &Repository{client: cli}, nil
}

// Add increments in one the counter for the given rating value and talk UUID.
func (r *Repository) Add(ctx context.Context, rating Rating) (int64, error) {
	return r.client.HIncrBy(ctx, toKey(rating.TalkUuid), fmt.Sprintf("%d", rating.Value), 1).Result()
}

// FindAllByUUID returns all the ratings and their counters for the given talk UUID.
func (r *Repository) FindAllByUUID(ctx context.Context, uid string) map[string]string {
	return r.client.HGetAll(ctx, toKey(uid)).Val()
}

// toKey is a helper function that returns the uuid prefixed with "ratings/".
func toKey(uuid string) string {
	return "ratings/" + uuid
}
