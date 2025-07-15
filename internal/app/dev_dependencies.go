//go:build dev || e2e
// +build dev e2e

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
