package app

import "github.com/gin-gonic/gin"

func SetupRouter() *gin.Engine {
	router := gin.Default()

	router.GET("/", Root)
	router.GET("/ratings", Ratings)
	router.POST("/ratings", AddRating)

	return router
}
