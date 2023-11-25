package ratings_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"github.com/testcontainers/workshop-go/internal/ratings"
)

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

func TestGetStats(t *testing.T) {
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
		panic(err)
	}

	url := strings.ReplaceAll(functionURL, "4566", mappedPort.Port())

	// now we can test the lambda function
	lambdaClient := ratings.NewLambdaClient(url)

	histogram := map[string]string{
		"0": "10",
		"1": "20",
		"2": "30",
		"3": "40",
		"4": "50",
		"5": "60",
	}

	stats, err := lambdaClient.GetStats(histogram)
	if err != nil {
		t.Fatalf("failed to get stats: %s", err)
	}

	expected := `{"avg":3.3333333333333335,"totalCount":210}`
	if string(stats) != expected {
		t.Fatalf("expected %s, got %s", expected, string(stats))
	}
}
