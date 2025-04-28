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

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
)

// buildLambda return the path to the ZIP file used to deploy the lambda function.
func buildLambda(t *testing.T) string {
	t.Helper()

	makeCmd := osexec.Command("make", "zip-lambda")
	makeCmd.Dir = "."

	err := makeCmd.Run()
	require.NoError(t, err)

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
	zipFile := buildLambda(t)

	var functionURL string

	c, err := localstack.Run(ctx,
		"localstack/localstack:latest",
		testcontainers.WithEnv(map[string]string{
			"SERVICES":            "lambda",
			"LAMBDA_DOCKER_FLAGS": flagsFn(),
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      zipFile,
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

								functionURL = v.FunctionURLConfigs[0].FunctionURL

								return nil
							},
						},
					},
				},
			},
		}),
	)
	testcontainers.CleanupContainer(t, c)
	require.NoError(t, err)

	// replace the port with the one exposed by the container
	mappedPort, err := c.MappedPort(ctx, "4566/tcp")
	require.NoError(t, err)

	url := strings.ReplaceAll(functionURL, "4566", mappedPort.Port())

	// The latest version of localstack does not add ".localstack.cloud" by default,
	// that's why need to add it to the URL.
	url = strings.ReplaceAll(url, ".localhost", ".localhost.localstack.cloud")

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
		Timeout: 15 * time.Second,
	}

	resp, err := httpClient.Post(url, "application/json", bytes.NewBufferString(payload))
	require.NoError(t, err)

	stats, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	expected := `{"avg":3.3333333333333335,"totalCount":210}`
	require.Equal(t, expected, string(stats))
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

test: mod-tidy
	go test -v -count=1 ./...

zip-lambda: build-lambda
	zip -j function.zip bootstrap
```

Now run the integration tests with your IDE or from a terminal, in the lambda directory, but first update the Go dependencies with the `make mod-tidy` command:

```shell
$ cd lambda-go
$ make test
go mod tidy
go test -v -count=1 ./...
=== RUN   TestDeployLambda
2025/03/25 14:02:12 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 27.5.0 (via Testcontainers Desktop 1.19.0)
  API Version: 1.46
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15368 MB
  Labels:
    cloud.docker.run.version=259.c712f5fd
  Testcontainers for Go Version: v0.35.0
  Resolved Docker Host: tcp://127.0.0.1:49982
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 32a04170fb07432f051c21db7aea06591d4537edfe5d3a798003ec1a9516539e
  Test ProcessID: d0b47b4c-d5fe-4836-b0d5-e1356491ba24
2025/03/25 14:02:12 Setting LOCALSTACK_HOST to 127.0.0.1 (to match host-routable address for container)
2025/03/25 14:02:12 🐳 Creating container for image localstack/localstack:latest
2025/03/25 14:02:12 🐳 Creating container for image testcontainers/ryuk:0.11.0
2025/03/25 14:02:12 ✅ Container created: e351225e5172
2025/03/25 14:02:12 🐳 Starting container: e351225e5172
2025/03/25 14:02:12 ✅ Container started: e351225e5172
2025/03/25 14:02:12 ⏳ Waiting for container id e351225e5172 image: testcontainers/ryuk:0.11.0. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms skipInternalCheck:false}
2025/03/25 14:02:13 🔔 Container is ready: e351225e5172
2025/03/25 14:02:13 ✅ Container created: 5bb9dd2564d8
2025/03/25 14:02:13 🐳 Starting container: 5bb9dd2564d8
2025/03/25 14:02:23 ✅ Container started: 5bb9dd2564d8
2025/03/25 14:02:23 ⏳ Waiting for container id 5bb9dd2564d8 image: localstack/localstack:latest. Waiting for: &{timeout:0x1400037d400 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x104af72a0 ResponseMatcher:0x104b89a80 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> Headers:map[] ResponseHeadersMatcher:0x104b89a90 PollInterval:100ms UserInfo: ForceIPv4LocalHost:false}
2025/03/25 14:02:24 🔔 Container is ready: 5bb9dd2564d8
--- PASS: TestDeployLambda (14.03s)
PASS
ok      github.com/testcontainers/workshop-go/lambda-go 14.380s
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
=== RUN   TestDeployLambda
2025/03/25 14:09:13 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 27.5.0 (via Testcontainers Desktop 1.19.0)
  API Version: 1.46
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15368 MB
  Labels:
    cloud.docker.run.version=259.c712f5fd
  Testcontainers for Go Version: v0.35.0
  Resolved Docker Host: tcp://127.0.0.1:49982
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 70e53e714c6ad65ab190099ba80262d1d14325fb6171596466683f56db98c1c1
  Test ProcessID: 8a35b781-968b-42a8-b8aa-7ab4cfcd3bbf
2025/03/25 14:09:13 Setting LOCALSTACK_HOST to 127.0.0.1 (to match host-routable address for container)
2025/03/25 14:09:13 🐳 Creating container for image localstack/localstack:latest
2025/03/25 14:09:13 🐳 Creating container for image testcontainers/ryuk:0.11.0
2025/03/25 14:09:13 ✅ Container created: 8ef67818a0be
2025/03/25 14:09:13 🐳 Starting container: 8ef67818a0be
2025/03/25 14:09:14 ✅ Container started: 8ef67818a0be
2025/03/25 14:09:14 ⏳ Waiting for container id 8ef67818a0be image: testcontainers/ryuk:0.11.0. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms skipInternalCheck:false}
2025/03/25 14:09:14 🔔 Container is ready: 8ef67818a0be
2025/03/25 14:09:14 ✅ Container created: fbc64ad5cd7f
2025/03/25 14:09:15 🐳 Starting container: fbc64ad5cd7f
2025/03/25 14:09:24 ✅ Container started: fbc64ad5cd7f
2025/03/25 14:09:24 ⏳ Waiting for container id fbc64ad5cd7f image: localstack/localstack:latest. Waiting for: &{timeout:0x140004e2530 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x100bfb2a0 ResponseMatcher:0x100c8da80 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> Headers:map[] ResponseHeadersMatcher:0x100c8da90 PollInterval:100ms UserInfo: ForceIPv4LocalHost:false}
2025/03/25 14:09:25 🔔 Container is ready: fbc64ad5cd7f
    main_test.go:188: expected {"avg":3.3333333333333335,"totalCount":210}, got {"avg":147000,"totalCount":210}
--- FAIL: TestDeployLambda (15.64s)
FAIL
FAIL    github.com/testcontainers/workshop-go/lambda-go 16.414s
FAIL
make: *** [test] Error 
```

As expected, the test failed because the lambda function is returning an incorrect average:

```text
    main_test.go:177: expected {"avg":3.3333333333333335,"totalCount":210}, got {"avg":147000,"totalCount":210}
```

Rollback the change in the `main.go` file, and run the tests again, they will pass again.

### 
[Next: exploring the running app](step-12-exploring-the-running-app.md)