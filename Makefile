fmt:
	gofmt -w -s .

test:
	go test -race -cover -coverprofile=coverage.out .

cover: 
	go tool cover -func=coverage.out

vet:
	go vet .
