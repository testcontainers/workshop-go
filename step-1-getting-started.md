# Step 1: Getting Started

## Check Make

We'll need Make to run the workshop. 

```shell
$ make --version
GNU Make 3.81
```

## Check Go

We'll need Go 1.20 or newer for this workshop.

For installing Go, please follow the instructions at [https://golang.org/doc/install](https://golang.org/doc/install), or use your favorite package manager, like [`gvm`](https://github.com/andrewkroh/gvm).

This workshop uses a [Gin](https://gin-gonic.com) application, which requires Go 1.13 or newer, but Testcontainers for Go is compatible with Go 1.20+.

## Check Docker

Make sure we have a Docker environment available on your machine. 

The recommended Docker environment is [Testcontainers Desktop](https://testcontainers.com/desktop), the free companion app that is the perfect for running Testcontainers on your machine. Please download and install it, and create a free account if you don't have one yet.

With Testcontainers Desktop, we can simply choose the container runtimes we want to use, and Testcontainers Desktop will take care of the rest. At the same time, we can choose running the container in an embedded runtime, which is a lightweight and performant Docker runtime that is bundled with Testcontainers Desktop (_only available for Mac at the moment_), or using [Testcontainers Cloud](https://testcontainers.com/cloud) as a remote runtime (recommended to avoid straining conference networks by pulling heavy Docker images).

If you already have a local Docker runtime (on Linux, For Mac, or For Windows), this workshop works perfectly fine with that as well.

We can check our container runtime by simply running: 

```shell
$ docker version
Client:
 Cloud integration: v1.0.35
 Version:           24.0.2
 API version:       1.42 (downgraded from 1.43)
 Go version:        go1.20.4
 Git commit:        cb74dfc
 Built:             Thu May 25 21:51:16 2023
 OS/Arch:           darwin/arm64
 Context:           tcd

Server:
 Engine:
  Version:          23.0.6
  API version:      1.42 (minimum version 1.12)
  Go version:       go1.20.10
  Git commit:       9dbdbd4b6d7681bd18c897a6ba0376073c2a72ff
  Built:            Thu Oct 12 14:14:03 2023
  OS/Arch:          linux/arm64
  Experimental:     false
 containerd:
  Version:          v1.7.2
  GitCommit:        0cae528dd6cb557f7201036e9f43420650207b58
 runc:
  Version:          1.1.7
  GitCommit:        860f061b76bb4fc671f0f9e900f7d80ff93d4eb7
 docker-init:
  Version:          0.19.0
  GitCommit: 
```

## Download the project

Clone the following project from GitHub to your computer:  
[https://github.com/testcontainers/workshop-go](https://github.com/testcontainers/workshop-go)

## Download the dependencies

```shell
go get github.com/google/uuid
go get github.com/jackc/pgx/v5
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/localstack
go get github.com/testcontainers/testcontainers-go/modules/postgres
go get github.com/testcontainers/testcontainers-go/modules/redis
go get github.com/testcontainers/testcontainers-go/modules/redpanda
go get github.com/stretchr/testify
```

## \(optionally\) Pull the required images before doing the workshop

This might be helpful if the internet connection at the workshop venue is somewhat slow.

```text
docker pull postgres:15.3-alpine
docker pull redis:6-alpine
docker pull docker.redpanda.com/redpandadata/redpanda:v24.3.7
docker pull localstack/localstack:2.3.0
```

### 
[Next: Exploring the app](step-2-exploring-the-app.md)
