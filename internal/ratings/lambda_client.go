package ratings

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// Repository is the interface that wraps the basic operations with the Redis store.
type LambdaClient struct {
	client *http.Client
	url    string
}

// NewLambdaClient creates a new client from the Lambda URL.
func NewLambdaClient(lambdaURL string) *LambdaClient {
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}

	return &LambdaClient{
		client: &httpClient,
		url:    lambdaURL,
	}
}

// GetStats returns the stats for the given talk, obtained from a call to the Lambda function.
// The payload is a JSON object with the following structure:
//
//	{
//	  "ratings": {
//	    "0": 10,
//	    "1": 20,
//	    "2": 30,
//	    "3": 40,
//	    "4": 50,
//	    "5": 60
//	  }
//	}
//
// The response from the Lambda function is a JSON object with the following structure:
//
//	{
//	   "avg": 3.5,
//	   "totalCount": 210,
//	}
func (c *LambdaClient) GetStats(histogram map[string]string) ([]byte, error) {
	payload := `{"ratings": {`
	for rating, count := range histogram {
		// we are passing the count as an integer, so we don't need to quote it
		payload += `"` + rating + `": ` + count + `,`
	}

	if len(histogram) > 0 {
		// remove the last comma onl for non-empty histograms
		payload = payload[:len(payload)-1]
	}
	payload += "}}"

	resp, err := c.client.Post(c.url, "application/json", bytes.NewBufferString(payload))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}
