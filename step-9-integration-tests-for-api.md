# Step 9: Integration tests for the API

In this step we will add integration tests for the API, and for that we are going to use the [`net/httptest`](https://pkg.go.dev/net/http/httptest) package from the standard library.

## The `net/httptest` package

The `net/httptest` package provides a set of utilities for HTTP testing. It includes a test server that implements the `http.Handler` interface, a global `Client` to make requests to test servers, and various functions to parse HTTP responses.

For the specific use case of `Gin`, we are going to follow its [official documentation](https://gin-gonic.com/docs/testing/).

## Testing the HTTP endpoints

For that, we are going to create a new file called `router_test.go` inside the `internal/app` package. Please add the following content:

```go
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

func TestRoutesFailBecauseDependenciesAreNotStarted(t *testing.T) {
	router := app.SetupRouter()

	t.Run("GET /ratings fails", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/ratings?talkId=testcontainers-integration-testing", nil)
		require.NoError(t, err)
		router.ServeHTTP(w, req)

		// we are receiving a 500 because the ratings repository is not started
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("POST /ratings fails", func(t *testing.T) {
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

Let's check what we are doing here:

- We are setting up the Gin's router, with the `app.SetupRouter` function.
- a new `httptest.Recorder` is used to record the response.
- each subtest defines a new `http.Request`, with the right method and path.
- the `ServeHTTP` method on the router is called with the `httptest.Recorder` and the `http.Request`.
- the `TestRoutesFailBecauseDependenciesAreNotStarted` test method is verifying that the routes that depend on the repositories are failing with a `500 Internal Server error` response code. That's because the runtime dependencies are not started for this test.

Let's run the test:

```bash
go test -v -count=1 ./internal/app -run TestRoutesFailBecauseDependenciesAreNotStarted
=== RUN   TestRoutesFailBecauseDependenciesAreNotStarted
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /                         --> github.com/testcontainers/workshop-go/internal/app.Root (3 handlers)
[GIN-debug] GET    /ratings                  --> github.com/testcontainers/workshop-go/internal/app.Ratings (3 handlers)
[GIN-debug] POST   /ratings                  --> github.com/testcontainers/workshop-go/internal/app.AddRating (3 handlers)
=== RUN   TestRoutesFailBecauseDependenciesAreNotStarted/GET_/ratings_fails
Unable to connect to database: failed to connect to `host=/private/tmp user=mdelapenya database=`: dial error (dial unix /private/tmp/.s.PGSQL.5432: connect: no such file or directory)
[GIN] 2023/10/30 - 14:29:12 | 500 |    3.623292ms |                 | GET      "/ratings?talkId=testcontainers-integration-testing"
=== RUN   TestRoutesFailBecauseDependenciesAreNotStarted/POST_/ratings_fails
Unable to connect to database: failed to connect to `host=/private/tmp user=mdelapenya database=`: dial error (dial unix /private/tmp/.s.PGSQL.5432: connect: no such file or directory)
[GIN] 2023/10/30 - 14:29:12 | 500 |     105.791µs |                 | POST     "/ratings"
--- PASS: TestRoutesFailBecauseDependenciesAreNotStarted (0.00s)
    --- PASS: TestRoutesFailBecauseDependenciesAreNotStarted/GET_/ratings_fails (0.00s)
    --- PASS: TestRoutesFailBecauseDependenciesAreNotStarted/POST_/ratings_fails (0.00s)
PASS
ok      github.com/testcontainers/workshop-go/internal/app      0.308
```

This unit test is not very useful, but it is a good starting point to understand how to test the HTTP endpoints.

### 
[Next: E2E tests with real dependencies](step-10-e2e-tests-with-real-dependencies.md)