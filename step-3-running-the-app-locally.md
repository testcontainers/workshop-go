# Step 3: Running the application locally

Go applications are usually started with `go run` while in development mode.

In order to simplify the experience of running the application locally, there is a Makefile in the root of the project with the `dev` target. This target starts the application in `local dev mode`, and it will basically be the entrypoint to start the application.

## Start the application

In a terminal, run the following command:

```bash
make dev
```

A similar output log will appear:

```text
go run ./...
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

If we open the browser in the URL http://localhost:8080, we will see the metadata of the application, but all the values are empty:

```json
{"metadata":{"ratings_lambda":"","ratings":"","streams":"","talks":""}}
```

On the contrary, if we open the ratings endpoint from the API (http://localhost:8080/ratings?talkId=testcontainers-integration-testing), we will get a 500 error and a similar message:

```text
{"message":"failed to connect to `host=/private/tmp user=mdelapenya database=`: dial error (dial unix /private/tmp/.s.PGSQL.5432: connect: no such file or directory)"}
```

The logs will show the following:

```text
Unable to connect to database: failed to connect to `user=mdelapenya database=`: /private/tmp/.s.PGSQL.5432 (/private/tmp): dial error: dial unix /private/tmp/.s.PGSQL.5432: connect: no such file or directory
[GIN] 2025/03/25 - 12:49:11 | 500 |   10.338208ms |             ::1 | GET      "/ratings?talkId=testcontainers-integration-testing"
```

It seems the application is not able to connect to the database. Let's fix it.

### 
[Next: Dev Mode with Testcontainers](step-4-dev-mode-with-testcontainers.md)