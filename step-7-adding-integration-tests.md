# Step 7: Adding Integration Tests

Ok, you have a working application, but you don't have any tests. Let's add some integration tests to verify that the application works as expected.

## Integration tests for the Ratings store

Let's add a new file `internal/ratings/repo_test.go` with the following content:

```go
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

```

This test will start a Redis container, and it will define two tests:

* `Add rating`: it will add a rating to the store and verify that the result is the same as the one provided
* `Add multiple ratings`: it will add 100 ratings to the store and verify that the distribution of ratings is correct

The package has been named with the `_test` suffix to indicate that it contains tests. This is a convention in Go and forces you to consume your code as a package, which is a good practice.

Now run `go mod tidy` from the root of the project to download the Go dependencies, as the workshop is using [testify](https://github.com/stretchr/testify) as the assertions library.

Finally, run your tests with `go test -v -count=1 ./internal/ratings` from the root of the project. You should see the following output:

```text
=== RUN   TestNewRepository
2023/10/19 17:24:24 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 78+testcontainerscloud (via Testcontainers Desktop 1.4.18)
  API Version: 1.43
  Operating System: Ubuntu 20.04 LTS
  Total Memory: 7407 MB
  Resolved Docker Host: tcp://127.0.0.1:62250
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: fbe36666d79be2fb28c514488a02c767da8b05033f4f9eb09c5863a6efa10f53
  Test ProcessID: 7c61e4fe-a9ae-44e3-a5b0-58fc46d28624
2023/10/19 17:24:24 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/19 17:24:24 ✅ Container created: b6dd4c4fff07
2023/10/19 17:24:24 🐳 Starting container: b6dd4c4fff07
2023/10/19 17:24:24 ✅ Container started: b6dd4c4fff07
2023/10/19 17:24:24 🚧 Waiting for container id b6dd4c4fff07 image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/19 17:24:25 🐳 Creating container for image docker.io/redis:7
2023/10/19 17:24:25 ✅ Container created: 5ee43720e9d3
2023/10/19 17:24:25 🐳 Starting container: 5ee43720e9d3
2023/10/19 17:24:25 ✅ Container started: 5ee43720e9d3
2023/10/19 17:24:25 🚧 Waiting for container id 5ee43720e9d3 image: docker.io/redis:7. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
=== RUN   TestNewRepository/Add_rating
=== RUN   TestNewRepository/Add_multiple_ratings
2023/10/19 17:24:25 🐳 Terminating container: 5ee43720e9d3
2023/10/19 17:24:26 🚫 Container terminated: 5ee43720e9d3
--- PASS: TestNewRepository (1.51s)
    --- PASS: TestNewRepository/Add_rating (0.01s)
    --- PASS: TestNewRepository/Add_multiple_ratings (0.52s)
PASS
ok      github.com/testcontainers/workshop-go/internal/ratings  1.844s
```

_NOTE: if you experiment longer test execution times it could caused by the need of pulling the images from the registry._

## Integration tests for the Streaming queue

Let's add a new file `internal/streams/broker_test.go` with the following content:

```go
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

```

This test will start a Redpanda container, and it will define two tests:

* `Send Rating without callback`: it will send a rating to the broker and verify that the result does not return an error after the callback is executed.
* `Send Rating with error in callback`: it will send a rating to the broker and verify that the result returns an error after the callback is executed.

Please notice that the package has been named with the `_test` suffix for the same reasons describe above.

There is no need to run `go mod tidy` again, as you have already downloaded the Go dependencies.

Finally, run your tests with `go test -v -count=1 ./internal/streams` from the root of the project. You should see the following output:

```text
=== RUN   TestBroker
2023/10/19 17:29:54 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 78+testcontainerscloud (via Testcontainers Desktop 1.4.18)
  API Version: 1.43
  Operating System: Ubuntu 20.04 LTS
  Total Memory: 7407 MB
  Resolved Docker Host: tcp://127.0.0.1:62250
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: b7cbe505ff02d97f5b7a59b3acd71206c68092791a7e48b07fc6b4a098a56e6c
  Test ProcessID: 652afab7-b384-450a-bebe-5ddc20444e7b
2023/10/19 17:29:54 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/19 17:29:54 ✅ Container created: 4b7aa6d82a38
2023/10/19 17:29:54 🐳 Starting container: 4b7aa6d82a38
2023/10/19 17:29:55 ✅ Container started: 4b7aa6d82a38
2023/10/19 17:29:55 🚧 Waiting for container id 4b7aa6d82a38 image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/19 17:29:55 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v23.1.7
2023/10/19 17:29:55 ✅ Container created: e0b8e5a71936
2023/10/19 17:29:55 🐳 Starting container: e0b8e5a71936
2023/10/19 17:29:55 ✅ Container started: e0b8e5a71936
=== RUN   TestBroker/Send_Rating_without_callback
=== RUN   TestBroker/Send_Rating_with_error_in_callback
--- PASS: TestBroker (2.07s)
    --- PASS: TestBroker/Send_Rating_without_callback (0.46s)
    --- PASS: TestBroker/Send_Rating_with_error_in_callback (0.01s)
PASS
ok      github.com/testcontainers/workshop-go/internal/streams  2.277s
```

_NOTE: if you experiment longer test execution times it could caused by the need of pulling the images from the registry._

## Integration tests for the Talks store

Let's add a new file `internal/talks/repo_test.go` with the following content:

```go
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

```

This test will start a Postgres container, and it will define three tests:

* `Create a talk and retrieve it by UUID`: it will create a talk in the store and verify that the result is the same as the one provided.
* `Exists by UUID`: it will create a talk in the store and verify that the talk exists.
* `Does not exist by UUID`: it will verify that a talk does not exist in the store.

Please notice that the package has been named with the `_test` suffix for the same reasons describe above.

There is no need to run `go mod tidy` again, as you have already downloaded the Go dependencies.

Finally, run your tests with `go test -v -count=1 ./internal/talks` from the root of the project. You should see the following output:

```text
=== RUN   TestNewRepository
2023/10/19 17:39:09 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 78+testcontainerscloud (via Testcontainers Desktop 1.4.18)
  API Version: 1.43
  Operating System: Ubuntu 20.04 LTS
  Total Memory: 7407 MB
  Resolved Docker Host: tcp://127.0.0.1:62250
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: b7be60cb63f0433cdafd50b8fe144c791959fd2aa586f8abaefa0661049a1fc8
  Test ProcessID: a5391e34-0c1c-4df5-b55d-4506c925edb8
2023/10/19 17:39:09 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/19 17:39:10 ✅ Container created: bf72ea10b512
2023/10/19 17:39:10 🐳 Starting container: bf72ea10b512
2023/10/19 17:39:10 ✅ Container started: bf72ea10b512
2023/10/19 17:39:10 🚧 Waiting for container id bf72ea10b512 image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/19 17:39:10 🐳 Creating container for image postgres:15.3-alpine
2023/10/19 17:39:10 ✅ Container created: 639b7afc1b51
2023/10/19 17:39:10 🐳 Starting container: 639b7afc1b51
2023/10/19 17:39:10 ✅ Container started: 639b7afc1b51
2023/10/19 17:39:10 🚧 Waiting for container id 639b7afc1b51 image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x140003f79f0 Strategies:[0x14000423bf0]}
=== RUN   TestNewRepository/Create_a_talk_and_retrieve_it_by_UUID
=== RUN   TestNewRepository/Exists_by_UUID
=== RUN   TestNewRepository/Does_not_exist_by_UUID
2023/10/19 17:39:12 🐳 Terminating container: 639b7afc1b51
2023/10/19 17:39:12 🚫 Container terminated: 639b7afc1b51
--- PASS: TestNewRepository (2.67s)
    --- PASS: TestNewRepository/Create_a_talk_and_retrieve_it_by_UUID (0.03s)
    --- PASS: TestNewRepository/Exists_by_UUID (0.01s)
    --- PASS: TestNewRepository/Does_not_exist_by_UUID (0.01s)
PASS
ok      github.com/testcontainers/workshop-go/internal/talks    2.995s
```

_NOTE: if you experiment longer test execution times it could caused by the need of pulling the images from the registry._

We have now added integration tests for the three stores of our application. Let's add some integration tests for the API.

### 
[Next](step-8-integration-tests-for-api.md)