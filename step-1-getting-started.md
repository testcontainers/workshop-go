# Step 1: Getting Started

## Check Make

We'll need Make to run the workshop. 

```shell
$ make --version
GNU Make 3.81
```

## Check Go

We'll need Go 1.24 or newer for this workshop.

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
 Version:           27.2.1-rd
 API version:       1.43 (downgraded from 1.47)
 Go version:        go1.22.7
 Git commit:        cc0ee3e
 Built:             Tue Sep 10 15:41:09 2024
 OS/Arch:           darwin/arm64
 Context:           tcd

Server: Docker Engine - Community
 Engine:
  Version:          27.5.0
  API version:      1.47 (minimum version 1.24)
  Go version:       go1.22.10
  Git commit:       38b84dc
  Built:            Thu Jan 16 09:42:44 2025
  OS/Arch:          linux/amd64
  Experimental:     false
 containerd:
  Version:          1.7.24
  GitCommit:        
 runc:
  Version:          1.1.12-0ubuntu2~22.04.1
  GitCommit:        
 docker-init:
  Version:          0.19.0
  GitCommit:        de40ad0
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
docker pull localstack/localstack:latest
```

### 
[Next: Exploring the app](step-2-exploring-the-app.md)
