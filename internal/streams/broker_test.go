package streams_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	"github.com/testcontainers/workshop-go/internal/ratings"
	"github.com/testcontainers/workshop-go/internal/streams"
)

func TestBroker(t *testing.T) {
	ctx := context.Background()

	redpandaC, err := redpanda.RunContainer(
		ctx,
		testcontainers.WithImage("docker.redpanda.com/redpandadata/redpanda:v23.1.7"),
		redpanda.WithAutoCreateTopics(),
	)
	if err != nil {
		t.Fatal(err)
	}

	seedBroker, err := redpandaC.KafkaSeedBroker(ctx)
	require.NoError(t, err)

	repo, err := streams.NewStream(ctx, seedBroker)
	require.NoError(t, err)

	err = repo.SendRating(ctx, ratings.Rating{
		TalkUuid: "uuid12345",
		Value:    5,
	})
	require.NoError(t, err)
}
