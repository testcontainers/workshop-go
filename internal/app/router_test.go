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
