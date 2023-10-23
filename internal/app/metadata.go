package app

import "os"

// The connection string for each of the services needed by the application.
// The application will need them to connect to services, reading it from
// the right environment variable in production, or from the container in development.
type connections struct {
	Ratings string `json:"ratings"` // Read from the RATINGS_CONNECTION environment variable
	Streams string `json:"streams"` // Read from the STREAMS_CONNECTION environment variable
	Talks   string `json:"talks"`   // Read from the TALKS_CONNECTION environment variable
}

var Connections *connections = &connections{
	Ratings: os.Getenv("RATINGS_CONNECTION"),
	Streams: os.Getenv("STREAMS_CONNECTION"),
	Talks:   os.Getenv("TALKS_CONNECTION"),
}
