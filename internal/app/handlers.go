package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Root(c *gin.Context) {
	c.HTML(http.StatusOK, "metadata.tmpl", gin.H{
		"metadata": Connections,
	})
}

func AddRating(c *gin.Context) {
	c.HTML(http.StatusOK, "ratings-add.tmpl", gin.H{})
}

func Ratings(c *gin.Context) {
	c.HTML(http.StatusOK, "ratings-list.tmpl", gin.H{})
}
