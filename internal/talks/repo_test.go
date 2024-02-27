package talks_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/testcontainers/workshop-go/internal/talks"
)

func TestNewRepository(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15.3-alpine"),
		postgres.WithInitScripts(filepath.Join("..", "..", "testdata", "dev-db.sql")), // path to the root of the project
		postgres.WithDatabase("talks-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate pgContainer: %s", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	assert.NoError(t, err)

	talksRepo, err := talks.NewRepository(ctx, connStr)
	assert.NoError(t, err)

	t.Run("Create a talk and retrieve it by UUID", func(t *testing.T) {
		uid := uuid.NewString()
		title := "Delightful Integration Tests with Testcontainers for Go"

		talk := talks.Talk{
			UUID:  uid,
			Title: title,
		}

		err = talksRepo.Create(ctx, &talk)
		assert.NoError(t, err)
		assert.Equal(t, talk.ID, 3) // the third, as there are two talks in the testdata/dev-db.sql file

		dbTalk, err := talksRepo.GetByUUID(ctx, uid)
		assert.NoError(t, err)
		assert.NotNil(t, dbTalk)
		assert.Equal(t, 3, talk.ID)
		assert.Equal(t, uid, talk.UUID)
		assert.Equal(t, title, talk.Title)
	})

	t.Run("Exists by UUID", func(t *testing.T) {
		uid := uuid.NewString()
		title := "Delightful Integration Tests with Testcontainers for Go"

		talk := talks.Talk{
			UUID:  uid,
			Title: title,
		}

		err = talksRepo.Create(ctx, &talk)
		assert.NoError(t, err)

		found := talksRepo.Exists(ctx, uid)
		assert.True(t, found)
	})

	t.Run("Does not exist by UUID", func(t *testing.T) {
		uid := uuid.NewString()

		found := talksRepo.Exists(ctx, uid)
		assert.False(t, found)
	})
}
