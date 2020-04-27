fmt:
	gofmt -w -s .

test:
	go test -race -cover -coverprofile=coverage.out .

cover:
	go tool cover -func=coverage.out

lint:
	go vet .
	golangci-lint run
