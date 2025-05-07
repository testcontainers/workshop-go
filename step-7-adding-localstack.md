# Step 7: Adding LocalStack

The application is using an AWS lambda function to calculate some statistics (average and total count) for the ratings of a talk. The lambda function is invoked by the application any time the ratings for a talk are requested, using HTTP calls to the function URL of the lambda.

To enhance the developer experience of consuming this lambda function while developing the application, we will use LocalStack to emulate the AWS cloud environment locally.

LocalStack is a cloud service emulator that runs in a single container on your laptop or in your CI environment. With LocalStack, we can run your AWS applications or Lambdas entirely on your local machine without connecting to a remote cloud provider!

## Creating the lambda function

The lambda function is a simple Go function that calculates the average rating of a talk. The function is defined in the `main.go` file under a `lambda-go` directory:

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

go 1.24

require github.com/aws/aws-lambda-go v1.48.0

```

Now, create a Makefile in the `lambda-go` directory. It will simplify how the Go lambda is compiled and packaged as a ZIP file for being deployed to LocalStack. Please add the following content:

```Makefile
mod-tidy:
	go mod tidy

build-lambda: mod-tidy
	# If you are using Testcontainers Cloud, please add 'GOARCH=amd64' in order to get the localstack's lambdas using the right architecture
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
	go run -tags dev -v ./...

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
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

2. Add these two functions to the file:

```go
// buildLambda return the path to the ZIP file used to deploy the lambda function.
func buildLambda() string {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)

	lambdaPath := filepath.Join(basepath, "..", "..", "lambda-go")

	makeCmd := osexec.Command("make", "zip-lambda")
	makeCmd.Dir = lambdaPath

	err := makeCmd.Run()
	if err != nil {
		panic(fmt.Errorf("failed to zip lambda: %w", err))
	}

	return filepath.Join(lambdaPath, "function.zip")
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

	var functionURL string

	c, err := localstack.Run(ctx,
		"localstack/localstack:latest",
		testcontainers.WithEnv(map[string]string{
			"SERVICES":            "lambda",
			"LAMBDA_DOCKER_FLAGS": flagsFn(),
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      buildLambda(),
			ContainerFilePath: "/tmp/function.zip",
		}),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
					{
						PostStarts: []testcontainers.ContainerHook{
							func(ctx context.Context, c testcontainers.Container) error {
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
										return err
									}
								}

								// 4. get the URL for the lambda function
								cmd := []string{
									"awslocal", "lambda", "list-function-url-configs", "--function-name", lambdaName,
								}
								_, reader, err := c.Exec(ctx, cmd, exec.Multiplexed())
								if err != nil {
									return err
								}

								buf := new(bytes.Buffer)
								_, err = buf.ReadFrom(reader)
								if err != nil {
									return err
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
									return err
								}

								// 5. finally, set the function URL from the response
								functionURL = v.FunctionURLConfigs[0].FunctionURL

								return nil
							},
						},
					},
				},
			},
		}),
	)
	if err != nil {
		return nil, err
	}

	// replace the port with the one exposed by the container
	mappedPort, err := c.MappedPort(ctx, "4566/tcp")
	if err != nil {
		return nil, err
	}

	functionURL = strings.ReplaceAll(functionURL, "4566", mappedPort.Port())

	// The latest version of localstack does not add ".localstack.cloud" by default,
	// that's why need to add it to the URL.
	functionURL = strings.ReplaceAll(functionURL, ".localhost", ".localhost.localstack.cloud")

	Connections.Lambda = functionURL

	return c, nil
}
```

The first function will perform a `make zip-lambda` to build the lambda function and package it as a ZIP file, which is convenient to integrate the `Make` build system into the local development experience.

The second function will:
- start a LocalStack instance, copying the zip file into the container before it starts. See the `Files` attribute in the container request.
- leverate the container lifecycle hooks to execute commands in the container right after it has started. We are going to execute `awslocal lambda` commands inside the LocalStack container to:
  - create the lambda from the zip file
  - create the URL function configuration for the lambda function
  - wait for the lambda function to be active
  - read the response of executing an `awslocal lambda` command to get the URL of the lambda function, parsing the JSON response to get the URL of the lambda function.
  - finally store the URL of the lambda function in a variable
- update the `Connections` struct with the lambda function URL.

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
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

	for _, fn := range startupDependenciesFns {
		_, err := fn()
		if err != nil {
			panic(err)
		}
	}
}

// buildLambda return the path to the ZIP file used to deploy the lambda function.
func buildLambda() string {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)

	lambdaPath := filepath.Join(basepath, "..", "..", "lambda-go")

	makeCmd := osexec.Command("make", "zip-lambda")
	makeCmd.Dir = lambdaPath

	err := makeCmd.Run()
	if err != nil {
		panic(fmt.Errorf("failed to zip lambda: %w", err))
	}

	return filepath.Join(lambdaPath, "function.zip")
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

	var functionURL string

	c, err := localstack.Run(ctx,
		"localstack/localstack:latest",
		testcontainers.WithEnv(map[string]string{
			"SERVICES":            "lambda",
			"LAMBDA_DOCKER_FLAGS": flagsFn(),
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      buildLambda(),
			ContainerFilePath: "/tmp/function.zip",
		}),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				LifecycleHooks: []testcontainers.ContainerLifecycleHooks{
					{
						PostStarts: []testcontainers.ContainerHook{
							func(ctx context.Context, c testcontainers.Container) error {
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
										return err
									}
								}

								// 4. get the URL for the lambda function
								cmd := []string{
									"awslocal", "lambda", "list-function-url-configs", "--function-name", lambdaName,
								}
								_, reader, err := c.Exec(ctx, cmd, exec.Multiplexed())
								if err != nil {
									return err
								}

								buf := new(bytes.Buffer)
								_, err = buf.ReadFrom(reader)
								if err != nil {
									return err
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
									return err
								}

								// 5. finally, set the function URL from the response
								functionURL = v.FunctionURLConfigs[0].FunctionURL

								return nil
							},
						},
					},
				},
			},
		}),
	)
	if err != nil {
		return nil, err
	}

	// replace the port with the one exposed by the container
	mappedPort, err := c.MappedPort(ctx, "4566/tcp")
	if err != nil {
		return nil, err
	}

	functionURL = strings.ReplaceAll(functionURL, "4566", mappedPort.Port())

	// The latest version of localstack does not add ".localstack.cloud" by default,
	// that's why we need to add it to the URL.
	functionURL = strings.ReplaceAll(functionURL, ".localhost", ".localhost.localstack.cloud")

	Connections.Lambda = functionURL

	return c, nil
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

func startStreamingQueue() (testcontainers.Container, error) {
	ctx := context.Background()

	c, err := redpanda.Run(
		ctx,
		"docker.redpanda.com/redpandadata/redpanda:v24.3.7",
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

Now run `go mod tidy` from the root of the project to download the Go dependencies, this time only the Testcontainers for Go's LocalStack module.

Also run `go mod tidy` from the `lambda-go` directory to download the Go dependencies for the lambda function.

Finally, stop the application with <kbd>Ctrl</kbd>+<kbd>C</kbd> and run the application again with `make dev`. This time, the application will build the lambda, will start all the services, and the application will be able to connect to it.

```text
go mod tidy
GOOS=linux go build -tags lambda.norpc -o bootstrap main.go
zip -j function.zip bootstrap
  adding: bootstrap (deflated 45%)
go run -tags dev -v ./...

 ┌───────────────────────────────────────────────────┐ 
 │                   Fiber v2.52.6                   │ 
 │               http://127.0.0.1:8080               │ 
 │       (bound on host 0.0.0.0 and port 8080)       │ 
 │                                                   │ 
 │ Handlers ............. 6  Processes ........... 1 │ 
 │ Prefork ....... Disabled  PID ............. 26322 │ 
 └───────────────────────────────────────────────────┘ 

```

In the second terminal, check the containers, we will see the LocalStack instance is running alongside the Postgres database, the Redis store and the Redpanda streaming queue:

```text
$ docker ps
CONTAINER ID   IMAGE                                               COMMAND                  CREATED         STATUS                   PORTS                                                                                                                                             NAMES
c514896580c1   localstack/localstack:latest                         "docker-entrypoint.sh"   2 minutes ago   Up 2 minutes (healthy)   4510-4559/tcp, 5678/tcp, 0.0.0.0:32792->4566/tcp, :::32792->4566/tcp                                                                              priceless_antonelli
07fb1e908b1e   docker.redpanda.com/redpandadata/redpanda:v24.3.7   "/entrypoint-tc.sh r…"   3 minutes ago   Up 3 minutes             8082/tcp, 0.0.0.0:32791->8081/tcp, :::32791->8081/tcp, 0.0.0.0:32790->9092/tcp, :::32790->9092/tcp, 0.0.0.0:32789->9644/tcp, :::32789->9644/tcp   loving_murdock
bf4fcb4cd74c   redis:6-alpine                                      "docker-entrypoint.s…"   3 minutes ago   Up 3 minutes             0.0.0.0:32788->6379/tcp, :::32788->6379/tcp                                                                                                       angry_shirley
d5ec7cecb562   postgres:15.3-alpine                                "docker-entrypoint.s…"   3 minutes ago   Up 3 minutes             0.0.0.0:32787->5432/tcp, :::32787->5432/tcp                                                                                                       laughing_kare
```

The LocalStack instance is now running, and a lambda function is deployed in it. We can verify the lambda function is running by sending a request to the function URL. But we first need to obtain the URL of the lambda. Please do a GET request to the `/` endpoint of the API, where we'll get the metadata of the application. Something similar to this:

```bash
$ curl -X GET http://localhost:8080/
```

The JSON response:

```json
{"metadata":{"ratings_lambda":"http://5xnih5q8vrjmzsmis1ic4740eszvzbao.lambda-url.us-east-1.localhost.localstack.cloud:32829/","ratings":"redis://localhost:32825","streams":"localhost:32827","talks":"postgres://postgres:postgres@localhost:32824/talks-db?"}}
```

In your terminal, copy the `ratings_lambda` URL from the response and send a POST request to it with `curl` (please remember to replace the URL with the one we got from the response):

```bash
curl -X POST http://5xnih5q8vrjmzsmis1ic4740eszvzbao.lambda-url.us-east-1.localhost.localstack.cloud:32829/ -d '{"ratings":{"2":1,"4":3,"5":1}}' -H "Content-Type: application/json"
```

The JSON response:

```json
{"avg": 3.8, "totalCount": 5}%
```

Great! the response contains the average rating of the talk, and the total number of ratings, calculated in the lambda function.

### 
[Next: Adding integration tests](step-8-adding-integration-tests.md)