package main

import (
	"github.com/testcontainers/workshop-go/internal/app"
)

func main() {
	router := app.SetupRouter()

	router.Run(":8080")
}
