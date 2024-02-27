# Integration tests for the Go Lambda

Up to this point, we have worked in the ratings application, which consumes a Go lambda. In this step, we will improve the experience on working on the Go lambda as a separate project. We will add integration tests for the lambda, and we will use Testcontainers to run the lambda on LocalStack.

## Adding integration tests for the lambda

We have a "working" lambda, but we don't have any tests for it. Let's add an integration test for it. It will deploy the lambda into LocalStack and invoke it.

Let's create a `main_test.go` file in the `lambda-go` folder:

```go
package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

// buildLambda return the path to the ZIP file used to deploy the lambda function.
func buildLambda() string {
	makeCmd := osexec.Command("make", "zip-lambda")
	makeCmd.Dir = "."

	err := makeCmd.Run()
	if err != nil {
		panic(fmt.Errorf("failed to zip lambda: %w", err))
	}

	return filepath.Join("function.zip")
}

func TestDeployLambda(t *testing.T) {
	ctx := context.Background()

	flagsFn := func() string {
		labels := testcontainers.GenericLabels()
		flags := ""
		for k, v := range labels {
			flags = fmt.Sprintf("%s -l %s=%s", flags, k, v)
		}
		return flags
	}

	// get the path to the function.zip file, which lives in the lambda-go folder of the project
	zipFile := buildLambda()

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
						HostFilePath:      zipFile,
						ContainerFilePath: "/tmp/function.zip",
					},
				},
			},
		}),
	)
	if err != nil {
		t.Fatalf("failed to start localstack container: %s", err)
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
			t.Fatalf("failed to execute command %s: %s", cmd, err)
		}
	}

	// 4. get the URL for the lambda function
	cmd := []string{
		"awslocal", "lambda", "list-function-url-configs", "--function-name", lambdaName,
	}
	_, reader, err := c.Exec(ctx, cmd, exec.Multiplexed())
	if err != nil {
		t.Fatalf("failed to execute command %s: %s", cmd, err)
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(reader)
	if err != nil {
		t.Fatalf("failed to read from reader: %s", err)
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
		t.Fatalf("failed to unmarshal content: %s", err)
	}

	functionURL := v.FunctionURLConfigs[0].FunctionURL

	// replace the port with the one exposed by the container
	mappedPort, err := c.MappedPort(ctx, "4566/tcp")
	if err != nil {
		t.Fatalf("failed to get mapped port: %s", err)
	}

	url := strings.ReplaceAll(functionURL, "4566", mappedPort.Port())

	// now we can test the lambda function

	histogram := map[string]string{
		"0": "10",
		"1": "20",
		"2": "30",
		"3": "40",
		"4": "50",
		"5": "60",
	}

	payload := `{"ratings": {`
	for rating, count := range histogram {
		// we are passing the count as an integer, so we don't need to quote it
		payload += `"` + rating + `": ` + count + `,`
	}

	if len(histogram) > 0 {
		// remove the last comma onl for non-empty histograms
		payload = payload[:len(payload)-1]
	}
	payload += "}}"

	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := httpClient.Post(url, "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("failed to send request: %s", err)
	}

	stats, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %s", err)
	}

	expected := `{"avg":3.3333333333333335,"totalCount":210}`
	if string(stats) != expected {
		t.Fatalf("expected %s, got %s", expected, string(stats))
	}
}

```

The test above is doing the following:

1. It starts LocalStack with the `lambda` service enabled.
2. It creates a lambda function and a URL configuration for the lambda function.
3. It gets the URL for the lambda function.
4. It invokes the lambda function with a payload containing a histogram of ratings.

The test is using the `testcontainers-go` library to start LocalStack and to execute commands inside the container. It is also using the `awslocal` command to interact with the LocalStack container.

Let's replace the contents of the `Makefile` for the lambda-go project. We are adding a new target for running the integration tests:

```makefile
mod-tidy:
	go mod tidy

build-lambda: mod-tidy
	# If you are using Testcontainers Cloud, please add 'GOARCH=amd64' in order to get the localstack's lambdas using the right architecture
	GOOS=linux go build -tags lambda.norpc -o bootstrap main.go

test:
	go test -v -count=1 ./...

zip-lambda: build-lambda
	zip -j function.zip bootstrap
```

Now run the integration tests with your IDE or from a terminal, in the lambda directory, but first update the Go dependencies with the `make mod-tidy` command:

```shell
$ cd lambda-go
$ make mod-tidy test
go test -v -count=1 ./...
# github.com/testcontainers/workshop-go/lambda-go.test
=== RUN   TestDeployLambda
2023/10/30 12:54:35 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 23.0.6 (via Testcontainers Desktop 1.5.0)
  API Version: 1.42
  Operating System: Alpine Linux v3.18
  Total Memory: 5256 MB
  Resolved Docker Host: tcp://127.0.0.1:54034
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 322d823de6e48bc23ed27eb788ce9e7af040a98061329aa861bcf9b0d075bbc8
  Test ProcessID: 6bf2f3be-b3f1-40fb-91a6-9038a23b5d9c
2023/10/30 12:54:35 Setting LOCALSTACK_HOST to 127.0.0.1 (to match host-routable address for container)
2023/10/30 12:54:35 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/30 12:54:35 ✅ Container created: 8b6d4a9cc768
2023/10/30 12:54:35 🐳 Starting container: 8b6d4a9cc768
2023/10/30 12:54:35 ✅ Container started: 8b6d4a9cc768
2023/10/30 12:54:35 🚧 Waiting for container id 8b6d4a9cc768 image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/30 12:54:36 🐳 Creating container for image localstack/localstack:2.3.0
2023/10/30 12:54:36 ✅ Container created: f0055b217203
2023/10/30 12:54:36 🐳 Starting container: f0055b217203
2023/10/30 12:54:36 ✅ Container started: f0055b217203
2023/10/30 12:54:36 🚧 Waiting for container id f0055b217203 image: localstack/localstack:2.3.0. Waiting for: &{timeout:0x140003d3238 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x1030748e0 ResponseMatcher:0x1031453a0 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> PollInterval:100ms UserInfo:}
--- PASS: TestDeployLambda (17.87s)
PASS
ok      github.com/testcontainers/workshop-go/lambda-go 18.207s
```

You'll probably see the `go.mod` and `go.sum` file to change, adding the `testcontainers-go` library and its Go dependencies.

## Making the tests to fail

Let's introduce a bug in the lambda function and see how the tests will fail. In the `main.go` file, let's change how the average of the ratings is calculated:


```diff
	var avg float64
	if totalCount > 0 {
-		avg = float64(sum) / float64(totalCount)
+		avg = float64(sum) * float64(totalCount)
	}
```

Now run the tests, with your IDE or from a terminal:

```shell
$ make test
go test -v -count=1 ./...
# github.com/testcontainers/workshop-go/lambda-go.test
=== RUN   TestDeployLambda
2023/10/30 12:59:40 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 23.0.6 (via Testcontainers Desktop 1.5.0)
  API Version: 1.42
  Operating System: Alpine Linux v3.18
  Total Memory: 5256 MB
  Resolved Docker Host: tcp://127.0.0.1:54034
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 16f6d59f4ea8d50a6114bb57f3d495bc869d5afa9cb78711b7e6ef22d692a88b
  Test ProcessID: ecde6c0f-2957-4f11-ae5a-85d61d5ff882
2023/10/30 12:59:40 Setting LOCALSTACK_HOST to 127.0.0.1 (to match host-routable address for container)
2023/10/30 12:59:40 🐳 Creating container for image docker.io/testcontainers/ryuk:0.5.1
2023/10/30 12:59:40 ✅ Container created: ebd4bbd7d64d
2023/10/30 12:59:40 🐳 Starting container: ebd4bbd7d64d
2023/10/30 12:59:41 ✅ Container started: ebd4bbd7d64d
2023/10/30 12:59:41 🚧 Waiting for container id ebd4bbd7d64d image: docker.io/testcontainers/ryuk:0.5.1. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms}
2023/10/30 12:59:41 🐳 Creating container for image localstack/localstack:2.3.0
2023/10/30 12:59:41 ✅ Container created: 40cb2869f67e
2023/10/30 12:59:41 🐳 Starting container: 40cb2869f67e
2023/10/30 12:59:42 ✅ Container started: 40cb2869f67e
2023/10/30 12:59:42 🚧 Waiting for container id 40cb2869f67e image: localstack/localstack:2.3.0. Waiting for: &{timeout:0x14000485148 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x10523c8e0 ResponseMatcher:0x10530d3a0 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> PollInterval:100ms UserInfo:}
    main_test.go:177: expected {"avg":3.3333333333333335,"totalCount":210}, got {"avg":147000,"totalCount":210}
--- FAIL: TestDeployLambda (17.60s)
FAIL
FAIL    github.com/testcontainers/workshop-go/lambda-go 17.880s
FAIL
make: *** [test] Error 1
```

As expected, the test failed because the lambda function is returning an incorrect average:

```text
    main_test.go:177: expected {"avg":3.3333333333333335,"totalCount":210}, got {"avg":147000,"totalCount":210}
```

Rollback the change in the `main.go` file, and run the tests again, they will pass again.

### 
[Next: exploring the running app](step-12-exploring-the-running-app.md)