package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Root(c *gin.Context) {
	c.HTML(http.StatusOK, "raw.tmpl", gin.H{})
}

func AddRating(c *gin.Context) {
	c.HTML(http.StatusOK, "ratings-add.tmpl", gin.H{})
}

func Ratings(c *gin.Context) {
	c.HTML(http.StatusOK, "ratings-list.tmpl", gin.H{})
}
