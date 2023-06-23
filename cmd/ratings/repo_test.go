package ratings_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/workshop-go/ratings"
)

func TestNewRepository(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := tcRedis.RunContainer(ctx, testcontainers.WithImage("docker.io/redis:7"))
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	connStr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	repo, err := ratings.NewRepository(ctx, connStr)
	require.NoError(t, err)
	assert.NotNil(t, repo)

	t.Run("Add rating", func(t *testing.T) {
		rating := ratings.Rating{
			TalkUuid: "uuid12345",
			Value:    5,
		}

		repo.Add(ctx, rating)

		result := repo.Get(ctx, rating.TalkUuid)
		assert.Equal(t, "5", result)
	})

	t.Run("Add multiple ratings", func(t *testing.T) {
		var incr int64 = 2
		rating := ratings.Rating{
			TalkUuid: "uuid67890",
			Value:    incr,
		}

		max := 100
		for i := 0; i < max; i++ {
			repo.Add(ctx, rating)
		}

		result := repo.Get(ctx, rating.TalkUuid)
		assert.Equal(t, fmt.Sprintf("%d", incr*int64(max)), result)

		values := repo.FindAllByByUUID(ctx, rating.TalkUuid)
		assert.Len(t, values, max)
	})
}
