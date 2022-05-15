package main

import (
	"context"

	"github.com/dell/gocsi"

	"/root/francis/grpc/newcsi/provider"
	"/root/francis/grpc/newcsi/service"
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
