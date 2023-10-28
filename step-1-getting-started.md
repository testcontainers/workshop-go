# Step 1: Getting Started

## Check Go
We'll need Go 1.20 or newer for this workshop. 

This workshop uses a [Gin](https://gin-gonic.com) application, which requires Go 1.13 or newer, but Testcontainers for Go is compatible with Go 1.20+.

## Check Docker

Make sure we have a Docker environment available on your machine. 

* It can be [Testcontainers Cloud](https://testcontainers.com/cloud) recommended to avoid straining the conference network by pulling heavy Docker images. 

* It can be local Docker, which we can check by running: 

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
docker pull docker.redpanda.com/redpandadata/redpanda:v23.1.7
docker pull localstack/localstack:2.3.0
```

### 
[Next](step-2-exploring-the-app.md)
