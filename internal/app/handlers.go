package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/testcontainers/workshop-go/internal/ratings"
	"github.com/testcontainers/workshop-go/internal/streams"
	"github.com/testcontainers/workshop-go/internal/talks"
)

func Root(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"metadata": Connections,
	})
}

// ratingForPost is the struct that will be used to read the JSON payload
// from the POST request when a new rating is added.
type ratingForPost struct {
	UUID   string `json:"talkId" form:"talkId" binding:"required"`
	Rating int64  `json:"value" form:"value" binding:"required"`
}

// AddRating is the handler for the `POST /ratings` endpoint.
// It will add a new rating to the store, where the rating is read from the JSON payload
// using the following format:
//
//	{
//	  "talkId": "123",
//	  "value": 5
//	}
//
// If the talk with the given UUID exists in the Talks repository, it will send the rating
// to the Streams repository, which will send it to the broker. If the talk does not exist,
// or any of the repositories cannot be created, it will return an error.
func AddRating(c *fiber.Ctx) error {
	var r ratingForPost

	if err := c.BodyParser(&r); err != nil {
		return handleError(c, http.StatusInternalServerError, err)
	}

	talksRepo, err := talks.NewRepository(c.Context(), Connections.Talks)
	if err != nil {
		return handleError(c, http.StatusInternalServerError, err)
	}

	if !talksRepo.Exists(c.Context(), r.UUID) {
		return handleError(c, http.StatusNotFound, fmt.Errorf("talk with UUID %s does not exist", r.UUID))
	}

	streamsRepo, err := streams.NewStream(c.Context(), Connections.Streams)
	if err != nil {
		return handleError(c, http.StatusInternalServerError, err)
	}

	ratingsRepo, err := ratings.NewRepository(c.Context(), Connections.Ratings)
	if err != nil {
		return handleError(c, http.StatusInternalServerError, err)
	}

	rating := ratings.Rating{
		TalkUuid: r.UUID,
		Value:    r.Rating,
	}

	ratingsCallback := func() error {
		_, err := ratingsRepo.Add(c.Context(), rating)
		return err
	}

	err = streamsRepo.SendRating(c.Context(), rating, ratingsCallback)
	if err != nil {
		return handleError(c, http.StatusInternalServerError, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"rating": rating,
	})
}

// talkForRatings is the struct that will be used to get a talk UUID from the query string
// of the GET request when the ratings for a talk are requested.
type talkForRatings struct {
	UUID string `json:"talkId" form:"talkId" binding:"required"`
}

type statsResponse struct {
	Avg        float64 `json:"avg"`
	TotalCount int64   `json:"totalCount"`
}

// Ratings is the handler for the `GET /ratings?talkId=xxx` endpoint. It will require a talkId parameter
// in the query string and will return all the ratings for the given talk UUID.
func Ratings(c *fiber.Ctx) error {
	talkID := c.Query("talkId", "")
	if talkID == "" {
		return handleError(c, http.StatusInternalServerError, errors.New("talkId is required"))
	}

	talk := talkForRatings{UUID: talkID}

	talksRepo, err := talks.NewRepository(c.Context(), Connections.Talks)
	if err != nil {
		return handleError(c, http.StatusInternalServerError, err)
	}

	if !talksRepo.Exists(c.Context(), talk.UUID) {
		return handleError(c, http.StatusNotFound, fmt.Errorf("talk with UUID %s does not exist", talk.UUID))
	}

	ratingsRepo, err := ratings.NewRepository(c.Context(), Connections.Ratings)
	if err != nil {
		return handleError(c, http.StatusInternalServerError, err)
	}

	histogram := ratingsRepo.FindAllByUUID(c.Context(), talk.UUID)

	// call the lambda function to get the stats
	lambdaClient := ratings.NewLambdaClient(Connections.Lambda)
	stats, err := lambdaClient.GetStats(histogram)
	if err != nil {
		// do not fail if the lambda function is not available, simply do not aggregate the stats
		log.Printf("error calling lambda function: %s", err.Error())
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"ratings": histogram,
		})
	}

	statsResp := &statsResponse{}
	err = json.Unmarshal(stats, statsResp)
	if err != nil {
		// do not fail if the lambda function is not available, simply do not aggregate the stats
		log.Printf("error unmarshalling lambda response: %s", err.Error())
		return c.Status(http.StatusOK).JSON(fiber.Map{
			"ratings": histogram,
		})
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"ratings": histogram,
		"stats":   statsResp,
	})
}

func handleError(c *fiber.Ctx, code int, err error) error {
	return c.Status(code).JSON(fiber.Map{
		"message": err.Error(),
	})
}
