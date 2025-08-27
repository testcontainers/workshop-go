//go:build e2e
// +build e2e

package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/workshop-go/internal/app"
)

// the "GET /" endpoint returns a JSON with metadata including
// the connection strings for the dependencies
type responseType struct {
	Connections app.Metadata `json:"metadata"`
}

func TestRootRouteWithDependencies(t *testing.T) {
	app := app.SetupApp()

	req, _ := http.NewRequest("GET", "/", nil)
	res, err := app.Test(req, -1)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, res.StatusCode)

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var response responseType
	err = json.Unmarshal(body, &response)
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

	require.True(t, matched, fmt.Sprintf("expected %s to be an URL: %s", actual, re))
}

func TestRoutesWithDependencies(t *testing.T) {
	app := app.SetupApp()

	t.Run("GET /ratings", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/ratings?talkId=testcontainers-integration-testing", nil)
		require.NoError(t, err)
		res, err := app.Test(req, -1)
		require.NoError(t, err)

		// we are receiving a 200 because the ratings repository is started
		require.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("POST /ratings", func(t *testing.T) {
		body := []byte(`{"talkId":"testcontainers-integration-testing","value":5}`)

		req, err := http.NewRequest("POST", "/ratings", bytes.NewReader(body))
		require.NoError(t, err)

		// we need to set the content type header because we are sending a body
		req.Header.Add("Content-Type", "application/json")

		res, err := app.Test(req, -1)
		require.NoError(t, err)

		// we are receiving a 200 because the ratings repository is started
		require.Equal(t, http.StatusOK, res.StatusCode)
	})
}
