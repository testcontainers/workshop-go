package main

import (
	"github.com/gin-gonic/gin"
	"github.com/testcontainers/workshop-go/internal/app"
)

func main() {
	router := gin.Default()

	router.LoadHTMLGlob("templates/**/*")

	router.GET("/", app.Root)
	router.GET("/ratings", app.Ratings)
	router.POST("/ratings", app.AddRating)

	router.Run(":8080")
}
