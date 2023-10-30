package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type RatingsEvent struct {
	Ratings map[string]int `json:"ratings"`
}

type Response struct {
	Avg        float64 `json:"avg"`
	TotalCount int     `json:"totalCount"`
}

var emptyResponse = Response{
	Avg:        0,
	TotalCount: 0,
}

// HandleStats returns the stats for the given talk, obtained from a call to the Lambda function.
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
func HandleStats(event events.APIGatewayProxyRequest) (Response, error) {
	ratingsEvent := RatingsEvent{}
	err := json.Unmarshal([]byte(event.Body), &ratingsEvent)
	if err != nil {
		return emptyResponse, fmt.Errorf("failed to unmarshal ratings event: %s", err)
	}

	var totalCount int
	var sum int
	for rating, count := range ratingsEvent.Ratings {
		totalCount += count

		r, err := strconv.Atoi(rating)
		if err != nil {
			return emptyResponse, fmt.Errorf("failed to convert rating %s to int: %s", rating, err)
		}

		sum += count * r
	}

	var avg float64
	if totalCount > 0 {
		avg = float64(sum) / float64(totalCount)
	}

	resp := Response{
		Avg:        avg,
		TotalCount: totalCount,
	}

	return resp, nil
}

func main() {
	lambda.Start(HandleStats)
}
