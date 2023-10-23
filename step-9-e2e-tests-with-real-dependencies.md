# Step 9: End-To-End tests with real dependencies

In the previous step we added integration tests for the API, and for that we used the [`net/httptest`](https://pkg.go.dev/net/http/httptest) package from the standard library. But the HTTP handlers in the application are checking for the existence of the dependencies, and if they are not there, they return an error (see `internal/app/handlers.go`).

The tests that we added in the previous step are using the `httptest` package to test the handlers, but they are not testing the dependencies, they are simply checking that the handlers return an error. In this step, we are going to reuse what we did for the `local dev mode` and start the dependencies using `Testcontainers`. The tests we are going to add in this step are called `End-To-End` tests (also known as `E2E`), because they are going to test the application with all its dependencies.

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

## E2E Testing the HTTP endpoints

Let's create a `router_e2e_test.go` file in the `internal/app` directory with the following content:

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

		// we are receiving a 500 because the ratings repository is not started
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("POST /ratings", func(t *testing.T) {
		body := []byte(`{"talkId":"testcontainers-integration-testing","value":5}`)

		w := httptest.NewRecorder()
		req, err := http.NewRequest("POST", "/ratings", bytes.NewReader(body))
		require.NoError(t, err)

        // we need to set the content type header because we are sending a body
		req.Header.Add("Content-Type", "application/json")

        router.ServeHTTP(w, req)

		// we are receiving a 500 because the ratings repository is not started
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

```

- It uses the `e2e` build tag to include the code only when the `e2e` build tag is present.
- It's an exact copy of the `routes_test.go` file, which checked for the errors, but updating the test names to not indicate that the tests are failing.

If we run the test in this file, we are going to see that it fails because the dependencies are indeed started, therefore no error should be thrown:

```bash
go test -v -count=1 -tags e2e ./... -run TestRoutesWithDependencies
?       github.com/testcontainers/workshop-go   [no test files]
2023/10/23 13:46:26 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 78+testcontainerscloud (via Testcontainers Desktop 1.4.18)
  API Version: 1.43
  Operating System: Ubuntu 20.04 LTS
  Total Memory: 7407 MB
  Resolved Docker Host: tcp://127.0.0.1:57158
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 1134b61999d615b16a66f3204c0d2d02e335cbcec3ce360c63676e55f3565b63
  Test ProcessID: cde64980-3b4f-4a42-9223-af0e9b9c34e8
2023/10/23 13:46:26 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/23 13:46:26 ✅ Container created: 3ba06d60968a
2023/10/23 13:46:26 🐳 Starting container: 3ba06d60968a
2023/10/23 13:46:26 ✅ Container started: 3ba06d60968a
2023/10/23 13:46:26 🚧 Waiting for container id 3ba06d60968a image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/23 13:46:27 🐳 Creating container for image redis:6-alpine
2023/10/23 13:46:27 ✅ Container created: d81aa1eea491
2023/10/23 13:46:27 🐳 Starting container: d81aa1eea491
2023/10/23 13:46:27 ✅ Container started: d81aa1eea491
2023/10/23 13:46:27 🚧 Waiting for container id d81aa1eea491 image: redis:6-alpine. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
2023/10/23 13:46:27 🐳 Creating container for image postgres:15.3-alpine
2023/10/23 13:46:27 ✅ Container created: f4c030158aa9
2023/10/23 13:46:27 🐳 Starting container: f4c030158aa9
2023/10/23 13:46:27 ✅ Container started: f4c030158aa9
2023/10/23 13:46:27 🚧 Waiting for container id f4c030158aa9 image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x14000488c58 Strategies:[0x1400015acc0]}
2023/10/23 13:46:29 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v23.1.7
2023/10/23 13:46:29 ✅ Container created: 2d0e15bd4718
2023/10/23 13:46:29 🐳 Starting container: 2d0e15bd4718
2023/10/23 13:46:29 ✅ Container started: 2d0e15bd4718
=== RUN   TestRoutesWithDependencies
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /                         --> github.com/testcontainers/workshop-go/internal/app.Root (3 handlers)
[GIN-debug] GET    /ratings                  --> github.com/testcontainers/workshop-go/internal/app.Ratings (3 handlers)
[GIN-debug] POST   /ratings                  --> github.com/testcontainers/workshop-go/internal/app.AddRating (3 handlers)
=== RUN   TestRoutesWithDependencies/GET_/ratings
[GIN] 2023/10/23 - 13:46:30 | 200 |   89.043666ms |                 | GET      "/ratings?talkId=testcontainers-integration-testing"
    router_e2e_test.go:53: 
                Error Trace:    /Users/mdelapenya/sourcecode/src/github.com/testcontainers/workshop-go/internal/app/router_e2e_test.go:53
                Error:          Not equal: 
                                expected: 500
                                actual  : 200
                Test:           TestRoutesWithDependencies/GET_/ratings
=== RUN   TestRoutesWithDependencies/POST_/ratings
[GIN] 2023/10/23 - 13:46:30 | 200 |     560.592ms |                 | POST     "/ratings"
    router_e2e_test.go:68: 
                Error Trace:    /Users/mdelapenya/sourcecode/src/github.com/testcontainers/workshop-go/internal/app/router_e2e_test.go:68
                Error:          Not equal: 
                                expected: 500
                                actual  : 200
                Test:           TestRoutesWithDependencies/POST_/ratings
--- FAIL: TestRoutesWithDependencies (0.65s)
    --- FAIL: TestRoutesWithDependencies/GET_/ratings (0.09s)
    --- FAIL: TestRoutesWithDependencies/POST_/ratings (0.56s)
FAIL
FAIL    github.com/testcontainers/workshop-go/internal/app      4.943s
testing: warning: no tests to run
PASS
ok      github.com/testcontainers/workshop-go/internal/ratings  0.483s [no tests to run]
testing: warning: no tests to run
PASS
ok      github.com/testcontainers/workshop-go/internal/streams  0.825s [no tests to run]
testing: warning: no tests to run
PASS
ok      github.com/testcontainers/workshop-go/internal/talks    0.315s [no tests to run]
FAIL
```

Please take a look at these things:

1. the `e2e` build tag is passed to the Go toolchain (e.g. `-tags e2e`), so the code in the `internal/app/dev_dependencies.go` file is executed for this test execution.
2. both tests for the endpoints (`GET /ratings` and `POST /ratings`) are failing because the endpoints are returning a `200` instead of a `500`: the dependencies are started, and the endpoints are not returning an error.

Let's change that by updating the assertions in the tests for both endpoints:

```diff
-		// we are receiving a 500 because the ratings repository is not started
-		assert.Equal(t, http.StatusInternalServerError, w.Code)
+		// we are receiving a 200 because the ratings repository is started
+		assert.Equal(t, http.StatusOK, w.Code)
```

Is we run the tests now, then the endpoints return a 200 OK, and the tests pass:

```bash
go test -v -count=1 -tags e2e ./... -run TestRoutesWithDependencies
...
--- PASS: TestRoutesWithDependencies (0.64s)
    --- PASS: TestRoutesWithDependencies/GET_/ratings (0.09s)
    --- PASS: TestRoutesWithDependencies/POST_/ratings (0.55s)
PASS
...
```

### Adding a test for the `GET /` endpoint

When running in production, the `GET /` endpoint returns metadata with the connections for the dependencies. Let's add a test for that endpoint.

Please add the following code snippet into the `internal/app/router_e2e_test.go` file:

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
}
```

- Because it's added to a test file with the `e2e` build tag, then the dependencies are started, therefore the connection strings are set into the response.
- It uses the `Metadata` struct from the `internal/app/metadata.go` file to unmarshal the response into a struct.
- It asserts that the different connection strings are set. Because the ports in which each dependency is started are random, we are only checking that the connection strings contain the expected values, without checking the exact port.
