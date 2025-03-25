# Step 4: Dev mode with Testcontainers

Remember the Makefile in the root of the project with the `dev` target, the one that starts the application in `local dev mode`? We are going to learn in this workshop how to leverage Go's build tags and init functions to selectively execute code when a `dev` tag is passed to the Go toolchain, only while developing our application. So when the application is started, it will start the runtime dependencies as Docker containers, leveraging Testcontainers for Go.

To understand how the `local dev mode` with Testcontainers for Go works, please read the following blog post: https://www.docker.com/blog/local-development-of-go-applications-with-testcontainers/

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
	"path/filepath"
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

	for _, fn := range startupDependenciesFns {
		_, err := fn()
		if err != nil {
			panic(err)
		}
	}
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

Let's understand what we have done here:

- The first two lines include the build tag `dev` and the build constraint `+build dev`. This means that the code in this file will only be compiled when the `dev` tag is passed to the Go toolchain.
- The `init` function will be executed when the application starts. It will start the runtime dependencies as Docker containers, leveraging Testcontainers for Go.
- The `init` function contains a `startupDependenciesFns` slice with the functions that will start the containers. In this case, we only have one function, `startTalksStore`.
- The `startTalksStore` function will start a Postgres database with the `testdata/dev-db.sql` file as initialization script.
- The `Connections.Talks` variable receives the connection string used to connect to the database. The code is overriding the default connection string for the database, which is read from an environment variable (see `internal/app/metadata.go`).

Now run `go mod tidy` from the root of the project to download the Go dependencies.

## Update the make dev target

The `make dev` target in the Makefile is using the `go run` command to start the application. We need to pass the `dev` build tag to the Go toolchain, so the `init` function in `internal/app/dev_dependencies.go` is executed.

Update the `make dev` target in the Makefile to pass the `dev` build tag:

```makefile
dev:
	go run -tags dev -v ./...
```

Finally, stop the application with <kbd>Ctrl</kbd>+<kbd>C</kbd> and run the application again with `make dev`. This time, the application will start the Postgres database and the application will be able to connect to it.

```text
go run -tags dev -v ./...
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

Now it seems the application is able to connect to the database, but not to Redis. Let's fix it, but first stop the application with <kbd>Ctrl</kbd>+<kbd>C</kbd>, so the application and the dependencies are terminated.

Let's add Redis as a dependency in development mode.

### 
[Next: Adding Redis](step-5-adding-redis.md)
