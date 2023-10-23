# Step 9: End-To-End tests with real dependencies

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
go test -v -count=1 -tags e2e ./... -run TestRoutesWithDependencies
?       github.com/testcontainers/workshop-go   [no test files]
2023/10/23 14:35:06 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 78+testcontainerscloud (via Testcontainers Desktop 1.4.18)
  API Version: 1.43
  Operating System: Ubuntu 20.04 LTS
  Total Memory: 7407 MB
  Resolved Docker Host: tcp://127.0.0.1:57158
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: c021f5a70f7c3e005e4389e30df96406ec6d5e6ab6cbdeaec2d9806b543acb00
  Test ProcessID: b5a46aa8-65d7-442c-8331-4e9a8ce07bae
2023/10/23 14:35:06 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/23 14:35:06 ✅ Container created: 6b7239ff59ec
2023/10/23 14:35:06 🐳 Starting container: 6b7239ff59ec
2023/10/23 14:35:06 ✅ Container started: 6b7239ff59ec
2023/10/23 14:35:06 🚧 Waiting for container id 6b7239ff59ec image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/23 14:35:07 🐳 Creating container for image redis:6-alpine
2023/10/23 14:35:07 ✅ Container created: d7332726b2d8
2023/10/23 14:35:07 🐳 Starting container: d7332726b2d8
2023/10/23 14:35:07 ✅ Container started: d7332726b2d8
2023/10/23 14:35:07 🚧 Waiting for container id d7332726b2d8 image: redis:6-alpine. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
2023/10/23 14:35:07 🐳 Creating container for image postgres:15.3-alpine
2023/10/23 14:35:07 ✅ Container created: bcf2e5b02a9a
2023/10/23 14:35:07 🐳 Starting container: bcf2e5b02a9a
2023/10/23 14:35:07 ✅ Container started: bcf2e5b02a9a
2023/10/23 14:35:07 🚧 Waiting for container id bcf2e5b02a9a image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x14000177218 Strategies:[0x140004f4d50]}
2023/10/23 14:35:09 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v23.1.7
2023/10/23 14:35:09 ✅ Container created: f9b685481e62
2023/10/23 14:35:09 🐳 Starting container: f9b685481e62
2023/10/23 14:35:09 ✅ Container started: f9b685481e62
=== RUN   TestRoutesWithDependencies
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /                         --> github.com/testcontainers/workshop-go/internal/app.Root (3 handlers)
[GIN-debug] GET    /ratings                  --> github.com/testcontainers/workshop-go/internal/app.Ratings (3 handlers)
[GIN-debug] POST   /ratings                  --> github.com/testcontainers/workshop-go/internal/app.AddRating (3 handlers)
=== RUN   TestRoutesWithDependencies/GET_/ratings
[GIN] 2023/10/23 - 14:35:10 | 200 |  104.857208ms |                 | GET      "/ratings?talkId=testcontainers-integration-testing"
=== RUN   TestRoutesWithDependencies/POST_/ratings
[GIN] 2023/10/23 - 14:35:11 | 200 |  572.012334ms |                 | POST     "/ratings"
--- PASS: TestRoutesWithDependencies (0.68s)
    --- PASS: TestRoutesWithDependencies/GET_/ratings (0.11s)
    --- PASS: TestRoutesWithDependencies/POST_/ratings (0.57s)
PASS
ok      github.com/testcontainers/workshop-go/internal/app      4.978s
testing: warning: no tests to run
PASS
ok      github.com/testcontainers/workshop-go/internal/ratings  0.641s [no tests to run]
testing: warning: no tests to run
PASS
ok      github.com/testcontainers/workshop-go/internal/streams  0.302s [no tests to run]
testing: warning: no tests to run
PASS
ok      github.com/testcontainers/workshop-go/internal/talks    0.813s [no tests to run]
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
}
```

- It uses the `Metadata` struct from the `internal/app/metadata.go` file to unmarshal the response into a response struct.
- It asserts that the different connection strings are set. Because the ports in which each dependency is started are random, we are only checking that the connection strings contain the expected values, without checking the exact port.
