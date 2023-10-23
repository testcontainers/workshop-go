# Step 8: Integration tests for the API

In this step you will add integration tests for the API, and for that we are going to use the [`net/httptest`](https://pkg.go.dev/net/http/httptest) package from the standard library.

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

func TestRootRoute(t *testing.T) {
	router := app.SetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// the metadata is empty because we do not have real dependencies started
	assert.Equal(t, `{"metadata":{"ratings":"","streams":"","talks":""}}`, w.Body.String())
}

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

Let's check what we are doing here. For the first test method, `TestRootRoute`, we are doing the following:

- We are creating a new `httptest.Recorder` to record the response.
- We are creating a new `http.Request` with the `GET` method and the `/` path.
- We are calling the `ServeHTTP` method on the router with the `httptest.Recorder` and the `http.Request`.
- We are asserting that the response code is `200`.
- We are asserting that the response body is `{"metadata":{"Ratings":"","Streams":"","Talks":""}}`. The values of the connection strings are empty because we do not have any dependency started.

The second test method, `TestRoutesFailBecauseDependenciesAreNotStarted`, is verifying that the routes that depend on the repositories are failing with a `500 Internal Server error` response code. That's because the runtime dependencies are not started for this test.
