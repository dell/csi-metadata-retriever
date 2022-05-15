NAME:=csi-metadata-retriever

.PHONY: all
all: go-build

ifneq (on,$(GO111MODULE))
export GO111MODULE := on
endif

.PHONY: go-vendor
go-vendor:
	go mod vendor

.PHONY: go-build
go-build: clean
	go build .
	
.PHONY: clean
clean:
	go clean
