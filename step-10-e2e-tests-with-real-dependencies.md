# Step 10: End-To-End tests with real dependencies

In the previous step we added integration tests for the API, and for that we used the [`net/httptest`](https://pkg.go.dev/net/http/httptest) package from the standard library. But the HTTP handlers in the application are checking for the existence of the dependencies, and if they are not there, they return an error (see `internal/app/handlers.go`).

The tests that we added in the previous step are using the `httptest` package to test the handlers, but they are not testing the dependencies, they are simply checking that the handlers return an error. In this step, we are going to reuse what we did for the `local dev mode` and start the dependencies using `Testcontainers`. The tests we are going to add in this step are called `End-To-End` tests (also known as `E2E`), because they are going to test the application with all its dependencies, as the HTTP handlers need them to work.

## Reusing the `local dev mode` code

In the step 4 we added the `internal/app/dev_dependencies.go` file to start the dependencies when running the application in `local dev mode`. It used a Go build tag to include the code only when the `dev` build tag is present. Let's add a build tag to also execute that code for the E2E tests of the handlers.

Please replace the build tags from to the `internal/app/dev_dependencies.go` file:

```diff
- //go:build dev
- // +build dev
+ //go:build dev || e2e
+ // +build dev e2e
```

The code in this file will be executed if and only if the build tags used in the Go toolchain match `dev` or `e2e`.

Now copy the `testdata` directory from the root directory of the project to the `internal/app` directory. This step is needed because the relative path to access the SQL script to initialize the database is different when running the tests from the root directory of the project or from the `internal/app` directory. Therefore, we need a `dev-db.sql` file in that package to be used for testing. This will allow having different data for the tests and for the application in `local dev mode`.

## Adding Make goals for running the tests

In order to simplify the experience of running the integration and the E2E tests, let's update the Makefile in the root of the project with two new targets. Please replace the content of the Makefile with the following:

```makefile
dev:
	TESTCONTAINERS_RYUK_DISABLED=true go run -tags dev -v ./...

test-integration:
	go test -v -count=1 ./...

test-e2e:
	go test -v -count=1 -tags e2e ./internal/app
```

The `test-integration` will run the integration tests, and the `test-e2e` will run the E2E tests.

At this moment the E2E tests live in the `internal/app` directory, only. Therefore the Make goal will specify that directory when running the E2E tests.

## E2E Testing the HTTP endpoints

Let's replace the entire content of the `router_test.go` file in the `internal/app` directory with the following content:

```go
//go:build e2e
// +build e2e

package app_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/workshop-go/internal/app"
)

func TestRoutesWithDependencies(t *testing.T) {
	router := app.SetupRouter()

	t.Run("GET /ratings", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/ratings?talkId=testcontainers-integration-testing", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, req)

		// we are receiving a 200 because the ratings repository is started
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST /ratings", func(t *testing.T) {
		body := []byte(`{"talkId":"testcontainers-integration-testing","value":5}`)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/ratings", bytes.NewReader(body))
		require.NoError(t, err)

        // we need to set the content type header because we are sending a body
		req.Header.Add("Content-Type", "application/json")

        router.ServeHTTP(w, req)

		// we are receiving a 200 because the ratings repository is started
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

```

- It uses the `e2e` build tag to include the code only when the `e2e` build tag is present.
- It's an exact copy of the `routes_test.go` file, which checked for the errors, but updating the test names to not indicate that the tests are failing.

If we run the test in this file, we are going to see that it fails because the dependencies are indeed started, therefore no error should be thrown:

```bash
make test-e2e
go test -v -count=1 -tags e2e ./internal/app
# github.com/testcontainers/workshop-go/internal/app.test
2023/10/25 18:58:21 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 78+testcontainerscloud (via Testcontainers Desktop 1.4.18)
  API Version: 1.43
  Operating System: Ubuntu 22.04.3 LTS
  Total Memory: 15689 MB
  Resolved Docker Host: tcp://127.0.0.1:56978
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 44ff7433ea9c6d6e734cab0e253968ed4ca7daebc03adfc8bd5275fcb94a48f2
  Test ProcessID: 00b5db03-6ae0-4f59-8198-2e98b746e571
2023/10/25 18:58:21 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/25 18:58:21 ✅ Container created: 1edcc0c3c6c8
2023/10/25 18:58:21 🐳 Starting container: 1edcc0c3c6c8
2023/10/25 18:58:21 ✅ Container started: 1edcc0c3c6c8
2023/10/25 18:58:21 🚧 Waiting for container id 1edcc0c3c6c8 image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/25 18:58:21 🐳 Creating container for image postgres:15.3-alpine
2023/10/25 18:58:21 ✅ Container created: 8fc1965aee1f
2023/10/25 18:58:21 🐳 Starting container: 8fc1965aee1f
2023/10/25 18:58:22 ✅ Container started: 8fc1965aee1f
2023/10/25 18:58:22 🚧 Waiting for container id 8fc1965aee1f image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x1400042ae88 Strategies:[0x1400043ce10]}
2023/10/25 18:58:23 🐳 Creating container for image redis:6-alpine
2023/10/25 18:58:23 ✅ Container created: e771098b1a3e
2023/10/25 18:58:23 🐳 Starting container: e771098b1a3e
2023/10/25 18:58:24 ✅ Container started: e771098b1a3e
2023/10/25 18:58:24 🚧 Waiting for container id e771098b1a3e image: redis:6-alpine. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
2023/10/25 18:58:24 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v23.1.7
2023/10/25 18:58:24 ✅ Container created: b86043dd8663
2023/10/25 18:58:24 🐳 Starting container: b86043dd8663
2023/10/25 18:58:24 ✅ Container started: b86043dd8663
2023/10/25 18:58:26 Setting LOCALSTACK_HOST to 127.0.0.1 (to match host-routable address for container)
2023/10/25 18:58:26 🐳 Creating container for image localstack/localstack:2.3.0
2023/10/25 18:58:26 ✅ Container created: 1b55bc25926b
2023/10/25 18:58:26 🐳 Starting container: 1b55bc25926b
2023/10/25 18:58:26 ✅ Container started: 1b55bc25926b
2023/10/25 18:58:26 🚧 Waiting for container id 1b55bc25926b image: localstack/localstack:2.3.0. Waiting for: &{timeout:0x140002980d0 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x102e96480 ResponseMatcher:0x102f67240 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> PollInterval:100ms UserInfo:}
=== RUN   TestRootRouteWithDependencies
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /                         --> github.com/testcontainers/workshop-go/internal/app.Root (3 handlers)
[GIN-debug] GET    /ratings                  --> github.com/testcontainers/workshop-go/internal/app.Ratings (3 handlers)
[GIN-debug] POST   /ratings                  --> github.com/testcontainers/workshop-go/internal/app.AddRating (3 handlers)
[GIN] 2023/10/25 - 18:58:37 | 200 |     415.334µs |                 | GET      "/"
--- PASS: TestRootRouteWithDependencies (0.00s)
=== RUN   TestRoutesWithDependencies
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /                         --> github.com/testcontainers/workshop-go/internal/app.Root (3 handlers)
[GIN-debug] GET    /ratings                  --> github.com/testcontainers/workshop-go/internal/app.Ratings (3 handlers)
[GIN-debug] POST   /ratings                  --> github.com/testcontainers/workshop-go/internal/app.AddRating (3 handlers)
=== RUN   TestRoutesWithDependencies/GET_/ratings
2023/10/25 18:58:38 error unmarshalling lambda response: invalid character 'I' looking for beginning of value
[GIN] 2023/10/25 - 18:58:38 | 200 |  1.002657417s |                 | GET      "/ratings?talkId=testcontainers-integration-testing"
=== RUN   TestRoutesWithDependencies/POST_/ratings
[GIN] 2023/10/25 - 18:58:39 | 200 |  868.640667ms |                 | POST     "/ratings"
--- PASS: TestRoutesWithDependencies (1.87s)
    --- PASS: TestRoutesWithDependencies/GET_/ratings (1.00s)
    --- PASS: TestRoutesWithDependencies/POST_/ratings (0.87s)
PASS
ok      github.com/testcontainers/workshop-go/internal/app      18.910s
```

Please take a look at these things:

1. the `e2e` build tag is passed to the Go toolchain (e.g. `-tags e2e`), so the code in the `internal/app/dev_dependencies.go` file is executed for this test execution.
2. both tests for the endpoints (`GET /ratings` and `POST /ratings`) are now passing because the endpoints are returning a `200` instead of a `500`: the dependencies are started, and the endpoints are not returning an error.

### Adding a test for the `GET /` endpoint

When running in production, the `GET /` endpoint returns metadata with the connections for the dependencies. Let's add a test for that endpoint.

First make sure the imports are properly updated into the `internal/app/router_test.go` file to include the `encoding/json`, `fmt`, and `strings` packages:

```go
import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/workshop-go/internal/app"
)
```

Then please add the following test function into the `internal/app/router_test.go` file:

```go
func TestRootRouteWithDependencies(t *testing.T) {
	router := app.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// the "GET /" endpoint returns a JSON with metadata including
	// the connection strings for the dependencies
	var response struct {
		Connections app.Metadata `json:"metadata"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// assert that the different connection strings are set
	assert.True(t, strings.Contains(response.Connections.Ratings, "redis://127.0.0.1:"), fmt.Sprintf("expected %s to be a Redis URL", response.Connections.Ratings))
	assert.True(t, strings.Contains(response.Connections.Streams, "127.0.0.1:"), fmt.Sprintf("expected %s to be Redpanda URL", response.Connections.Streams))
	assert.True(t, strings.Contains(response.Connections.Talks, "postgres://postgres:postgres@127.0.0.1:"), fmt.Sprintf("expected %s to be a Postgres URL", response.Connections.Talks))
	assert.True(t, strings.Contains(response.Connections.Lambda, "lambda-url.us-east-1.localhost.localstack.cloud:"), fmt.Sprintf("expected %s to be a Lambda URL", response.Connections.Lambda))
}
```

- It uses the `Metadata` struct from the `internal/app/metadata.go` file to unmarshal the response into a response struct.
- It asserts that the different connection strings are set. Because the ports in which each dependency is started are random, we are only checking that the connection strings contain the expected values, without checking the exact port.

Running the tests again with `make test-e2e` shows that the new test is also passing:

```bash
=== RUN   TestRootRouteWithDependencies
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /                         --> github.com/testcontainers/workshop-go/internal/app.Root (3 handlers)
[GIN-debug] GET    /ratings                  --> github.com/testcontainers/workshop-go/internal/app.Ratings (3 handlers)
[GIN-debug] POST   /ratings                  --> github.com/testcontainers/workshop-go/internal/app.AddRating (3 handlers)
[GIN] 2023/10/23 - 15:32:22 | 200 |     131.458µs |                 | GET      "/"
--- PASS: TestRootRouteWithDependencies (0.00s)
``````