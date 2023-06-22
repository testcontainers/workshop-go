package main

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	router.LoadHTMLFiles(filepath.Join("testdata", "raw.tmpl"))

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "raw.tmpl", gin.H{})
	})

	router.Run(":8080")
}
