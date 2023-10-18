//go:build dev
// +build dev

package app

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

// init will be used to start up the containers for development mode. It will use
// testcontainers-go to start up the following containers:
// - Redis: store for ratings
// All the containers will contribute their connection strings to the Connections struct.
func init() {
	ctx := context.Background()

	c, err := redis.RunContainer(ctx, testcontainers.WithImage("redis:6-alpine"))
	if err != nil {
		panic(err)
	}

	ratingsConn, err := c.ConnectionString(ctx)
	if err != nil {
		panic(err)
	}

	Connections.Ratings = ratingsConn
}
