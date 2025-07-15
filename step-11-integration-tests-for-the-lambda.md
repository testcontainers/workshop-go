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
		testcontainers.WithAdditionalLifecycleHooks(testcontainers.ContainerLifecycleHooks{
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
		}),
	)
	testcontainers.CleanupContainer(t, c)
	require.NoError(t, err)

	// replace the port with the one exposed by the container
	mappedPort, err := c.MappedPort(ctx, "4566/tcp")
	require.NoError(t, err)

	url := strings.ReplaceAll(functionURL, "4566", mappedPort.Port())

	// The latest version of localstack does not add ".localstack.cloud" by default,
	// that's why we need to add it to the URL.
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
2025/05/07 13:27:48 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 27.5.0
  API Version: 1.47
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15368 MB
  Labels:
    cloud.docker.run.version=259.c712f5fd
    cloud.docker.run.plugin.version=0.2.20
    com.docker.desktop.address=unix:///Users/mdelapenya/Library/Containers/com.docker.docker/Data/docker-cli.sock
  Testcontainers for Go Version: v0.37.0
  Resolved Docker Host: unix:///var/run/docker.sock
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: e2a7d32bf743b96698083a73e6d8e091f30cd208028037421e783e8a3840fd43
  Test ProcessID: 7fe35795-7665-491d-9523-f1a6118fb8a9
2025/05/07 13:27:48 Setting LOCALSTACK_HOST to localhost (to match host-routable address for container)
2025/05/07 13:27:48 🐳 Creating container for image localstack/localstack:latest
2025/05/07 13:27:48 🐳 Creating container for image testcontainers/ryuk:0.11.0
2025/05/07 13:27:49 ✅ Container created: 1af05cd523b9
2025/05/07 13:27:49 🐳 Starting container: 1af05cd523b9
2025/05/07 13:27:49 ✅ Container started: 1af05cd523b9
2025/05/07 13:27:49 ⏳ Waiting for container id 1af05cd523b9 image: testcontainers/ryuk:0.11.0. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms skipInternalCheck:false}
2025/05/07 13:27:50 🔔 Container is ready: 1af05cd523b9
2025/05/07 13:27:50 ✅ Container created: 4df47531ebad
2025/05/07 13:27:51 🐳 Starting container: 4df47531ebad
2025/05/07 13:27:59 ✅ Container started: 4df47531ebad
2025/05/07 13:27:59 ⏳ Waiting for container id 4df47531ebad image: localstack/localstack:latest. Waiting for: &{timeout:0x140001265b0 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x1011f3930 ResponseMatcher:0x10125b620 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> Headers:map[] ResponseHeadersMatcher:0x10125b630 PollInterval:100ms UserInfo: ForceIPv4LocalHost:false}
2025/05/07 13:28:00 🔔 Container is ready: 4df47531ebad
2025/05/07 13:28:01 🐳 Stopping container: 4df47531ebad
2025/05/07 13:28:04 ✅ Container stopped: 4df47531ebad
2025/05/07 13:28:04 🐳 Terminating container: 4df47531ebad
2025/05/07 13:28:04 🚫 Container terminated: 4df47531ebad
--- PASS: TestDeployLambda (17.93s)
PASS
ok      github.com/testcontainers/workshop-go/lambda-go 18.756s
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
2025/05/07 13:30:26 github.com/testcontainers/testcontainers-go - Connected to docker: 
  Server Version: 27.5.0
  API Version: 1.47
  Operating System: Ubuntu 22.04.5 LTS
  Total Memory: 15368 MB
  Labels:
    cloud.docker.run.version=259.c712f5fd
    cloud.docker.run.plugin.version=0.2.20
    com.docker.desktop.address=unix:///Users/mdelapenya/Library/Containers/com.docker.docker/Data/docker-cli.sock
  Testcontainers for Go Version: v0.37.0
  Resolved Docker Host: unix:///var/run/docker.sock
  Resolved Docker Socket Path: /var/run/docker.sock
  Test SessionID: 7ed8e0d5c1fb58d25e1c52f105288d105b645d89f000355c46de9acd9d622ec5
  Test ProcessID: e35bd5f4-db46-4a61-afce-4279c957cb82
2025/05/07 13:30:26 Setting LOCALSTACK_HOST to localhost (to match host-routable address for container)
2025/05/07 13:30:26 🐳 Creating container for image localstack/localstack:latest
2025/05/07 13:30:27 🐳 Creating container for image testcontainers/ryuk:0.11.0
2025/05/07 13:30:27 ✅ Container created: 0a30b25b9bf9
2025/05/07 13:30:27 🐳 Starting container: 0a30b25b9bf9
2025/05/07 13:30:27 ✅ Container started: 0a30b25b9bf9
2025/05/07 13:30:27 ⏳ Waiting for container id 0a30b25b9bf9 image: testcontainers/ryuk:0.11.0. Waiting for: &{Port:8080/tcp timeout:<nil> PollInterval:100ms skipInternalCheck:false}
2025/05/07 13:30:28 🔔 Container is ready: 0a30b25b9bf9
2025/05/07 13:30:28 ✅ Container created: 30af32569a44
2025/05/07 13:30:29 🐳 Starting container: 30af32569a44
2025/05/07 13:30:38 ✅ Container started: 30af32569a44
2025/05/07 13:30:38 ⏳ Waiting for container id 30af32569a44 image: localstack/localstack:latest. Waiting for: &{timeout:0x1400048ea90 Port:4566/tcp Path:/_localstack/health StatusCodeMatcher:0x105437930 ResponseMatcher:0x10549f620 UseTLS:false AllowInsecure:false TLSConfig:<nil> Method:GET Body:<nil> Headers:map[] ResponseHeadersMatcher:0x10549f630 PollInterval:100ms UserInfo: ForceIPv4LocalHost:false}
2025/05/07 13:30:39 🔔 Container is ready: 30af32569a44
    main_test.go:183: 
                Error Trace:    /Users/mdelapenya/sourcecode/src/github.com/testcontainers/workshop-go/lambda-go/main_test.go:183
                Error:          Not equal: 
                                expected: "{\"avg\":3.3333333333333335,\"totalCount\":210}"
                                actual  : "{\"avg\":147000,\"totalCount\":210}"
                            
                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1 +1 @@
                                -{"avg":3.3333333333333335,"totalCount":210}
                                +{"avg":147000,"totalCount":210}
                Test:           TestDeployLambda
2025/05/07 13:30:39 🐳 Stopping container: 30af32569a44
2025/05/07 13:30:44 ✅ Container stopped: 30af32569a44
2025/05/07 13:30:44 🐳 Terminating container: 30af32569a44
2025/05/07 13:30:44 🚫 Container terminated: 30af32569a44
--- FAIL: TestDeployLambda (20.30s)
FAIL
FAIL    github.com/testcontainers/workshop-go/lambda-go 21.239s
FAIL
make: *** [test] Error 1
```

As expected, the test failed because the lambda function is returning an incorrect average:

```text
    main_test.go:183: 
                Error Trace:    /Users/mdelapenya/sourcecode/src/github.com/testcontainers/workshop-go/lambda-go/main_test.go:183
                Error:          Not equal: 
                                expected: "{\"avg\":3.3333333333333335,\"totalCount\":210}"
                                actual  : "{\"avg\":147000,\"totalCount\":210}"
```

Rollback the change in the `main.go` file, and run the tests again, they will pass again.

### 
[Next: exploring the running app](step-12-exploring-the-running-app.md)