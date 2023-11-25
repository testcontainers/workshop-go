build-lambda:
	$(MAKE) -C lambda-go zip-lambda

dev: build-lambda
	go run -tags dev -v ./...
