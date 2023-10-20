package main

import (
	"github.com/testcontainers/workshop-go/internal/app"
)

func main() {
	router := app.SetupRouter()

	router.LoadHTMLGlob("templates/*.tmpl")

	router.Run(":8080")
}
