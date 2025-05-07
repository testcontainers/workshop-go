# Step 9: Integration tests for the API

In this step we will add integration tests for the API, and for that we are going to use the [`net/httptest`](https://pkg.go.dev/net/http/httptest) package from the standard library.

## The `net/httptest` package

The `net/httptest` package provides a set of utilities for HTTP testing. It includes a test server that implements the `http.Handler` interface, a global `Client` to make requests to test servers, and various functions to parse HTTP responses.

For the specific use case of `GoFiber`, we are going to follow its [official documentation](https://docs.gofiber.io/recipes/unit-test/).

## Testing the HTTP endpoints

For that, we are going to create a new file called `router_test.go` inside the `internal/app` package. Please add the following content:

```go
package app_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/workshop-go/internal/app"
)

func TestRoutesFailBecauseDependenciesAreNotStarted(t *testing.T) {
	app := app.SetupApp()

	t.Run("GET /ratings fails", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/ratings?talkId=testcontainers-integration-testing", nil)
		require.NoError(t, err)
		res, err := app.Test(req, -1)
		require.NoError(t, err)

		// we are receiving a 500 because the ratings repository is not started
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	})

	t.Run("POST /ratings fails", func(t *testing.T) {
		body := []byte(`{"talkId":"testcontainers-integration-testing","value":5}`)

		req, err := http.NewRequest("POST", "/ratings", bytes.NewReader(body))
		require.NoError(t, err)

		// we need to set the content type header because we are sending a body
		req.Header.Add("Content-Type", "application/json")

		res, err := app.Test(req, -1)
		require.NoError(t, err)

		// we are receiving a 500 because the ratings repository is not started
		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	})
}

```

Let's check what we are doing here:

- We are setting up the Gin's router, with the `app.SetupApp` function.
- each subtest defines a new `http.Request`, with the right method and path.
- the `app.Test` method from GoFiber is called with the `http.Request`.
- the `TestRoutesFailBecauseDependenciesAreNotStarted` test method is verifying that the routes that depend on the repositories are failing with a `500 Internal Server error` response code. That's because the runtime dependencies are not started for this test.

Let's run the test:

```bash
go test -v -count=1 ./internal/app -run TestRoutesFailBecauseDependenciesAreNotStarted
=== RUN   TestRoutesFailBecauseDependenciesAreNotStarted
=== RUN   TestRoutesFailBecauseDependenciesAreNotStarted/GET_/ratings_fails
Unable to connect to database: failed to connect to `host=/private/tmp user=mdelapenya database=`: dial error (dial unix /private/tmp/.s.PGSQL.5432: connect: no such file or directory)
=== RUN   TestRoutesFailBecauseDependenciesAreNotStarted/POST_/ratings_fails
Unable to connect to database: failed to connect to `host=/private/tmp user=mdelapenya database=`: dial error (dial unix /private/tmp/.s.PGSQL.5432: connect: no such file or directory)
--- PASS: TestRoutesFailBecauseDependenciesAreNotStarted (0.00s)
    --- PASS: TestRoutesFailBecauseDependenciesAreNotStarted/GET_/ratings_fails (0.00s)
    --- PASS: TestRoutesFailBecauseDependenciesAreNotStarted/POST_/ratings_fails (0.00s)
PASS
ok  	github.com/testcontainers/workshop-go/internal/app	1.092s
```

This unit test is not very useful, but it is a good starting point to understand how to test the HTTP endpoints.

### 
[Next: E2E tests with real dependencies](step-10-e2e-tests-with-real-dependencies.md)