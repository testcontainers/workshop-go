package app

import "os"

type connections struct {
	// The connection string for the ratings store. The application will need it to connect to the store,
	// reading it from the RATINGS_CONNECTION environment variable in production, or from the container in development.
	Ratings string
}

var Connections *connections = &connections{
	Ratings: os.Getenv("RATINGS_CONNECTION"),
}
