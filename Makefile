build-lambda:
	$(MAKE) -C lambda-go zip-lambda

dev: build-lambda
	TESTCONTAINERS_RYUK_DISABLED=true go run -tags dev -v ./...

test-integration:
	go test -v -count=1 ./...

test-e2e:
	go test -v -count=1 -tags e2e ./internal/app
