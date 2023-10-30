# Step 7: Adding LocalStack

The application is using an AWS lambda function to calculate some statistics (average and total count) for the ratings of a talk. The lambda function is invoked by the application any time the ratings for a talk are requested, using HTTP calls to the function URL of the lambda.

To enhance the developer experience of consuming this lambda function while developing the application, we will use LocalStack to emulate the AWS cloud environment locally.

LocalStack is a cloud service emulator that runs in a single container on your laptop or in your CI environment. With LocalStack, we can run your AWS applications or Lambdas entirely on your local machine without connecting to a remote cloud provider!

## Creating the lambda function

The lambda function is a simple Go function that calculates the average rating of a talk. The function is defined in the `lambda-go` directory:

```go
package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type RatingsEvent struct {
	Ratings map[string]int `json:"ratings"`
}

type Response struct {
	Avg        float64 `json:"avg"`
	TotalCount int     `json:"totalCount"`
}

var emptyResponse = Response{
	Avg:        0,
	TotalCount: 0,
}

// HandleStats returns the stats for the given talk, obtained from a call to the Lambda function.
// The payload is a JSON object with the following structure:
//
//	{
//	  "ratings": {
//	    "0": 10,
//	    "1": 20,
//	    "2": 30,
//	    "3": 40,
//	    "4": 50,
//	    "5": 60
//	  }
//	}
//
// The response from the Lambda function is a JSON object with the following structure:
//
//	{
//	   "avg": 3.5,
//	   "totalCount": 210,
//	}
func HandleStats(event events.APIGatewayProxyRequest) (Response, error) {
	ratingsEvent := RatingsEvent{}
	err := json.Unmarshal([]byte(event.Body), &ratingsEvent)
	if err != nil {
		return emptyResponse, fmt.Errorf("failed to unmarshal ratings event: %s", err)
	}

	var totalCount int
	var sum int
	for rating, count := range ratingsEvent.Ratings {
		totalCount += count

		r, err := strconv.Atoi(rating)
		if err != nil {
			return emptyResponse, fmt.Errorf("failed to convert rating %s to int: %s", rating, err)
		}

		sum += count * r
	}

	var avg float64
	if totalCount > 0 {
		avg = float64(sum) / float64(totalCount)
	}

	resp := Response{
		Avg:        avg,
		TotalCount: totalCount,
	}

	return resp, nil
}

func main() {
	lambda.Start(HandleStats)
}

```

Now, in the `lambda-go` directory, create the `go.mod` file for the lambda function:

```go
module github.com/testcontainers/workshop-go/lambda-go

go 1.20

require github.com/aws/aws-lambda-go v1.41.0

```

Now, create a Makefile in the `lambda-go` directory. It will simplify how the Go lambda is compiled and packaged as a ZIP file for being deployed to LocalStack. Please add the following content:

```Makefile
mod-tidy:
	go mod tidy

build-lambda: mod-tidy
	GOOS=linux go build -tags lambda.norpc -o bootstrap main.go

zip-lambda: build-lambda
	zip -j function.zip bootstrap

```

At this point of the workshop, we are treating the lambda as a dependency of our ratings application. In the following steps, we will see how to add integration tests for the lambda function.

Finally, to integrate the package of the lambda into the local development mode of the application, please replace the contents of the Makefile in the root of the project with the following:

```Makefile
build-lambda:
	$(MAKE) -C lambda-go zip-lambda

dev: build-lambda
	TESTCONTAINERS_RYUK_DISABLED=true go run -tags dev -v ./...

test-integration:
	go test -v -count=1 ./...

test-e2e:
	go test -v -count=1 -tags e2e ./internal/app

```

We are adding a `build-lambda` goal that will build the lambda function and package it as a ZIP file. The `dev` goal will build the lambda function and start the application in development mode. The rest of the goals are the same as before.

## Adding the LocalStack instance

Let's add a LocalStack instance using Testcontainers for Go.

1. In the `internal/app/dev_dependencies.go` file, add the following imports:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/modules/redpanda"
	"github.com/testcontainers/testcontainers-go/wait"
)
```

2. Add this function to the file:

```go
func startRatingsLambda() (testcontainers.Container, error) {
	ctx := context.Background()

	flagsFn := func() string {
		labels := testcontainers.GenericLabels()
		flags := ""
		for k, v := range labels {
			flags = fmt.Sprintf("%s -l %s=%s", flags, k, v)
		}
		return flags
	}

	c, err := localstack.RunContainer(ctx,
		testcontainers.WithImage("localstack/localstack:2.3.0"),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Env: map[string]string{
					"SERVICES":            "lambda",
					"LAMBDA_DOCKER_FLAGS": flagsFn(),
				},
				Files: []testcontainers.ContainerFile{
					{
						HostFilePath:      filepath.Join("lambda-go", "function.zip"), // path to the root of the project
						ContainerFilePath: "/tmp/function.zip",
					},
				},
			},
		}),
	)
	if err != nil {
		return nil, err
	}

	lambdaName := "localstack-lambda-url-example"

	// the three commands below are doing the following:
	// 1. create a lambda function
	// 2. create the URL function configuration for the lambda function
	// 3. wait for the lambda function to be active
	lambdaCommands := [][]string{
		{
			"awslocal", "lambda",
			"create-function", "--function-name", lambdaName,
			"--runtime", "provided.al2",
			"--handler", "bootstrap",
			"--role", "arn:aws:iam::111122223333:role/lambda-ex",
			"--zip-file", "fileb:///tmp/function.zip",
		},
		{"awslocal", "lambda", "create-function-url-config", "--function-name", lambdaName, "--auth-type", "NONE"},
		{"awslocal", "lambda", "wait", "function-active-v2", "--function-name", lambdaName},
	}
	for _, cmd := range lambdaCommands {
		_, _, err := c.Exec(ctx, cmd)
		if err != nil {
			return nil, err
		}
	}

	// 4. get the URL for the lambda function
	cmd := []string{
		"awslocal", "lambda", "list-function-url-configs", "--function-name", lambdaName,
	}
	_, reader, err := c.Exec(ctx, cmd, exec.Multiplexed())
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}

	content := buf.Bytes()

	type FunctionURLConfig struct {
		FunctionURLConfigs []struct {
			FunctionURL      string `json:"FunctionUrl"`
			FunctionArn      string `json:"FunctionArn"`
			CreationTime     string `json:"CreationTime"`
			LastModifiedTime string `json:"LastModifiedTime"`
			AuthType         string `json:"AuthType"`
		} `json:"FunctionUrlConfigs"`
	}

	v := &FunctionURLConfig{}
	err = json.Unmarshal(content, v)
	if err != nil {
		return nil, err
	}

	functionURL := v.FunctionURLConfigs[0].FunctionURL

	// replace the port with the one exposed by the container
	mappedPort, err := c.MappedPort(ctx, "4566/tcp")
	if err != nil {
		return nil, err
	}

	Connections.Lambda = strings.ReplaceAll(functionURL, "4566", mappedPort.Port())
	return c, nil
}
```

This function will:
- start a LocalStack instance, copying the zip file into the container before it starts. See the `Files` attribute in the container request.
- using the `Exec` methods of the container API, execute `awslocal lambda` commands inside the LocalStack container to:
  - create the lambda from the zip file
  - create the URL function configuration for the lambda function
  - wait for the lambda function to be active
- read the response of executing an `awslocal lambda` command to get the URL of the lambda function, parsing the JSON response to get the URL of the lambda function.
- add the URL of the lambda function to the `Connections` struct.

3. Update the comments for the init function `startupDependenciesFn` slice to include the LocalStack store:

```go
// init will be used to start up the containers for development mode. It will use
// testcontainers-go to start up the following containers:
// - Postgres: store for talks
// - Redis: store for ratings
// - Redpanda: message queue for the ratings
// - LocalStack: cloud emulator for AWS Lambdas
// All the containers will contribute their connection strings to the Connections struct.
// Please read this blog post for more information: https://www.atomicjar.com/2023/08/local-development-of-go-applications-with-testcontainers/
func init() {
```

4. Update the `startupDependenciesFn` slice to include the function that starts the ratings store:

```go
	startupDependenciesFns := []func() (testcontainers.Container, error){
		startTalksStore,
		startRatingsStore,
		startStreamingQueue,
		startRatingsLambda,
	}
```

The complete file should look like this:

```go
//go:build dev
// +build dev

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
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
// - LocalStack: cloud emulator for AWS Lambdas
// All the containers will contribute their connection strings to the Connections struct.
// Please read this blog post for more information: https://www.atomicjar.com/2023/08/local-development-of-go-applications-with-testcontainers/
func init() {
	startupDependenciesFns := []func() (testcontainers.Container, error){
		startTalksStore,
		startRatingsStore,
		startStreamingQueue,
		startRatingsLambda,
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

func startRatingsLambda() (testcontainers.Container, error) {
	ctx := context.Background()

	flagsFn := func() string {
		labels := testcontainers.GenericLabels()
		flags := ""
		for k, v := range labels {
			flags = fmt.Sprintf("%s -l %s=%s", flags, k, v)
		}
		return flags
	}

	c, err := localstack.RunContainer(ctx,
		testcontainers.WithImage("localstack/localstack:2.3.0"),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Env: map[string]string{
					"SERVICES":            "lambda",
					"LAMBDA_DOCKER_FLAGS": flagsFn(),
				},
				Files: []testcontainers.ContainerFile{
					{
						HostFilePath:      filepath.Join("lambda-go", "function.zip"), // path to the root of the project
						ContainerFilePath: "/tmp/function.zip",
					},
				},
			},
		}),
	)
	if err != nil {
		return nil, err
	}

	lambdaName := "localstack-lambda-url-example"

	// the three commands below are doing the following:
	// 1. create a lambda function
	// 2. create the URL function configuration for the lambda function
	// 3. wait for the lambda function to be active
	lambdaCommands := [][]string{
		{
			"awslocal", "lambda",
			"create-function", "--function-name", lambdaName,
			"--runtime", "provided.al2",
			"--handler", "bootstrap",
			"--role", "arn:aws:iam::111122223333:role/lambda-ex",
			"--zip-file", "fileb:///tmp/function.zip",
		},
		{"awslocal", "lambda", "create-function-url-config", "--function-name", lambdaName, "--auth-type", "NONE"},
		{"awslocal", "lambda", "wait", "function-active-v2", "--function-name", lambdaName},
	}
	for _, cmd := range lambdaCommands {
		_, _, err := c.Exec(ctx, cmd)
		if err != nil {
			return nil, err
		}
	}

	// 4. get the URL for the lambda function
	cmd := []string{
		"awslocal", "lambda", "list-function-url-configs", "--function-name", lambdaName,
	}
	_, reader, err := c.Exec(ctx, cmd, exec.Multiplexed())
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}

	content := buf.Bytes()

	type FunctionURLConfig struct {
		FunctionURLConfigs []struct {
			FunctionURL      string `json:"FunctionUrl"`
			FunctionArn      string `json:"FunctionArn"`
			CreationTime     string `json:"CreationTime"`
			LastModifiedTime string `json:"LastModifiedTime"`
			AuthType         string `json:"AuthType"`
		} `json:"FunctionUrlConfigs"`
	}

	v := &FunctionURLConfig{}
	err = json.Unmarshal(content, v)
	if err != nil {
		return nil, err
	}

	functionURL := v.FunctionURLConfigs[0].FunctionURL

	// replace the port with the one exposed by the container
	mappedPort, err := c.MappedPort(ctx, "4566/tcp")
	if err != nil {
		return nil, err
	}

	Connections.Lambda = strings.ReplaceAll(functionURL, "4566", mappedPort.Port())
	return c, nil
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

Now run `go mod tidy` from the root of the project to download the Go dependencies, only the Testcontainers for Go's LocalStack module.

Finally, stop the application with <kbd>Ctrl</kbd>+<kbd>C</kbd> and run the application again with `make dev`. This time, the application will start the Redis store and the application will be able to connect to it.

```text
TESTCONTAINERS_RYUK_DISABLED=true go run -tags dev -v ./...
# github.com/testcontainers/workshop-go

**********************************************************************************************
Ryuk has been disabled for the current execution. This can cause unexpected behavior in your environment.
More on this: https://golang.testcontainers.org/features/garbage_collector/
**********************************************************************************************
2023/10/26 12:09:37 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 23.0.6 (via Testcontainers Desktop 1.5.0)
  API Version: 1.42
  Operating System: Alpine Linux v3.18
  Total Memory: 5256 MB
  Resolved Docker Host: tcp://127.0.0.1:49342
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: daefc07421b8d6bafd1212dbe6e8e550c6fa29cac9a025b46385f75eb5e2cb57
  Test ProcessID: 884e159f-f492-41d8-b9ee-7fe78b576108
2023/10/26 12:09:37 🐳 Creating container for image postgres:15.3-alpine
2023/10/26 12:09:38 ✅ Container created: d5ec7cecb562
2023/10/26 12:09:38 🐳 Starting container: d5ec7cecb562
2023/10/26 12:09:38 ✅ Container started: d5ec7cecb562
2023/10/26 12:09:38 🚧 Waiting for container id d5ec7cecb562 image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x140003fb400 Strategies:[0x1400040b1a0]}
2023/10/26 12:09:50 🐳 Creating container for image redis:6-alpine
2023/10/26 12:09:50 ✅ Container created: bf4fcb4cd74c
2023/10/26 12:09:50 🐳 Starting container: bf4fcb4cd74c
2023/10/26 12:09:51 ✅ Container started: bf4fcb4cd74c
2023/10/26 12:09:51 🚧 Waiting for container id bf4fcb4cd74c image: redis:6-alpine. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
2023/10/26 12:09:51 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v23.1.7
2023/10/26 12:09:51 ✅ Container created: 07fb1e908b1e
2023/10/26 12:09:51 🐳 Starting container: 07fb1e908b1e
2023/10/26 12:09:52 ✅ Container started: 07fb1e908b1e
2023/10/26 12:09:53 Setting LOCALSTACK_HOST to 127.0.0.1 (to match host-routable address for container)
2023/10/26 12:09:53 🐳 Creating container for image localstack/localstack:2.3.0
2023/10/26 12:09:53 ✅ Container created: c514896580c1
2023/10/26 12:09:53 🐳 Starting container: c514896580c1
2023/10/26 12:09:53 ✅ Container started: c514896580c1
2023/10/26 12:09:53 🚧 Waiting for container id c514896580c1 image: localstack/localstack:2.3.0. Waiting for: &{timeout:0x14000369ca0 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x1024f5090 ResponseMatcher:0x1025c66a0 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> PollInterval:100ms UserInfo:}
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

In the second terminal, check the containers, we will see the LocalStack instance is running alongside the Postgres database, the Redis store and the Redpanda streaming queue:

```text
$ docker ps
CONTAINER ID   IMAGE                                               COMMAND                  CREATED         STATUS                   PORTS                                                                                                                                             NAMES
c514896580c1   localstack/localstack:2.3.0                         "docker-entrypoint.sh"   2 minutes ago   Up 2 minutes (healthy)   4510-4559/tcp, 5678/tcp, 0.0.0.0:32792->4566/tcp, :::32792->4566/tcp                                                                              priceless_antonelli
07fb1e908b1e   docker.redpanda.com/redpandadata/redpanda:v23.1.7   "/entrypoint-tc.sh r…"   3 minutes ago   Up 3 minutes             8082/tcp, 0.0.0.0:32791->8081/tcp, :::32791->8081/tcp, 0.0.0.0:32790->9092/tcp, :::32790->9092/tcp, 0.0.0.0:32789->9644/tcp, :::32789->9644/tcp   loving_murdock
bf4fcb4cd74c   redis:6-alpine                                      "docker-entrypoint.s…"   3 minutes ago   Up 3 minutes             0.0.0.0:32788->6379/tcp, :::32788->6379/tcp                                                                                                       angry_shirley
d5ec7cecb562   postgres:15.3-alpine                                "docker-entrypoint.s…"   3 minutes ago   Up 3 minutes             0.0.0.0:32787->5432/tcp, :::32787->5432/tcp                                                                                                       laughing_kare
```

The LocalStack instance is now running, and a lambda function is deployed in it. We can verify the lambda function is running by sending a request to the function URL. But we first need to obtain the URL of the lambda. Please do a GET request to the `/` endpoint of the API, where we'll get the metadata of the application. Something similar to this:

```bash
$ curl -X GET http://localhost:8080/
```

The JSON response:

```json
{"metadata":{"ratings_lambda":"http://bwtiue69l3njrfnm2v27qgql2n0dwbew.lambda-url.us-east-1.localhost.localstack.cloud:32773/","ratings":"redis://127.0.0.1:32769","streams":"127.0.0.1:32771","talks":"postgres://postgres:postgres@127.0.0.1:32768/talks-db?"}}
```

In your terminal, copy the `ratings_lambda` URL from the response and send a POST request to it with `curl` (please remember to replace the URL with the one we got from the response):

```bash
curl -X POST http://bwtiue69l3njrfnm2v27qgql2n0dwbew.lambda-url.us-east-1.localhost.localstack.cloud:32773/ -d '{"ratings":{"2":1,"4":3,"5":1}}' -H "Content-Type: application/json"
```

The JSON response:

```json
{"avg": 3.8, "totalCount": 5}%
```

Great! the response contains the average rating of the talk, and the total number of ratings, calculated in the lambda function.

### 
[Next](step-8-adding-integration-tests.md)