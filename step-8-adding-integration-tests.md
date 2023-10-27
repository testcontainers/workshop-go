# Step 8: Adding Integration Tests

Ok, we have a working application, but we don't have any tests. Let's add some integration tests to verify that the application works as expected.

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

The package has been named with the `_test` suffix to indicate that it contains tests. This is a convention in Go and forces us to consume your code as a package, which is a good practice.

Now run `go mod tidy` from the root of the project to download the Go dependencies, as the workshop is using [testify](https://github.com/stretchr/testify) as the assertions library.

Finally, run your tests with `go test -v -count=1 ./internal/ratings -run TestNewRepository` from the root of the project. We should see the following output:

```text
=== RUN   TestNewRepository
=== RUN   TestNewRepository
2023/10/26 15:34:04 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 23.0.6 (via Testcontainers Desktop 1.5.0)
  API Version: 1.42
  Operating System: Alpine Linux v3.18
  Total Memory: 5256 MB
  Resolved Docker Host: tcp://127.0.0.1:62516
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 8a48163f15565f205b07aa6020b119ec9c37eea28fd3bfebdda79746d7a4e35c
  Test ProcessID: 233b242a-1da4-4135-8dc4-d64c74b12169
2023/10/26 15:34:04 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/26 15:34:04 ✅ Container created: 57807689ca9a
2023/10/26 15:34:04 🐳 Starting container: 57807689ca9a
2023/10/26 15:34:04 ✅ Container started: 57807689ca9a
2023/10/26 15:34:04 🚧 Waiting for container id 57807689ca9a image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/26 15:34:04 🐳 Creating container for image docker.io/redis:7
2023/10/26 15:34:04 ✅ Container created: d831506102ae
2023/10/26 15:34:04 🐳 Starting container: d831506102ae
2023/10/26 15:34:04 ✅ Container started: d831506102ae
2023/10/26 15:34:04 🚧 Waiting for container id d831506102ae image: docker.io/redis:7. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
=== RUN   TestNewRepository/Add_rating
=== RUN   TestNewRepository/Add_multiple_ratings
2023/10/26 15:34:04 🐳 Terminating container: d831506102ae
2023/10/26 15:34:04 🚫 Container terminated: d831506102ae
--- PASS: TestNewRepository (0.75s)
    --- PASS: TestNewRepository/Add_rating (0.00s)
    --- PASS: TestNewRepository/Add_multiple_ratings (0.04s)
PASS
ok      github.com/testcontainers/workshop-go/internal/ratings  0.915s
```

_NOTE: if we experiment longer test execution times it could be caused by the need of pulling the images from the registry._

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

There is no need to run `go mod tidy` again, as we have already downloaded the Go dependencies.

Finally, run your tests with `go test -v -count=1 ./internal/streams -run TestBroker` from the root of the project. We should see the following output:

```text
=== RUN   TestBroker
2023/10/26 15:35:50 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 23.0.6 (via Testcontainers Desktop 1.5.0)
  API Version: 1.42
  Operating System: Alpine Linux v3.18
  Total Memory: 5256 MB
  Resolved Docker Host: tcp://127.0.0.1:62516
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 38e98e183213936ff72705d5df8e99537879dffcc5361a7062d14dd1f250b6b8
  Test ProcessID: d31a09a5-50df-4723-bfa6-b11f6f08e323
2023/10/26 15:35:50 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/26 15:35:50 ✅ Container created: 06e23826a3e6
2023/10/26 15:35:50 🐳 Starting container: 06e23826a3e6
2023/10/26 15:35:51 ✅ Container started: 06e23826a3e6
2023/10/26 15:35:51 🚧 Waiting for container id 06e23826a3e6 image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/26 15:35:51 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v23.1.7
2023/10/26 15:35:51 ✅ Container created: 125662db9cef
2023/10/26 15:35:51 🐳 Starting container: 125662db9cef
2023/10/26 15:35:51 ✅ Container started: 125662db9cef
=== RUN   TestBroker/Send_Rating_without_callback
=== RUN   TestBroker/Send_Rating_with_error_in_callback
--- PASS: TestBroker (1.57s)
    --- PASS: TestBroker/Send_Rating_without_callback (0.57s)
    --- PASS: TestBroker/Send_Rating_with_error_in_callback (0.00s)
PASS
ok      github.com/testcontainers/workshop-go/internal/streams  1.714s
```

_NOTE: if we experiment longer test execution times it could be caused by the need of pulling the images from the registry._

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

There is no need to run `go mod tidy` again, as we have already downloaded the Go dependencies.

Finally, run your tests with `go test -v -count=1 ./internal/talks -run TestNewRepository` from the root of the project. We should see the following output:

```text
=== RUN   TestNewRepository
2023/10/26 15:37:24 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 23.0.6 (via Testcontainers Desktop 1.5.0)
  API Version: 1.42
  Operating System: Alpine Linux v3.18
  Total Memory: 5256 MB
  Resolved Docker Host: tcp://127.0.0.1:62516
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 0755278e5207f829c9e4a1ee277604705ee78931ce1df769b6e9e77e57159258
  Test ProcessID: 729be1dc-ef48-4df4-bcac-b33551ef98e7
2023/10/26 15:37:24 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/26 15:37:24 ✅ Container created: 602d40bb5aa5
2023/10/26 15:37:24 🐳 Starting container: 602d40bb5aa5
2023/10/26 15:37:25 ✅ Container started: 602d40bb5aa5
2023/10/26 15:37:25 🚧 Waiting for container id 602d40bb5aa5 image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/26 15:37:25 🐳 Creating container for image postgres:15.3-alpine
2023/10/26 15:37:25 ✅ Container created: 38de68a70e57
2023/10/26 15:37:25 🐳 Starting container: 38de68a70e57
2023/10/26 15:37:25 ✅ Container started: 38de68a70e57
2023/10/26 15:37:25 🚧 Waiting for container id 38de68a70e57 image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x140000362e0 Strategies:[0x140004ae720]}
=== RUN   TestNewRepository/Create_a_talk_and_retrieve_it_by_UUID
=== RUN   TestNewRepository/Exists_by_UUID
=== RUN   TestNewRepository/Does_not_exist_by_UUID
2023/10/26 15:37:26 🐳 Terminating container: 38de68a70e57
2023/10/26 15:37:26 🚫 Container terminated: 38de68a70e57
--- PASS: TestNewRepository (1.55s)
    --- PASS: TestNewRepository/Create_a_talk_and_retrieve_it_by_UUID (0.00s)
    --- PASS: TestNewRepository/Exists_by_UUID (0.00s)
    --- PASS: TestNewRepository/Does_not_exist_by_UUID (0.00s)
PASS
ok      github.com/testcontainers/workshop-go/internal/talks    1.685s
```

_NOTE: if we experiment longer test execution times it could be caused by the need of pulling the images from the registry._

## Integration tests for the Ratings Lambda

Let's add a new file `internal/ratings/lambda_client_test.go` with the following content:

```go
package ratings_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"github.com/testcontainers/workshop-go/internal/ratings"
)

func TestGetStats(t *testing.T) {
	ctx := context.Background()

	flagsFn := func() string {
		labels := testcontainers.GenericLabels()
		flags := ""
		for k, v := range labels {
			flags = fmt.Sprintf("%s -l %s=%s", flags, k, v)
		}
		return flags
	}

	c, err := localstack.RunContainer(ctx,
		testcontainers.WithImage("localstack/localstack:2.3.0"),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Env: map[string]string{
					"SERVICES":            "lambda",
					"LAMBDA_DOCKER_FLAGS": flagsFn(),
				},
				Files: []testcontainers.ContainerFile{
					{
						HostFilePath:      filepath.Join("..", "..", "testdata", "function.zip"), // path to the root of the project
						ContainerFilePath: "/tmp/function.zip",
					},
				},
			},
		}),
	)
	if err != nil {
		t.Fatalf("failed to start localstack container: %s", err)
	}

	lambdaName := "localstack-lambda-url-example"

	// the three commands below are doing the following:
	// 1. create a lambda function
	// 2. create the URL function configuration for the lambda function
	// 3. wait for the lambda function to be active
	lambdaCommands := [][]string{
		{
			"awslocal", "lambda",
			"create-function", "--function-name", lambdaName,
			"--runtime", "nodejs18.x",
			"--zip-file",
			"fileb:///tmp/function.zip",
			"--handler", "index.handler",
			"--role", "arn:aws:iam::000000000000:role/lambda-role",
		},
		{"awslocal", "lambda", "create-function-url-config", "--function-name", lambdaName, "--auth-type", "NONE"},
		{"awslocal", "lambda", "wait", "function-active-v2", "--function-name", lambdaName},
	}
	for _, cmd := range lambdaCommands {
		_, _, err := c.Exec(ctx, cmd)
		if err != nil {
			t.Fatalf("failed to execute command %s: %s", cmd, err)
		}
	}

	// 4. get the URL for the lambda function
	cmd := []string{
		"awslocal", "lambda", "list-function-url-configs", "--function-name", lambdaName,
	}
	_, reader, err := c.Exec(ctx, cmd, exec.Multiplexed())
	if err != nil {
		t.Fatalf("failed to execute command %s: %s", cmd, err)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		t.Fatalf("failed to read from reader: %s", err)
	}

	content := buf.Bytes()

	type FunctionURLConfig struct {
		FunctionURLConfigs []struct {
			FunctionURL      string `json:"FunctionUrl"`
			FunctionArn      string `json:"FunctionArn"`
			CreationTime     string `json:"CreationTime"`
			LastModifiedTime string `json:"LastModifiedTime"`
			AuthType         string `json:"AuthType"`
		} `json:"FunctionUrlConfigs"`
	}

	v := &FunctionURLConfig{}
	err = json.Unmarshal(content, v)
	if err != nil {
		t.Fatalf("failed to unmarshal content: %s", err)
	}

	functionURL := v.FunctionURLConfigs[0].FunctionURL

	// replace the port with the one exposed by the container
	mappedPort, err := c.MappedPort(ctx, "4566/tcp")
	if err != nil {
		panic(err)
	}

	url := strings.ReplaceAll(functionURL, "4566", mappedPort.Port())

	// now we can test the lambda function
	lambdaClient := ratings.NewLambdaClient(url)

	histogram := map[string]string{
		"0": "10",
		"1": "20",
		"2": "30",
		"3": "40",
		"4": "50",
		"5": "60",
	}

	stats, err := lambdaClient.GetStats(histogram)
	if err != nil {
		t.Fatalf("failed to get stats: %s", err)
	}

	expected := `{"avg":3.3333333333333335,"totalCount":210}`
	if string(stats) != expected {
		t.Fatalf("expected %s, got %s", expected, string(stats))
	}
}

```

This test will start a LocalStack container, and it will define one test to verify that the lambda function returns the stats for a given rating ratings:

* `Retrieve the stats for a given histogram of ratings`: it will call the lambda deployed in the LocalStack instance, using a map of ratings as the histogram, and it will verify that the result includes the calculated average and the total count of ratings.

Please notice that the package has been named with the `_test` suffix for the same reasons describe above.

There is no need to run `go mod tidy` again, as we have already downloaded the Go dependencies.

Finally, run your tests with `go test -v -count=1 ./internal/ratings -run TestGetStats` from the root of the project. We should see the following output:

```text
=== RUN   TestGetStats
2023/10/26 15:39:18 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 23.0.6 (via Testcontainers Desktop 1.5.0)
  API Version: 1.42
  Operating System: Alpine Linux v3.18
  Total Memory: 5256 MB
  Resolved Docker Host: tcp://127.0.0.1:62516
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 0ca0574e4a18316ac8ca83fbef2c0a5d7f89a54b2c0a7a69613117c220ebfe58
  Test ProcessID: d07f13cd-3eae-411e-991d-e45c5dde742d
2023/10/26 15:39:18 Setting LOCALSTACK_HOST to 127.0.0.1 (to match host-routable address for container)
2023/10/26 15:39:18 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/26 15:39:18 ✅ Container created: e83acae6602a
2023/10/26 15:39:18 🐳 Starting container: e83acae6602a
2023/10/26 15:39:18 ✅ Container started: e83acae6602a
2023/10/26 15:39:18 🚧 Waiting for container id e83acae6602a image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/26 15:39:18 🐳 Creating container for image localstack/localstack:2.3.0
2023/10/26 15:39:18 ✅ Container created: 11cf0a213798
2023/10/26 15:39:18 🐳 Starting container: 11cf0a213798
2023/10/26 15:39:18 ✅ Container started: 11cf0a213798
2023/10/26 15:39:18 🚧 Waiting for container id 11cf0a213798 image: localstack/localstack:2.3.0. Waiting for: &{timeout:0x140000241a8 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x10117ccd0 ResponseMatcher:0x10124d8d0 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> PollInterval:100ms UserInfo:}
--- PASS: TestGetStats (8.22s)
PASS
ok      github.com/testcontainers/workshop-go/internal/ratings  8.357s
```

_NOTE: if we experiment longer test execution times it could be caused by the need of pulling the images from the registry._

We have now added integration tests for the three stores of our application, and our AWS lambda. Let's add some integration tests for the API.

### 
[Next](step-9-integration-tests-for-api.md)