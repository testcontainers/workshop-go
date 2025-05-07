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

 ┌───────────────────────────────────────────────────┐ 
 │                   Fiber v2.52.6                   │ 
 │               http://127.0.0.1:8080               │ 
 │       (bound on host 0.0.0.0 and port 8080)       │ 
 │                                                   │ 
 │ Handlers ............. 5  Processes ........... 1 │ 
 │ Prefork ....... Disabled  PID ............. 17433 │ 
 └───────────────────────────────────────────────────┘ 


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
Unable to connect to database: failed to connect to `host=/private/tmp user=mdelapenya database=`: dial error (dial unix /private/tmp/.s.PGSQL.5432: connect: no such file or directory)
13:03:27 | 500 |     4.20625ms | 127.0.0.1 | GET | /ratings | -
```

It seems the application is not able to connect to the database. Let's fix it.

### 
[Next: Dev Mode with Testcontainers](step-4-dev-mode-with-testcontainers.md)