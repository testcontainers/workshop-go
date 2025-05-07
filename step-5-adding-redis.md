# Step 5: Adding Redis

When the application started, and the ratings for a talk were requested, it failed because we need to connect to a Redis database before we can do anything useful with the ratings.

Let's add a Redis instance using Testcontainers for Go.

1. In the `internal/app/dev_dependencies.go` file, add the following imports:

```go
import (
       "context"
       "path/filepath"
       "time"

       "github.com/testcontainers/testcontainers-go"
       "github.com/testcontainers/testcontainers-go/modules/postgres"
       "github.com/testcontainers/testcontainers-go/modules/redis"
       "github.com/testcontainers/testcontainers-go/wait"
)
```

2. Add this function to the file:

```go
func startRatingsStore() (testcontainers.Container, error) {
       ctx := context.Background()

       c, err := redis.Run(ctx, "redis:6-alpine")
       if err != nil {
               return nil, err
       }

       ratingsConn, err := c.ConnectionString(ctx)
       if err != nil {
               return nil, err
       }

       Connections.Ratings = ratingsConn
       return c, nil
}
```

3. Update the comments for the init function `startupDependenciesFn` slice to include the Redis store:

```go
// init will be used to start up the containers for development mode. It will use
// testcontainers-go to start up the following containers:
// - Postgres: store for talks
// - Redis: store for ratings
// All the containers will contribute their connection strings to the Connections struct.
// Please read this blog post for more information: https://www.atomicjar.com/2023/08/local-development-of-go-applications-with-testcontainers/
func init() {
```

4. Update the `startupDependenciesFn` slice to include the function that starts the ratings store:

```go
	startupDependenciesFns := []func() (testcontainers.Container, error){
		startTalksStore,
		startRatingsStore,
	}
```

The complete file should look like this:

```go
//go:build dev
// +build dev

package app

import (
	"context"
	"path/filepath"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// init will be used to start up the containers for development mode. It will use
// testcontainers-go to start up the following containers:
// - Postgres: store for talks
// - Redis: store for ratings
// All the containers will contribute their connection strings to the Connections struct.
// Please read this blog post for more information: https://www.atomicjar.com/2023/08/local-development-of-go-applications-with-testcontainers/
func init() {
	startupDependenciesFns := []func() (testcontainers.Container, error){
		startTalksStore,
		startRatingsStore,
	}

	for _, fn := range startupDependenciesFns {
		_, err := fn()
		if err != nil {
			panic(err)
		}
	}
}

func startRatingsStore() (testcontainers.Container, error) {
	ctx := context.Background()

	c, err := redis.Run(ctx, "redis:6-alpine")
	if err != nil {
		return nil, err
	}

	ratingsConn, err := c.ConnectionString(ctx)
	if err != nil {
		return nil, err
	}

	Connections.Ratings = ratingsConn
	return c, nil
}

func startTalksStore() (testcontainers.Container, error) {
	ctx := context.Background()
	c, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithInitScripts(filepath.Join(".", "testdata", "dev-db.sql")),
		postgres.WithDatabase("talks-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(15*time.Second)),
	)
	if err != nil {
		return nil, err
	}

	talksConn, err := c.ConnectionString(ctx)
	if err != nil {
		return nil, err
	}

	Connections.Talks = talksConn
	return c, nil
}

```

Now run `go mod tidy` from the root of the project to download the Go dependencies, only the Testcontainers for Go's Redis module.

Finally, stop the application with <kbd>Ctrl</kbd>+<kbd>C</kbd> and run the application again with `make dev`. This time, the application will start the Redis store and the application will be able to connect to it.

```text
go run -tags dev -v ./...

 ┌───────────────────────────────────────────────────┐ 
 │                   Fiber v2.52.6                   │ 
 │               http://127.0.0.1:8080               │ 
 │       (bound on host 0.0.0.0 and port 8080)       │ 
 │                                                   │ 
 │ Handlers ............. 6  Processes ........... 1 │ 
 │ Prefork ....... Disabled  PID ............. 22626 │ 
 └───────────────────────────────────────────────────┘ 

```

In the second terminal, check the containers, we will see the Redis store running alongside the Postgres database:

```text
$ docker ps
CONTAINER ID   IMAGE                  COMMAND                  CREATED          STATUS          PORTS                                         NAMES
4ef6b38b1baa   redis:6-alpine         "docker-entrypoint.s…"   2 seconds ago    Up 1 second     0.0.0.0:32776->6379/tcp, :::32776->6379/tcp   epic_haslett
0fe7e41a8954   postgres:15.3-alpine   "docker-entrypoint.s…"   14 seconds ago   Up 13 seconds   0.0.0.0:32775->5432/tcp, :::32775->5432/tcp   affectionate_cori
```

If we now open the ratings endpoint from the API (http://localhost:8080/ratings?talkId=testcontainers-integration-testing), then a 200 OK response code is returned, but there are no ratings for the given talk:

```text
{"ratings":{}}
```

With `curl`:

```shell
curl -X GET http://localhost:8080/ratings\?talkId\=testcontainers-integration-testing                                                         
{"ratings":{}}% 
```

If we check the logs, we'll notice an error regarding the connection to the AWS lambda function that is used to calculate some statistics for a given rating. By design, if the AWS lambda is not available, the application will not add the statistics to the response, so it's expected to see this error but a valid HTTP response:

```text
2025/05/07 13:10:15 error calling lambda function: Post "": unsupported protocol scheme ""
13:10:13 | 200 |  1.253877458s | 127.0.0.1 | GET | /ratings | -
```

We are going to fix that in the next steps, adding a way to reproduce the AWS lambda but in a local environment, using LocalStack and Testcontainers for Go.

Nevertheless, now it seems the application is able to connect to the database, and to Redis. Let's try to send a POST request adding a rating for the talk, using the JSON payload format accepted by the API endopint:

```json
{
  "talkId": "testcontainers-integration-testing",
  "value": 5
}
```

In a terminal, let's send a POST request with `curl`:

```shell
curl -X POST -H "Content-Type: application/json" http://localhost:8080/ratings -d '{"talkId":"testcontainers-integration-testing", "value":5}'
```

This time, the response is a 500 error, but different:

```json
{"message":"unable to dial: dial tcp :9092: connect: connection refused"}%
```

And in the logs, we'll see the following error:

```text
Unable to ping the streams: unable to dial: dial tcp :9092: connect: connection refused
13:11:05 | 500 |  967.728458ms | 127.0.0.1 | POST | /ratings | -
```

If we recall correctly, the application was using a message queue to send the ratings before storing them in Redis (see `internal/app/handlers.go`), so we need to add a message queue for that. Let's fix it, but first stop the application with <kbd>Ctrl</kbd>+<kbd>C</kbd> and the application and the dependencies will be terminated.

### 
[Next: Adding the streaming queue](step-6-adding-redpanda.md)