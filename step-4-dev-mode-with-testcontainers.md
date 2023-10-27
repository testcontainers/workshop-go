# Step 4: Dev mode with Testcontainers

Remember the Makefile in the root of the project with the `dev` target, the one that starts the application in `local dev mode`? We are going to learn in this workshop how to leverage Go's build tags and init functions to selectively execute code when a `dev` tag is passed to the Go toolchain, only while developing our application. So when the application is started, it will start the runtime dependencies as Docker containers, leveraging Testcontainers for Go.

To understand how the `local dev mode` with Testcontainers for Go works, please read the following blog post: https://www.atomicjar.com/2023/08/local-development-of-go-applications-with-testcontainers/

## Adding Talks store

When the application started, it failed because we need to connect to a Postgres database including some data before we can do anything useful with the talks.

Let's add a `testdata/dev-db.sql` file with the following content:

```sql
CREATE TABLE IF NOT EXISTS talks (
  id serial,
  uuid varchar(255),
  title varchar(255)
);

INSERT
  INTO talks (uuid, title)
  VALUES ('testcontainers-integration-testing', 'Modern Integration Testing with Testcontainers')
  ON CONFLICT do nothing;

INSERT
  INTO talks (uuid, title)
  VALUES ('flight-of-the-gopher', 'A look at Go scheduler')
  ON CONFLICT do nothing;

```

Also create an `internal/app/dev_dependencies.go` file with the following content:

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
	"github.com/testcontainers/testcontainers-go/wait"
)

// init will be used to start up the containers for development mode. It will use
// testcontainers-go to start up the following containers:
// - Redis: store for ratings
// All the containers will contribute their connection strings to the Connections struct.
// Please read this blog post for more information: https://www.atomicjar.com/2023/08/local-development-of-go-applications-with-testcontainers/
func init() {
	startupDependenciesFns := []func() (testcontainers.Container, error){
		startTalksStore,
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

Let's understand what we have done here:

- The first two lines include the build tag `dev` and the build constraint `+build dev`. This means that the code in this file will only be compiled when the `dev` tag is passed to the Go toolchain.
- The `init` function will be executed when the application starts. It will start the runtime dependencies as Docker containers, leveraging Testcontainers for Go.
- The `init` function contains a `startupDependenciesFns` slice with the functions that will start the containers. In this case, we only have one function, `startTalksStore`.
- The `init` function also contains a `gracefulStop` channel to stop the dependencies when the application is stopped.
- The `shutdownDependencies` function will stop the dependencies when the application is stopped.
- The `startTalksStore` function will start a Postgres database with the `testdata/dev-db.sql` file as initialization script.
- The `Connections.Talks` variable receives the connection string used to connect to the database. The code is overriding the default connection string for the database, which is read from an environment variable (see `internal/app/metadata.go`).

Now run `go mod tidy` from the root of the project to download the Go dependencies.

## Update the make dev target

The `make dev` target in the Makefile is using the `go run` command to start the application. We need to pass the `dev` build tag to the Go toolchain, so the `init` function in `internal/app/dev_dependencies.go` is executed.

Update the `make dev` target in the Makefile to pass the `dev` build tag:

```makefile
dev:
	TESTCONTAINERS_RYUK_DISABLED=true go run -tags dev -v ./...
```

We need to disable Ryuk in development mode, because we are starting the containers from the application, and Ryuk will try to stop them at some point, which will make the application fail as the database will be stopped. To know more about Ryuk as the resource reaper, please read https://golang.testcontainers.org/features/garbage_collector/#ryuk.

Finally, stop the application with <kbd>Ctrl</kbd>+<kbd>C</kbd> and run the application again with `make dev`. This time, the application will start the Postgres database and the application will be able to connect to it.

```text
TESTCONTAINERS_RYUK_DISABLED=true go run -tags dev -v ./...

**********************************************************************************************
Ryuk has been disabled for the current execution. This can cause unexpected behavior in your environment.
More on this: https://golang.testcontainers.org/features/garbage_collector/
**********************************************************************************************
2023/10/26 11:24:40 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 23.0.6 (via Testcontainers Desktop 1.5.0)
  API Version: 1.42
  Operating System: Alpine Linux v3.18
  Total Memory: 5256 MB
  Resolved Docker Host: tcp://127.0.0.1:49342
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 81b67cdfeb4575f43b46473fcf4b211e01e4729370afd2fe7bfe697183890bf5
  Test ProcessID: c759c04a-3f04-427f-a91d-95dbe8ed3b3c
2023/10/26 11:24:40 🐳 Creating container for image postgres:15.3-alpine
2023/10/26 11:24:40 ✅ Container created: 2d5155cb8e58
2023/10/26 11:24:40 🐳 Starting container: 2d5155cb8e58
2023/10/26 11:24:41 ✅ Container started: 2d5155cb8e58
2023/10/26 11:24:41 🚧 Waiting for container id 2d5155cb8e58 image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x140003a3470 Strategies:[0x140003bd1a0]}
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

If we open a second terminal and check the containers, we will see the Postgres database running:

```text
$ docker ps
CONTAINER ID   IMAGE                  COMMAND                  CREATED          STATUS          PORTS                                         NAMES
2d5155cb8e58   postgres:15.3-alpine   "docker-entrypoint.s…"   36 seconds ago   Up 35 seconds   0.0.0.0:32771->5432/tcp, :::32771->5432/tcp   gifted_villani
```

On the contrary, if we open again the ratings endpoint from the API (http://localhost:8080/ratings?talkId=testcontainers-integration-testing), we'll still get a 500 error, but with a different message:

```text
{"message":"redis: invalid URL scheme: "}
```

Now it seems the application is able to connect to the database, but not to Redis. Let's fix it, but first stop the application with <kbd>Ctrl</kbd>+<kbd>C</kbd>, so the application and the dependencies are terminated by the signals we added in the `init` function of the `internal/app/dev_dependencies.go` file.

Let's add Redis as a dependency in development mode.

### 
[Next](step-5-adding-redis.md)