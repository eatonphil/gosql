package main

import (
	"github.com/eatonphil/gosql"
)

func main() {
	mb := gosql.NewMemoryBackend()

	gosql.RunRepl(mb)
}
