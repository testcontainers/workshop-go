package main

import (
	"github.com/testcontainers/workshop-go/internal/app"
)

func main() {
	app := app.SetupApp()

	app.Listen(":8080")
}
