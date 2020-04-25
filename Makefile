fmt:
	gofmt -w -s .

test:
	go test -race -cover .

vet:
	go vet .
