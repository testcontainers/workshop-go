mod-tidy:
	go mod tidy

build-lambda: mod-tidy
	GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o bootstrap main.go

test:
	go test -v -count=1 ./...

zip-lambda: build-lambda
	zip -j function.zip bootstrap