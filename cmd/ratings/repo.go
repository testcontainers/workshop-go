package ratings

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// Repository is the interface that wraps the basic operations with the Redis store.
type Repository struct {
	client *redis.Client
}

// NewRepository creates a new repository. It will receive a context and the Redis connection string.
func NewRepository(ctx context.Context, connStr string) (*Repository, error) {
	// You will likely want to wrap your Redis package of choice in an
	// interface to aid in unit testing and limit lock-in throughtout your
	// codebase but that's out of scope for this example
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

// Add adds a new rating for a talk identified by its UUID to the Redis store.
func (r *Repository) Add(ctx context.Context, rating Rating) {
	_ = r.client.IncrBy(ctx, toKey(rating.TalkUuid), rating.Value).Val()
}

// Get retrieves a rating for a talk identified by its UUID from the Redis store.
func (r *Repository) Get(ctx context.Context, uid string) string {
	return r.client.Get(ctx, toKey(uid)).Val()
}

// toKey is a helper function that returns the uuid prefixed with "ratings/".
func toKey(uuid string) string {
	return "ratings/" + uuid
}
