# Step 6: Adding Redpanda

When the application started, it failed because you need to connect to a message queue before you can adds the ratings for a talk.

Let's add a Redpanda instance using Testcontainers for Go.

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
       "github.com/testcontainers/testcontainers-go/modules/redpanda"
       "github.com/testcontainers/testcontainers-go/wait"
)
```

2. Add this function to the file:

```go
func startStreamingQueue() (testcontainers.Container, error) {
       ctx := context.Background()

       c, err := redpanda.RunContainer(
               ctx,
               testcontainers.WithImage("docker.redpanda.com/redpandadata/redpanda:v23.1.7"),
               redpanda.WithAutoCreateTopics(),
       )

       seedBroker, err := c.KafkaSeedBroker(ctx)
       if err != nil {
               return nil, err
       }

       Connections.Streams = seedBroker
       return c, nil
}
```

3. Update the comments for the init function `startupDependenciesFn` slice to include the Redpanda queue:

```go
// init will be used to start up the containers for development mode. It will use
// testcontainers-go to start up the following containers:
// - Postgres: store for talks
// - Redis: store for ratings
// - Redpanda: message queue for the ratings
// All the containers will contribute their connection strings to the Connections struct.
// Please read this blog post for more information: https://www.atomicjar.com/2023/08/local-development-of-go-applications-with-testcontainers/
func init() {
```

4. Update the `startupDependenciesFn` slice to include the function that starts the streaming queue:

```go
	startupDependenciesFns := []func() (testcontainers.Container, error){
		startTalksStore,
		startRatingsStore,
		startStreamingQueue,
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
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	"github.com/testcontainers/testcontainers-go/wait"
)

// init will be used to start up the containers for development mode. It will use
// testcontainers-go to start up the following containers:
// - Postgres: store for talks
// - Redis: store for ratings
// - Redpanda: streaming queue for ratings
// All the containers will contribute their connection strings to the Connections struct.
// Please read this blog post for more information: https://www.atomicjar.com/2023/08/local-development-of-go-applications-with-testcontainers/
func init() {
	startupDependenciesFns := []func() (testcontainers.Container, error){
		startTalksStore,
		startRatingsStore,
		startStreamingQueue,
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

func startStreamingQueue() (testcontainers.Container, error) {
	ctx := context.Background()

	c, err := redpanda.RunContainer(
		ctx,
		testcontainers.WithImage("docker.redpanda.com/redpandadata/redpanda:v23.1.7"),
		redpanda.WithAutoCreateTopics(),
	)

	seedBroker, err := c.KafkaSeedBroker(ctx)
	if err != nil {
		return nil, err
	}

	Connections.Streams = seedBroker
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

Now run `go mod tidy` from the root of the project to download the Go dependencies, only the Testcontainers for Go's Redpanda module.

Finally, run the application again with `make dev`. This time, the application will start the Redis store and the application will be able to connect to it.

```text
TESTCONTAINERS_RYUK_DISABLED=true go run -tags dev -v ./...
github.com/testcontainers/workshop-go/internal/app
github.com/testcontainers/workshop-go

**********************************************************************************************
Ryuk has been disabled for the current execution. This can cause unexpected behavior in your environment.
More on this: https://golang.testcontainers.org/features/garbage_collector/
**********************************************************************************************
2023/10/19 14:51:45 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 78+testcontainerscloud (via Testcontainers Desktop 1.4.18)
  API Version: 1.43
  Operating System: Ubuntu 20.04 LTS
  Total Memory: 7407 MB
  Resolved Docker Host: tcp://127.0.0.1:62250
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 05d4b52d5c0529fb2af88b26a13b0e55a0294ddbcb8052253460ad5df41173ad
  Test ProcessID: 99603e83-9b75-4e90-bc91-7a17665cd9f5
2023/10/19 14:51:45 🐳 Creating container for image postgres:15.3-alpine
2023/10/19 14:51:45 ✅ Container created: 622b614402ce
2023/10/19 14:51:45 🐳 Starting container: 622b614402ce
2023/10/19 14:51:45 ✅ Container started: 622b614402ce
2023/10/19 14:51:45 🚧 Waiting for container id 622b614402ce image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x1400012a728 Strategies:[0x1400012f6b0]}
2023/10/19 14:51:47 🐳 Creating container for image redis:6-alpine
2023/10/19 14:51:47 ✅ Container created: e990535b05ba
2023/10/19 14:51:47 🐳 Starting container: e990535b05ba
2023/10/19 14:51:47 ✅ Container started: e990535b05ba
2023/10/19 14:51:47 🚧 Waiting for container id e990535b05ba image: redis:6-alpine. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
2023/10/19 14:51:47 Failed to get image auth for docker.redpanda.com. Setting empty credentials for the image: docker.redpanda.com/redpandadata/redpanda:v23.1.7. Error is:credentials not found in native keychain
2023/10/19 14:51:58 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v23.1.7
2023/10/19 14:51:59 ✅ Container created: 721158044e98
2023/10/19 14:51:59 🐳 Starting container: 721158044e98
2023/10/19 14:51:59 ✅ Container started: 721158044e98
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] Loaded HTML Templates (5): 
        - 
        - metadata.tmpl

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
CONTAINER ID   IMAGE                                               COMMAND                  CREATED              STATUS              PORTS                                                                                                                                             NAMES
721158044e98   docker.redpanda.com/redpandadata/redpanda:v23.1.7   "/entrypoint-tc.sh r…"   54 seconds ago       Up 53 seconds       8082/tcp, 0.0.0.0:32780->8081/tcp, :::32780->8081/tcp, 0.0.0.0:32779->9092/tcp, :::32779->9092/tcp, 0.0.0.0:32778->9644/tcp, :::32778->9644/tcp   upbeat_mendel
e990535b05ba   redis:6-alpine                                      "docker-entrypoint.s…"   About a minute ago   Up About a minute   0.0.0.0:32777->6379/tcp, :::32777->6379/tcp                                                                                                       busy_brahmagupta
622b614402ce   postgres:15.3-alpine                                "docker-entrypoint.s…"   About a minute ago   Up About a minute   0.0.0.0:32776->5432/tcp, :::32776->5432/tcp                                                                                                       infallible_kare
```

Now it seems the application is able to connect to the database, to Redis and to the Redpanda streaming queue. Let's try to send a POST request adding a rating for the talk. If you remember, the API accepted a JSON payload with the following format:

```json
{
  "talkId": "testcontainers-integration-testing",
  "value": 5
}
```

In a terminal, let's send a POST request with `curl`:

```shell
curl -X POST -H "Content-Type: application/json" http://localhost:8080/ratings -d '{"talkId":"testcontainers-integration-testing", "value":5}'
{"rating":{"talk_uuid":"testcontainers-integration-testing","value":5}}%  
```

The log entry for the POST request:

```text
[GIN] 2023/10/19 - 14:53:33 | 200 |     550.807ms |       127.0.0.1 | POST     "/ratings"
```

If you open now the ratings endpoint from the API (http://localhost:8080/ratings?talkId=testcontainers-integration-testing), then a 200 OK response code is returned, and the first ratings for the given talk is there. The `5` rating has been voted `1` time!

```text
map[5:1]
```

Play around sending multiple POST requests for the two talks, and check the histogram that is created for the different rating values.

### 
[Next](step-7-adding-integration-tests.md)