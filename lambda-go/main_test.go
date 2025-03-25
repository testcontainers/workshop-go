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
		"localstack/localstack:2.3.0",
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
	if err != nil {
		t.Fatalf("failed to start localstack container: %s", err)
	}

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
		Timeout: 15 * time.Second,
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
