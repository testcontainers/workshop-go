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

// AddRating is the handler for the `POST /ratings` endpoint.
// It will add a new rating to the store, where the rating is read from the JSON payload
// using the following format:
//
//	{
//	  "talk_uuid": "123",
//	  "rating": 5
//	}
//
// If the talk with the given UUID exists in the Talks repository, it will send the rating
// to the Streams repository, which will send it to the broker. If the talk does not exist,
// or any of the repositories cannot be created, it will return an error.
func AddRating(c *gin.Context) {
	var rating ratings.Rating
	err := c.ShouldBind(&rating)
	if err != nil {
		handleError(c, err)
		return
	}

	talksRepo, err := talks.NewRepository(c, Connections.Talks)
	if err != nil {
		handleError(c, err)
		return
	}

	if !talksRepo.Exists(c, rating.TalkUuid) {
		handleError(c, fmt.Errorf("talk with UUID %s does not exist", rating.TalkUuid))
		return
	}

	streamsRepo, err := streams.NewStream(c, Connections.Streams)
	if err != nil {
		handleError(c, err)
		return
	}

	err = streamsRepo.SendRating(c, rating)
	if err != nil {
		handleError(c, err)
		return
	}

	c.HTML(http.StatusOK, "ratings-add.tmpl", gin.H{
		"rating": rating,
	})
}

type talkForRatings struct {
	UUID string `json:"talkId" form:"talkId" binding:"required"`
}

// Ratings is the handler for the `GET /ratings?talkId=xxx` endpoint. It will require a talkId parameter
// in the query string and will return all the ratings for the given talk UUID.
func Ratings(c *gin.Context) {
	var talk talkForRatings
	if err := c.ShouldBind(&talk); err != nil {
		handleError(c, err)
		return
	}

	talksRepo, err := talks.NewRepository(c, Connections.Talks)
	if err != nil {
		handleError(c, err)
		return
	}

	if !talksRepo.Exists(c, talk.UUID) {
		handleError(c, fmt.Errorf("talk with UUID %s does not exist", talk.UUID))
		return
	}

	ratingsRepo, err := ratings.NewRepository(c, Connections.Ratings)
	if err != nil {
		handleError(c, err)
		return
	}

	histogram := ratingsRepo.FindAllByUUID(c, talk.UUID)

	c.HTML(http.StatusOK, "ratings-list.tmpl", gin.H{
		"ratings": histogram,
	})
}

func handleError(c *gin.Context, err error) {
	c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
		"message": err.Error(),
	})
}
