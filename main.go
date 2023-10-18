package main

import (
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/workshop-go/internal/app"
)

func main() {
	router := gin.Default()

	router.LoadHTMLFiles(filepath.Join("testdata", "raw.tmpl"))

	router.GET("/", app.Root)

	router.Run(":8080")
}
