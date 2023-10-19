package app

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/workshop-go/internal/ratings"
	"github.com/testcontainers/workshop-go/internal/streams"
	"github.com/testcontainers/workshop-go/internal/talks"
)

func Root(c *gin.Context) {
	c.HTML(http.StatusOK, "metadata.tmpl", gin.H{
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
func AddRating(c *gin.Context) {
	var r ratingForPost
	err := c.ShouldBind(&r)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	talksRepo, err := talks.NewRepository(c, Connections.Talks)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	if !talksRepo.Exists(c, r.UUID) {
		handleError(c, http.StatusNotFound, fmt.Errorf("talk with UUID %s does not exist", r.UUID))
		return
	}

	streamsRepo, err := streams.NewStream(c, Connections.Streams)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	ratingsRepo, err := ratings.NewRepository(c, Connections.Ratings)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	rating := ratings.Rating{
		TalkUuid: r.UUID,
		Value:    r.Rating,
	}

	ratingsCallback := func() error {
		_, err := ratingsRepo.Add(c, rating)
		return err
	}

	err = streamsRepo.SendRating(c, rating, ratingsCallback)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"rating": rating,
	})
}

// talkForRatings is the struct that will be used to get a talk UUID from the query string
// of the GET request when the ratings for a talk are requested.
type talkForRatings struct {
	UUID string `json:"talkId" form:"talkId" binding:"required"`
}

// Ratings is the handler for the `GET /ratings?talkId=xxx` endpoint. It will require a talkId parameter
// in the query string and will return all the ratings for the given talk UUID.
func Ratings(c *gin.Context) {
	var talk talkForRatings
	if err := c.ShouldBind(&talk); err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	talksRepo, err := talks.NewRepository(c, Connections.Talks)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	if !talksRepo.Exists(c, talk.UUID) {
		handleError(c, http.StatusNotFound, fmt.Errorf("talk with UUID %s does not exist", talk.UUID))
		return
	}

	ratingsRepo, err := ratings.NewRepository(c, Connections.Ratings)
	if err != nil {
		handleError(c, http.StatusInternalServerError, err)
		return
	}

	histogram := ratingsRepo.FindAllByUUID(c, talk.UUID)

	c.JSON(http.StatusOK, gin.H{
		"ratings": histogram,
	})
}

func handleError(c *gin.Context, code int, err error) {
	c.JSON(code, gin.H{
		"message": err.Error(),
	})
}
