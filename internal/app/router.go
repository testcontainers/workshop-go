package app

import (
	"github.com/gofiber/fiber/v2"
)

func SetupApp() *fiber.App {
	app := fiber.New()

	app.Get("/", Root)
	app.Get("/ratings", Ratings)
	app.Post("/ratings", AddRating)

	return app
}
