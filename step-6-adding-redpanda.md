# Step 6: Adding Redpanda

When the application started, it failed because we need to connect to a message queue before we can add the ratings for a talk.

Let's add a Redpanda instance using Testcontainers for Go.

1. In the `internal/app/dev_dependencies.go` file, add the following imports:

```go
import (
	"context"
	"path/filepath"
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
	"path/filepath"
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
// - Redpanda: message queue for the ratings
// All the containers will contribute their connection strings to the Connections struct.
// Please read this blog post for more information: https://www.atomicjar.com/2023/08/local-development-of-go-applications-with-testcontainers/
func init() {
	startupDependenciesFns := []func() (testcontainers.Container, error){
		startTalksStore,
		startRatingsStore,
		startStreamingQueue,
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

Now run `go mod tidy` from the root of the project to download the Go dependencies, only the Testcontainers for Go's Redpanda module.

Finally, stop the application with <kbd>Ctrl</kbd>+<kbd>C</kbd> and run the application again with `make dev`. This time, the application will start the Redis store and the application will be able to connect to it.

```text
go run -tags dev -v ./...
# github.com/testcontainers/workshop-go

2023/10/26 11:43:59 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 23.0.6 (via Testcontainers Desktop 1.5.0)
  API Version: 1.42
  Operating System: Alpine Linux v3.18
  Total Memory: 5256 MB
  Resolved Docker Host: tcp://127.0.0.1:49342
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: e13901151c68b9c4fac245114619b7caf8fea93050aadd29152276b91fce6330
  Test ProcessID: 598de86d-1870-42c4-ad0e-2d9c9b676fc4
2023/10/26 11:43:59 🐳 Creating container for image postgres:15.3-alpine
2023/10/26 11:43:59 ✅ Container created: 00bca83e66ca
2023/10/26 11:43:59 🐳 Starting container: 00bca83e66ca
2023/10/26 11:44:00 ✅ Container started: 00bca83e66ca
2023/10/26 11:44:00 🚧 Waiting for container id 00bca83e66ca image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x1400043d420 Strategies:[0x1400044f1d0]}
2023/10/26 11:44:11 🐳 Creating container for image redis:6-alpine
2023/10/26 11:44:11 ✅ Container created: 373f523c83ac
2023/10/26 11:44:11 🐳 Starting container: 373f523c83ac
2023/10/26 11:44:12 ✅ Container started: 373f523c83ac
2023/10/26 11:44:12 🚧 Waiting for container id 373f523c83ac image: redis:6-alpine. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
2023/10/26 11:44:12 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v23.1.7
2023/10/26 11:44:12 ✅ Container created: 1811a3de1f8f
2023/10/26 11:44:12 🐳 Starting container: 1811a3de1f8f
2023/10/26 11:44:13 ✅ Container started: 1811a3de1f8f
[GIN-debug] [WARNING] Creating an Engine instance with the Logger and Recovery middleware already attached.

[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
 - using env:   export GIN_MODE=release
 - using code:  gin.SetMode(gin.ReleaseMode)

[GIN-debug] GET    /                         --> github.com/testcontainers/workshop-go/internal/app.Root (3 handlers)
[GIN-debug] GET    /ratings                  --> github.com/testcontainers/workshop-go/internal/app.Ratings (3 handlers)
[GIN-debug] POST   /ratings                  --> github.com/testcontainers/workshop-go/internal/app.AddRating (3 handlers)
[GIN-debug] [WARNING] You trusted all proxies, this is NOT safe. We recommend you to set a value.
Please check https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies for details.
[GIN-debug] Listening and serving HTTP on :8080
```

In the second terminal, check the containers, we will see the Redpanda streaming queue is running alongside the Postgres database and the Redis store:

```text
$ docker ps
CONTAINER ID   IMAGE                                               COMMAND                  CREATED         STATUS         PORTS                                                                                                                                             NAMES
1811a3de1f8f   docker.redpanda.com/redpandadata/redpanda:v23.1.7   "/entrypoint-tc.sh r…"   3 minutes ago   Up 3 minutes   8082/tcp, 0.0.0.0:32781->8081/tcp, :::32781->8081/tcp, 0.0.0.0:32780->9092/tcp, :::32780->9092/tcp, 0.0.0.0:32779->9644/tcp, :::32779->9644/tcp   elegant_goldberg
373f523c83ac   redis:6-alpine                                      "docker-entrypoint.s…"   3 minutes ago   Up 3 minutes   0.0.0.0:32778->6379/tcp, :::32778->6379/tcp                                                                                                       stupefied_franklin
00bca83e66ca   postgres:15.3-alpine                                "docker-entrypoint.s…"   3 minutes ago   Up 3 minutes   0.0.0.0:32777->5432/tcp, :::32777->5432/tcp                                                                                                       wizardly_snyder
```

Now the application should be able to connect to the database, to Redis and to the Redpanda streaming queue. Let's try to send a POST request adding a rating for the talk. If we remember, the API accepted a JSON payload with the following format:

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

The response should be a 200 OK:

```json
{"rating":{"talk_uuid":"testcontainers-integration-testing","value":5}}%
```

The log entry for the POST request:

```text
[GIN] 2023/10/26 - 11:48:25 | 200 |  597.468542ms |       127.0.0.1 | POST     "/ratings"
```

If we open now the ratings endpoint from the API (http://localhost:8080/ratings?talkId=testcontainers-integration-testing), then a 200 OK response code is returned, and the first ratings for the given talk is there. It was a five! ⭐️⭐️⭐️⭐️⭐️

```text
{"ratings":{"5":"1"}}
```

With `curl`:

```shell
curl -X GET http://localhost:8080/ratings\?talkId\=testcontainers-integration-testing                                                         
{"ratings":{"5":"1"}}%
```

Play around sending multiple POST requests for the two talks, and check the histogram that is created for the different rating values.

in any POST request we'll still see the log entry for the AWS lambda failing to be called.

```text
2023/10/26 11:48:42 error calling lambda function: Post "": unsupported protocol scheme ""
```

It's time now to fix it, adding a cloud emulator for the AWS Lambda function.

### 
[Next: Adding Localstack](step-7-adding-localstack.md)