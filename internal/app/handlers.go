package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/workshop-go/internal/ratings"
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
func AddRating(c *gin.Context) {
	ratingsRepo, err := ratings.NewRepository(c, Connections.Ratings)
	if err != nil {
		handleError(c, err)
		return
	}

	var rating ratings.Rating
	err = c.ShouldBind(&rating)
	if err != nil {
		handleError(c, err)
		return
	}

	ratingsRepo.Add(c, rating)

	c.HTML(http.StatusOK, "ratings-add.tmpl", gin.H{
		"rating": rating,
	})
}

func Ratings(c *gin.Context) {
	c.HTML(http.StatusOK, "ratings-list.tmpl", gin.H{})
}

func handleError(c *gin.Context, err error) {
	c.HTML(http.StatusInternalServerError, "error.tmpl", gin.H{
		"message": err.Error(),
	})
}
