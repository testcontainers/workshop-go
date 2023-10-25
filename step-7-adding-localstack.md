# Step 7: Adding Localstack

The application is using an AWS lambda function to calculate the average rating of a talk. The lambda functions are invoked by the application when the ratings for a talk are requested, using HTTP calls to the function URL of the lambda.

LocalStack is a cloud service emulator that runs in a single container on your laptop or in your CI environment. With LocalStack, you can run your AWS applications or Lambdas entirely on your local machine without connecting to a remote cloud provider!

## Creating the lambda function

The lambda function is a simple Node.js function that calculates the average rating of a talk. The function is defined in the `testdata/index.js` file:

```javascript
// it will receive a json object with a map of entries, where the key is the rating, and the value is the counts of that rating
exports.handler = async (event) => {
    let body = JSON.parse(event.body)
    
    let ratings = body.ratings;
    let avg = 0;

    let total = 0;
    let totalCount = 0;

    for (let ratingValue in ratings) {
        totalCount += parseInt(ratings[ratingValue]);
        total += parseInt(ratingValue) * ratings[ratingValue];
    }

    avg = total / totalCount;

    const response = {
        statusCode: 200,
        body: JSON.stringify({
            'avg': avg,
            'totalCount': totalCount,
        }),
    };

    return response;
};
```

Now `zip` the Javascript file into the `function.zip` file:

```bash
zip -r function.zip index.js
```

This zip file will be used by the lambda function to deploy the function in the Localstack instance.

## Adding the Localstack instance

Let's add a Localstack instance using Testcontainers for Go.

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
						HostFilePath:      filepath.Join("testdata", "function.zip"),
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
			"--runtime", "nodejs18.x",
			"--zip-file",
			"fileb:///tmp/function.zip",
			"--handler", "index.handler",
			"--role", "arn:aws:iam::000000000000:role/lambda-role",
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

This function will start a Localstack instance, deploy the lambda function in it, and adds the URL of the lambda function to the `Connections` struct.

3. Update the comments for the init function `startupDependenciesFn` slice to include the Localstack store:

```go
// init will be used to start up the containers for development mode. It will use
// testcontainers-go to start up the following containers:
// - Postgres: store for talks
// - Redis: store for ratings
// - Redpanda: message queue for the ratings
// - Localstack: cloud emulator for AWS Lambdas
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
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/testcontainers/testcontainers-go"
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
// - Localstack: cloud emulator for AWS Lambdas
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
						HostFilePath:      filepath.Join("testdata", "function.zip"),
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
			"--runtime", "nodejs18.x",
			"--zip-file",
			"fileb:///tmp/function.zip",
			"--handler", "index.handler",
			"--role", "arn:aws:iam::000000000000:role/lambda-role",
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
2023/10/25 17:18:19 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 78+testcontainerscloud (via Testcontainers Desktop 1.4.18)
  API Version: 1.43
  Operating System: Ubuntu 22.04.3 LTS
  Total Memory: 15689 MB
  Resolved Docker Host: tcp://127.0.0.1:56978
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 1435e98a6f031d06b4f6faf981e91c33d30284b1da6a6b5eb5e5979b014549c9
  Test ProcessID: 570696e0-c933-4478-a9b0-468286e0dbe1
2023/10/25 17:18:19 🐳 Creating container for image postgres:15.3-alpine
2023/10/25 17:18:19 ✅ Container created: f032cfd9263a
2023/10/25 17:18:19 🐳 Starting container: f032cfd9263a
2023/10/25 17:18:19 ✅ Container started: f032cfd9263a
2023/10/25 17:18:19 🚧 Waiting for container id f032cfd9263a image: postgres:15.3-alpine. Waiting for: &{timeout:<nil> deadline:0x140004214f8 Strategies:[0x14000431200]}
2023/10/25 17:18:21 🐳 Creating container for image redis:6-alpine
2023/10/25 17:18:21 ✅ Container created: acbd71c8e808
2023/10/25 17:18:21 🐳 Starting container: acbd71c8e808
2023/10/25 17:18:21 ✅ Container started: acbd71c8e808
2023/10/25 17:18:21 🚧 Waiting for container id acbd71c8e808 image: redis:6-alpine. Waiting for: &{timeout:<nil> Log:* Ready to accept connections IsRegexp:false Occurrence:1 PollInterval:100ms}
2023/10/25 17:18:21 🐳 Creating container for image docker.redpanda.com/redpandadata/redpanda:v23.1.7
2023/10/25 17:18:21 ✅ Container created: a49ae549af9c
2023/10/25 17:18:21 🐳 Starting container: a49ae549af9c
2023/10/25 17:18:22 ✅ Container started: a49ae549af9c
2023/10/25 17:18:22 Setting LOCALSTACK_HOST to 127.0.0.1 (to match host-routable address for container)
2023/10/25 17:18:22 🐳 Creating container for image localstack/localstack:2.3.0
2023/10/25 17:18:22 ✅ Container created: 1d0a4c2f4503
2023/10/25 17:18:22 🐳 Starting container: 1d0a4c2f4503
2023/10/25 17:18:23 ✅ Container started: 1d0a4c2f4503
2023/10/25 17:18:23 🚧 Waiting for container id 1d0a4c2f4503 image: localstack/localstack:2.3.0. Waiting for: &{timeout:0x14000636448 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x104e95080 ResponseMatcher:0x104f66690 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> PollInterval:100ms UserInfo:}
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

In the second terminal, check the containers, you will see the Localstack instance is running alongside the Postgres database, the Redis store and the Redpanda streaming queue:

```text
$ docker ps
CONTAINER ID   IMAGE                                               COMMAND                  CREATED              STATUS                        PORTS                                                                                                                                             NAMES
1d0a4c2f4503   localstack/localstack:2.3.0                         "docker-entrypoint.sh"   About a minute ago   Up About a minute (healthy)   4510-4559/tcp, 5678/tcp, 0.0.0.0:32788->4566/tcp, :::32788->4566/tcp                                                                              distracted_robinson
a49ae549af9c   docker.redpanda.com/redpandadata/redpanda:v23.1.7   "/entrypoint-tc.sh r…"   About a minute ago   Up About a minute             8082/tcp, 0.0.0.0:32787->8081/tcp, :::32787->8081/tcp, 0.0.0.0:32786->9092/tcp, :::32786->9092/tcp, 0.0.0.0:32785->9644/tcp, :::32785->9644/tcp   strange_mendeleev
acbd71c8e808   redis:6-alpine                                      "docker-entrypoint.s…"   About a minute ago   Up About a minute             0.0.0.0:32784->6379/tcp, :::32784->6379/tcp                                                                                                       busy_swirles
f032cfd9263a   postgres:15.3-alpine                                "docker-entrypoint.s…"   About a minute ago   Up About a minute             0.0.0.0:32783->5432/tcp, :::32783->5432/tcp                                                                                                       modest_einstein
```

The Localstack instance is now running, and a lambda function is deployed in it. We can verify the lambda function is running by sending a request to the function URL. But we first need to obtain the URL of the lambda. Please do a GET request to the `/` endpoint of the API, where you'll get the metadata of the application. Something similar to this:

```bash
$ curl -X GET http://localhost:8080/
{"metadata":{"ratings_lambda":"http://46aofv7yecnd1ncaaue4qkhtnwok8hyv.lambda-url.us-east-1.localhost.localstack.cloud:32811/","ratings":"redis://127.0.0.1:32784","streams":"127.0.0.1:32786","talks":"postgres://postgres:postgres@127.0.0.1:32783/talks-db?"}}
```

In your terminal, copy the `ratings_lambda` URL from the response and send a POST request to it with `curl` (please remember to replace the URL with the one you got from the response):

```bash
curl -X POST http://46aofv7yecnd1ncaaue4qkhtnwok8hyv.lambda-url.us-east-1.localhost.localstack.cloud:32811/ -d '{"ratings":{"2":"1","4":"3","5":"1"}}' -H "Content-Type: application/json"
{"avg": 3.8, "totalCount": 5}%
```

The response contains the average rating of the talk, and the total number of ratings, calculated in the lambda function.

### 
[Next](step-8-adding-integration-tests.md)