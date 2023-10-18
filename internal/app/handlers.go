package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Root(c *gin.Context) {
	c.HTML(http.StatusOK, "raw.tmpl", gin.H{})
}
