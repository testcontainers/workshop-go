//go:build e2e
// +build e2e

package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/workshop-go/internal/app"
)

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
	matches(t, response.Connections.Ratings, `redis://(.*):`)
	matches(t, response.Connections.Streams, `(.*):`)
	matches(t, response.Connections.Talks, `postgres://postgres:postgres@(.*):`)
	matches(t, response.Connections.Lambda, `lambda-url.us-east-1.localhost.localstack.cloud:`)
}

func matches(t *testing.T, actual string, re string) {
	matched, err := regexp.MatchString(re, actual)
	require.NoError(t, err)

	assert.True(t, matched, fmt.Sprintf("expected %s to be an URL: %s", actual, re))
}

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
