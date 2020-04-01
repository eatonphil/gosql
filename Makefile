fmt:
	gofmt -w -s .

test:
	go test -race .
