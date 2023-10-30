mod-tidy:
	go mod tidy

build-lambda: mod-tidy
	GOOS=linux go build -tags lambda.norpc -o bootstrap main.go

zip-lambda: build-lambda
	zip -j function.zip bootstrap
