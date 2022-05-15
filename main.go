package main

import (
	"context"

	"github.com/dell/gocsi"
)

// main is ignored when this package is built as a go plug-in.
func main() {
	retreiver.Run(
		context.Background(),
		service.Name,
		"A description of the SP",
		"",
		provider.New())
}
