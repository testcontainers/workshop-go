package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func SetupApp() *fiber.App {
	app := fiber.New()

	app.Use(logger.New())

	app.Get("/", Root)
	app.Get("/ratings", Ratings)
	app.Post("/ratings", AddRating)

	return app
}
