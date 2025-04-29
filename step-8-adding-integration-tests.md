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

	redisContainer, err := tcRedis.Run(ctx, "redis:6-alpine")
	testcontainers.CleanupContainer(t, redisContainer)
	require.NoError(t, err)

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
2025/03/25 13:28:43 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 27.5.0 (via Testcontainers Desktop 1.19.0)
  API Version: 1.46
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15368 MB
  Labels:
    cloud.docker.run.version=259.c712f5fd
  Testcontainers for Go Version: v0.35.0
  Resolved Docker Host: tcp://127.0.0.1:49982
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 108e56b58c673b34136ef7aff4cc8629b6101a9737009f275fed7592aa75d3af
  Test ProcessID: f3796459-23e9-4320-a1db-328094645da2
2025/03/25 13:28:43 🐳 Creating container for image redis:6-alpine
2025/03/25 13:28:44 🐳 Creating container for image testcontainers/ryuk:0.11.0
2025/03/25 13:28:44 ✅ Container created: 609a132f79e0
2025/03/25 13:28:44 🐳 Starting container: 609a132f79e0
2025/03/25 13:28:44 ✅ Container started: 609a132f79e0
2025/03/25 13:28:44 ⏳ Waiting for container id 609a132f79e0 image: testcontainers/ryuk:0.11.0. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms skipInternalCheck:false}
2025/03/25 13:28:44 🔔 Container is ready: 609a132f79e0
2025/03/25 13:28:44 ✅ Container created: dac0babc7b42
2025/03/25 13:28:44 🐳 Starting container: dac0babc7b42
2025/03/25 13:28:45 ✅ Container started: dac0babc7b42
2025/03/25 13:28:45 ⏳ Waiting for container id dac0babc7b42 image: redis:6-alpine. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms check:<nil> submatchCallback:<nil> re:<nil> log:[]}
2025/03/25 13:28:45 🔔 Container is ready: dac0babc7b42
=== RUN   TestNewRepository/Add_rating
=== RUN   TestNewRepository/Add_multiple_ratings
2025/03/25 13:28:48 🐳 Stopping container: dac0babc7b42
2025/03/25 13:28:48 ✅ Container stopped: dac0babc7b42
2025/03/25 13:28:48 🐳 Terminating container: dac0babc7b42
2025/03/25 13:28:48 🚫 Container terminated: dac0babc7b42
--- PASS: TestNewRepository (5.18s)
    --- PASS: TestNewRepository/Add_rating (0.03s)
    --- PASS: TestNewRepository/Add_multiple_ratings (3.35s)
PASS
ok      github.com/testcontainers/workshop-go/internal/ratings  5.492s
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

	redpandaC, err := redpanda.Run(
		ctx,
		"docker.redpanda.com/redpandadata/redpanda:v24.3.7",
		redpanda.WithAutoCreateTopics(),
	)
	testcontainers.CleanupContainer(t, redpandaC)
	require.NoError(t, err)

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
2025/03/25 13:27:43 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 27.5.0 (via Testcontainers Desktop 1.19.0)
  API Version: 1.46
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15368 MB
  Labels:
    cloud.docker.run.version=259.c712f5fd
  Testcontainers for Go Version: v0.35.0
  Resolved Docker Host: tcp://127.0.0.1:49982
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: f3efe8aa74049e456c1d8711ec74a7ac666105ea4996c5f5166099592f93160c
  Test ProcessID: 803f586a-f2aa-41aa-adb5-e5bb2a8ce85e
2025/03/25 13:27:43 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v24.3.7
2025/03/25 13:27:43 🐳 Creating container for image testcontainers/ryuk:0.11.0
2025/03/25 13:27:43 ✅ Container created: 220b54e84226
2025/03/25 13:27:43 🐳 Starting container: 220b54e84226
2025/03/25 13:27:43 ✅ Container started: 220b54e84226
2025/03/25 13:27:43 ⏳ Waiting for container id 220b54e84226 image: testcontainers/ryuk:0.11.0. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms skipInternalCheck:false}
2025/03/25 13:27:44 🔔 Container is ready: 220b54e84226
2025/03/25 13:27:44 ✅ Container created: 801391cb30bf
2025/03/25 13:27:44 🐳 Starting container: 801391cb30bf
2025/03/25 13:27:44 ✅ Container started: 801391cb30bf
2025/03/25 13:27:44 ⏳ Waiting for container id 801391cb30bf image: docker.redpanda.com/redpandadata/redpanda:v24.3.7. Waiting for: &{timeout:<nil> deadline:<nil> Strategies:[0x1400041cae0 0x1400041cb10 0x1400041cb40]}
2025/03/25 13:27:44 🔔 Container is ready: 801391cb30bf
=== RUN   TestBroker/Send_Rating_without_callback
=== RUN   TestBroker/Send_Rating_with_error_in_callback
2025/03/25 13:27:47 🐳 Stopping container: 801391cb30bf
2025/03/25 13:27:47 ✅ Container stopped: 801391cb30bf
2025/03/25 13:27:47 🐳 Terminating container: 801391cb30bf
2025/03/25 13:27:47 🚫 Container terminated: 801391cb30bf
--- PASS: TestBroker (4.59s)
    --- PASS: TestBroker/Send_Rating_without_callback (0.72s)
    --- PASS: TestBroker/Send_Rating_with_error_in_callback (0.03s)
PASS
ok      github.com/testcontainers/workshop-go/internal/streams  4.938s
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

	pgContainer, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithInitScripts(filepath.Join("..", "..", "testdata", "dev-db.sql")), // path to the root of the project
		postgres.WithDatabase("talks-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	testcontainers.CleanupContainer(t, pgContainer)
	assert.NoError(t, err)

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
2025/03/25 13:31:18 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 27.5.0 (via Testcontainers Desktop 1.19.0)
  API Version: 1.46
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15368 MB
  Labels:
    cloud.docker.run.version=259.c712f5fd
  Testcontainers for Go Version: v0.35.0
  Resolved Docker Host: tcp://127.0.0.1:49982
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: f2ef0f015b36b470b519d04f7a37ceed9394461a3e34adc604278fcbb1a4d0b3
  Test ProcessID: b107a0b2-5185-46ca-b78a-1527cc6c54ce
2025/03/25 13:31:18 🐳 Creating container for image postgres:15.3-alpine
2025/03/25 13:31:18 🐳 Creating container for image testcontainers/ryuk:0.11.0
2025/03/25 13:31:18 ✅ Container created: 6dd266218b3f
2025/03/25 13:31:18 🐳 Starting container: 6dd266218b3f
2025/03/25 13:31:18 ✅ Container started: 6dd266218b3f
2025/03/25 13:31:18 ⏳ Waiting for container id 6dd266218b3f image: testcontainers/ryuk:0.11.0. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms skipInternalCheck:false}
2025/03/25 13:31:19 🔔 Container is ready: 6dd266218b3f
2025/03/25 13:31:19 ✅ Container created: 73c4474e064e
2025/03/25 13:31:19 🐳 Starting container: 73c4474e064e
2025/03/25 13:31:19 ✅ Container started: 73c4474e064e
2025/03/25 13:31:19 ⏳ Waiting for container id 73c4474e064e image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x140003d5f88 Strategies:[0x14000116840]}
2025/03/25 13:31:20 🔔 Container is ready: 73c4474e064e
=== RUN   TestNewRepository/Create_a_talk_and_retrieve_it_by_UUID
=== RUN   TestNewRepository/Exists_by_UUID
=== RUN   TestNewRepository/Does_not_exist_by_UUID
2025/03/25 13:31:21 🐳 Stopping container: 73c4474e064e
2025/03/25 13:31:21 ✅ Container stopped: 73c4474e064e
2025/03/25 13:31:21 🐳 Terminating container: 73c4474e064e
2025/03/25 13:31:21 🚫 Container terminated: 73c4474e064e
--- PASS: TestNewRepository (3.70s)
    --- PASS: TestNewRepository/Create_a_talk_and_retrieve_it_by_UUID (0.18s)
    --- PASS: TestNewRepository/Exists_by_UUID (0.07s)
    --- PASS: TestNewRepository/Does_not_exist_by_UUID (0.04s)
PASS
ok      github.com/testcontainers/workshop-go/internal/talks    4.433s
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
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"github.com/testcontainers/workshop-go/internal/ratings"
)

// buildLambda return the path to the ZIP file used to deploy the lambda function.
func buildLambda(t *testing.T) string {
	t.Helper()

	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)

	lambdaPath := filepath.Join(basepath, "..", "..", "lambda-go")

	makeCmd := osexec.Command("make", "zip-lambda")
	makeCmd.Dir = lambdaPath

	err := makeCmd.Run()
	require.NoError(t, err)

	return filepath.Join(lambdaPath, "function.zip")
}

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

	// get the path to the function.zip file, which lives in the lambda-go folder of the project
	zipFile := buildLambda(t)

	var functionURL string

	c, err := localstack.Run(ctx,
		"localstack/localstack:latest",
		testcontainers.WithEnv(map[string]string{
			"SERVICES":            "lambda",
			"LAMBDA_DOCKER_FLAGS": flagsFn(),
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      zipFile,
			ContainerFilePath: "/tmp/function.zip",
		}),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
					{
						PostStarts: []testcontainers.ContainerHook{
							func(ctx context.Context, c testcontainers.Container) error {
								lambdaName := "localstack-lambda-url-example"

								// the three commands below are doing the following:
								// 1. create a lambda function
								// 2. create the URL function configuration for the lambda function
								// 3. wait for the lambda function to be active
								lambdaCommands := [][]string{
									{
										"awslocal", "lambda",
										"create-function", "--function-name", lambdaName,
										"--runtime", "provided.al2",
										"--handler", "bootstrap",
										"--role", "arn:aws:iam::111122223333:role/lambda-ex",
										"--zip-file", "fileb:///tmp/function.zip",
									},
									{"awslocal", "lambda", "create-function-url-config", "--function-name", lambdaName, "--auth-type", "NONE"},
									{"awslocal", "lambda", "wait", "function-active-v2", "--function-name", lambdaName},
								}
								for _, cmd := range lambdaCommands {
									_, _, err := c.Exec(ctx, cmd)
									if err != nil {
										return err
									}
								}

								// 4. get the URL for the lambda function
								cmd := []string{
									"awslocal", "lambda", "list-function-url-configs", "--function-name", lambdaName,
								}
								_, reader, err := c.Exec(ctx, cmd, exec.Multiplexed())
								if err != nil {
									return err
								}

								buf := new(bytes.Buffer)
								_, err = buf.ReadFrom(reader)
								if err != nil {
									return err
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
									return err
								}

								// 5. finally, set the function URL from the response
								functionURL = v.FunctionURLConfigs[0].FunctionURL

								return nil
							},
						},
					},
				},
			},
		}),
	)
	testcontainers.CleanupContainer(t, c)
	require.NoError(t, err)

	// replace the port with the one exposed by the container
	mappedPort, err := c.MappedPort(ctx, "4566/tcp")
	require.NoError(t, err)

	url := strings.ReplaceAll(functionURL, "4566", mappedPort.Port())

	// The latest version of localstack does not add ".localstack.cloud" by default,
	// that's why we need to add it to the URL.
	url = strings.ReplaceAll(url, ".localhost", ".localhost.localstack.cloud")

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
	require.NoError(t, err)

	expected := `{"avg":3.3333333333333335,"totalCount":210}`
	require.Equal(t, expected, string(stats))
}

```

This test will start a LocalStack container, previously building the ZIP file representing the lambda, and it will define one test to verify that the lambda function returns the stats for a given histogram of ratings:

* `Retrieve the stats for a given histogram of ratings`: it will call the lambda deployed in the LocalStack instance, using a map of ratings as the histogram, and it will verify that the result includes the calculated average and the total count of ratings.

Please notice that the package has been named with the `_test` suffix for the same reasons describe above.

There is no need to run `go mod tidy` again, as we have already downloaded the Go dependencies.

Finally, run your tests with `go test -v -count=1 ./internal/ratings -run TestGetStats` from the root of the project. We should see the following output:

```text
=== RUN   TestGetStats
2025/03/25 13:35:14 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 27.5.0 (via Testcontainers Desktop 1.19.0)
  API Version: 1.46
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15368 MB
  Labels:
    cloud.docker.run.version=259.c712f5fd
  Testcontainers for Go Version: v0.35.0
  Resolved Docker Host: tcp://127.0.0.1:49982
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 4537b6af9f46af836f202c95ef2e5dadf3ba8c33ef605e0191ae857cb20e2ae3
  Test ProcessID: 975e388b-e4ee-4f73-8d5f-f16b26a07464
2025/03/25 13:35:14 Setting LOCALSTACK_HOST to 127.0.0.1 (to match host-routable address for container)
2025/03/25 13:35:14 🐳 Creating container for image localstack/localstack:latest
2025/03/25 13:35:15 🐳 Creating container for image testcontainers/ryuk:0.11.0
2025/03/25 13:35:15 ✅ Container created: 0cfa2462825f
2025/03/25 13:35:15 🐳 Starting container: 0cfa2462825f
2025/03/25 13:35:15 ✅ Container started: 0cfa2462825f
2025/03/25 13:35:15 ⏳ Waiting for container id 0cfa2462825f image: testcontainers/ryuk:0.11.0. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms skipInternalCheck:false}
2025/03/25 13:35:15 🔔 Container is ready: 0cfa2462825f
2025/03/25 13:35:15 ✅ Container created: 7bbf96d6bcca
2025/03/25 13:35:16 🐳 Starting container: 7bbf96d6bcca
2025/03/25 13:35:25 ✅ Container started: 7bbf96d6bcca
2025/03/25 13:35:25 ⏳ Waiting for container id 7bbf96d6bcca image: localstack/localstack:latest. Waiting for: &{timeout:0x140003b7b40 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x1009efae0 ResponseMatcher:0x100a435e0 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> Headers:map[] ResponseHeadersMatcher:0x100a435f0 PollInterval:100ms UserInfo: ForceIPv4LocalHost:false}
2025/03/25 13:35:25 🔔 Container is ready: 7bbf96d6bcca
2025/03/25 13:35:25 🐳 Stopping container: 7bbf96d6bcca
2025/03/25 13:35:31 ✅ Container stopped: 7bbf96d6bcca
2025/03/25 13:35:31 🐳 Terminating container: 7bbf96d6bcca
2025/03/25 13:35:31 🚫 Container terminated: 7bbf96d6bcca
--- PASS: TestGetStats (16.97s)
PASS
ok      github.com/testcontainers/workshop-go/internal/ratings  17.966s
```

_NOTE: if we experiment longer test execution times it could be caused by the need of pulling the images from the registry._

We have now added integration tests for the three stores of our application, and our AWS lambda. Let's add some integration tests for the API.

### 
[Next: Adding integration tests for the APIs](step-9-integration-tests-for-api.md)