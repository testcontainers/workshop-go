package ratings_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/workshop-go/internal/ratings"
)

func TestNewRepository(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := tcRedis.RunContainer(ctx, testcontainers.WithImage("docker.io/redis:6-alpine"))
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

		result, err := repo.Add(ctx, rating)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), result)
	})

	t.Run("Add multiple ratings", func(t *testing.T) {
		takUUID := "uuid67890"
		max := 100
		distribution := 5

		for i := 0; i < max; i++ {
			rating := ratings.Rating{
				TalkUuid: takUUID,
				Value:    int64(i % distribution), // creates a distribution of ratings, 20 of each
			}

			// don't care about the result
			_, _ = repo.Add(ctx, rating)
		}

		values := repo.FindAllByUUID(ctx, takUUID)
		assert.Len(t, values, distribution)

		for i := 0; i < distribution; i++ {
			assert.Equal(t, fmt.Sprintf("%d", (max/distribution)), values[fmt.Sprintf("%d", i)])
		}
	})
}
