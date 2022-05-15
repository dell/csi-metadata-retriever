package main

import (
	"context"

	"github.com/dell/csi-metadata-retriever/provider"
	"github.com/dell/csi-metadata-retriever/retriever"
)

// main is ignored when this package is built as a go plug-in.
func main() {
	retreiver.Run(
		context.Background(),
		"MetadataRetriever",
		"A description of the SP",
		"",
		provider.New())
}
