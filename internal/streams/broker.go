package streams

import (
	"context"

	"github.com/testcontainers/workshop-go/internal/ratings"
	"github.com/twmb/franz-go/pkg/kgo"
)

const RatingsTopic = "ratings"

// Repository is the interface that wraps the basic operations with the broker store.
type Repository struct {
	client *kgo.Client
}

// NewStream creates a new repository. It will receive a context and the connection string for the broker.
func NewStream(ctx context.Context, connStr string) (*Repository, error) {
	cli, err := kgo.NewClient(
		kgo.SeedBrokers(connStr),
		kgo.ConsumeTopics(RatingsTopic),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		// You probably want to retry here
		return nil, err
	}

	return &Repository{client: cli}, nil
}

// SendRating sends a rating to the broker in an asynchronous way, executing a callback
// when the record is produced. It will notifiy the caller if the operation errored or
// if the context was cancelled.
func (r *Repository) SendRating(ctx context.Context, rating ratings.Rating, produceCallback func() error) error {
	record := &kgo.Record{Topic: RatingsTopic, Value: []byte("test")}

	errChan := make(chan error, 1)

	r.client.Produce(ctx, record, func(producedRecord *kgo.Record, err error) {
		if err != nil {
			errChan <- err
			return
		}

		err = produceCallback()
		if err != nil {
			errChan <- err
			return
		}

		errChan <- nil
	})

	// we are actively waiting for an error to be returned or for the context to be cancelled,
	// because we want to notify the caller in those cases
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
