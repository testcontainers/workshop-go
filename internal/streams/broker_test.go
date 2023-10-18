package streams_test

import (
	"context"
	"errors"
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

	t.Run("Send Rating without callback", func(t *testing.T) {
		noopFn := func() error { return nil }

		err = repo.SendRating(ctx, ratings.Rating{
			TalkUuid: "uuid12345",
			Value:    5,
		}, noopFn)
		require.NoError(t, err)
	})

	t.Run("Send Rating with error in callback", func(t *testing.T) {
		var ErrInCallback error = errors.New("error in callback")

		errorFn := func() error { return ErrInCallback }

		err = repo.SendRating(ctx, ratings.Rating{
			TalkUuid: "uuid12345",
			Value:    5,
		}, errorFn)
		require.ErrorIs(t, ErrInCallback, err)
	})
}
