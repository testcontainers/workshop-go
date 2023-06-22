# Step 1: Getting Started

## Check Go
You'll need Go 1.19 or newer for this workshop. 

This workshop uses a [Go Fiber](https://gofiber.io/) application, which requires Go 1.17 or newer, but Testcontainers for Go is compatible with Go 1.19+.

## Check Docker

Make sure you have a Docker environment available on your machine. 

* It can be [Testcontainers Cloud](https://testcontainers.com/cloud) recommended to avoid straining the conference network by pulling heavy Docker images. 

* It can be local Docker, which you can check by running: 

```shell
$ docker version
Client:
 Cloud integration: v1.0.31
 Version:           20.10.23
 API version:       1.41
 Go version:        go1.18.10
 Git commit:        7155243
 Built:             Thu Jan 19 17:35:19 2023
 OS/Arch:           darwin/arm64
 Context:           default
 Experimental:      true

Server: Docker Desktop 4.17.0 (99724)
 Engine:
  Version:          20.10.23
  API version:      1.41 (minimum version 1.12)
  Go version:       go1.18.10
  Git commit:       6051f14
  Built:            Thu Jan 19 17:31:28 2023
  OS/Arch:          linux/arm64
  Experimental:     false
 containerd:
  Version:          1.6.18
  GitCommit:        2456e983eb9e37e47538f59ea18f2043c9a73640
 runc:
  Version:          1.1.4
  GitCommit:        v1.1.4-0-g5fd4c4d
 docker-init:
  Version:          0.19.0
  GitCommit:        de40ad0
```

## Download the project

Clone the following project from GitHub to your computer:  
[https://github.com/testcontainers/workshop-go](https://github.com/testcontainers/workshop-go)

## Download the dependencies

```shell
go get github.com/jackc/pgx/v5
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
go get github.com/testcontainers/testcontainers-go/modules/redis
go get github.com/testcontainers/testcontainers-go/modules/redpanda
go get github.com/stretchr/testify
```

## \(optionally\) Pull the required images before doing the workshop

This might be helpful if the internet connection at the workshop venue is somewhat slow.

```text
docker pull postgres:14-alpine
docker pull redis:6-alpine
docker pull openjdk:8-jre-alpine
docker pull confluentinc/cp-kafka:6.2.1
```

### 
[Next](step-2-exploring-the-app.md)
