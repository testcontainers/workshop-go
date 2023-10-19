# Step 5: Adding Redis

When the application started, it failed because we need to connect to a Redis database before we can do anything useful with the ratings.

Let's add a Redis instance using Testcontainers for Go.

1. Add the following `internal/app/dev_dependencies.go` file, add the following imports:

```go
import (
       "context"
       "fmt"
       "os"
       "os/signal"
       "path/filepath"
       "syscall"
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

       c, err := redis.RunContainer(ctx, testcontainers.WithImage("redis:6-alpine"))
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
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
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

	runtimeDependencies := make([]testcontainers.Container, 0, len(startupDependenciesFns))

	for _, fn := range startupDependenciesFns {
		c, err := fn()
		if err != nil {
			panic(err)
		}
		runtimeDependencies = append(runtimeDependencies, c)
	}

	// register a graceful shutdown to stop the dependencies when the application is stopped
	// only in development mode
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go func() {
		// also use the shutdown function when the SIGTERM or SIGINT signals are received
		sig := <-gracefulStop
		fmt.Printf("caught sig: %+v\n", sig)
		err := shutdownDependencies(runtimeDependencies...)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}()
}

// helper function to stop the dependencies
func shutdownDependencies(containers ...testcontainers.Container) error {
	ctx := context.Background()
	for _, c := range containers {
		err := c.Terminate(ctx)
		if err != nil {
			return fmt.Errorf("failed to terminate container: %w", err)
		}
	}

	return nil
}

func startRatingsStore() (testcontainers.Container, error) {
	ctx := context.Background()

	c, err := redis.RunContainer(ctx, testcontainers.WithImage("redis:6-alpine"))
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
	c, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15.3-alpine"),
		postgres.WithInitScripts(filepath.Join(".", "testdata", "dev-db.sql")),
		postgres.WithDatabase("talks-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
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

Finally, run the application again with `make dev`. This time, the application will start the Redis store and the application will be able to connect to it.

```text
TESTCONTAINERS_RYUK_DISABLED=true go run -tags dev -v ./...
github.com/testcontainers/workshop-go/internal/app
github.com/testcontainers/workshop-go

**********************************************************************************************
Ryuk has been disabled for the current execution. This can cause unexpected behavior in your environment.
More on this: https://golang.testcontainers.org/features/garbage_collector/
**********************************************************************************************
2023/10/19 14:30:14 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 78+testcontainerscloud (via Testcontainers Desktop 1.4.18)
  API Version: 1.43
  Operating System: Ubuntu 20.04 LTS
  Total Memory: 7407 MB
  Resolved Docker Host: tcp://127.0.0.1:62250
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 4f01a36717f20795574c5a04da2025224377b8e0e7e2a99459e73f30de5bbef6
  Test ProcessID: ba950bae-881a-45b8-909a-9ca627e8ac09
2023/10/19 14:30:24 🐳 Creating container for image postgres:15.3-alpine
2023/10/19 14:30:24 ✅ Container created: 622ab97d4266
2023/10/19 14:30:24 🐳 Starting container: 622ab97d4266
2023/10/19 14:30:24 ✅ Container started: 622ab97d4266
2023/10/19 14:30:24 🚧 Waiting for container id 622ab97d4266 image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x140002f7b08 Strategies:[0x14000499950]}
2023/10/19 14:30:29 🐳 Creating container for image redis:6-alpine
2023/10/19 14:30:29 ✅ Container created: f77dfdf91c37
2023/10/19 14:30:29 🐳 Starting container: f77dfdf91c37
2023/10/19 14:30:30 ✅ Container started: f77dfdf91c37
2023/10/19 14:30:30 🚧 Waiting for container id f77dfdf91c37 image: redis:6-alpine. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] Loaded HTML Templates (5): 
        - 
        - error.tmpl
        - metadata.tmpl
        - ratings-add.tmpl
        - ratings-list.tmpl

[GIN-debug] GET    /                         --> github.com/testcontainers/workshop-go/internal/app.Root (3 handlers)
[GIN-debug] GET    /ratings                  --> github.com/testcontainers/workshop-go/internal/app.Ratings (3 handlers)
[GIN-debug] POST   /ratings                  --> github.com/testcontainers/workshop-go/internal/app.AddRating (3 handlers)
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on :8080
```

If the second terminal, check the containers, you will see the Redis store running alongside the Postgres database:

```text
$ docker ps
CONTAINER ID   IMAGE                  COMMAND                  CREATED              STATUS              PORTS                                         NAMES
f77dfdf91c37   redis:6-alpine         "docker-entrypoint.s…"   About a minute ago   Up About a minute   0.0.0.0:32769->6379/tcp, :::32769->6379/tcp   laughing_wiles
622ab97d4266   postgres:15.3-alpine   "docker-entrypoint.s…"   About a minute ago   Up About a minute   0.0.0.0:32768->5432/tcp, :::32768->5432/tcp   vigilant_ptolemy
```

If you open now the ratings endpoint from the API (http://localhost:8080/ratings?talkId=testcontainers-integration-testing), then a 200 OK response code is returned, but there are no ratings for the given talk:

```text
map[]
```

Now it seems the application is able to connect to the database, and to Redis. Let's try to send a POST request adding a rating for the talk. If you remember, the API accepted a JSON payload with the following format:

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

```text
Unable to ping the streams: unable to dial: dial tcp :9092: connect: connection refused
[GIN] 2023/10/19 - 14:40:04 | 500 |   64.879083ms |       127.0.0.1 | POST     "/ratings"
```

If you recall correctly, the application was using a message queue to send the ratings before storing them in Redis (see `internal/app/handlers.go`), so we need to add a message queue for that. Let's fix it.

### 
[Next](step-6-adding-redpanda.md)