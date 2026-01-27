# Copyright © 2022-2026 Dell Inc. or its subsidiaries. All Rights Reserved.
#
# Dell Technologies, Dell and other trademarks are trademarks of Dell Inc.
# or its subsidiaries. Other trademarks may be trademarks of their respective 
# owners.


.PHONY: all

all: help

# include an overrides file, which sets up default values and allows user overrides
include images.mk

# Help target, prints usefule information
help:
	@echo
	@echo "The following targets are commonly used:"
	@echo
	@echo "build            - Builds the code locally, you may need to run make vendor"
	@echo "check            - Runs the suite of code checking tools: lint, format, etc"
	@echo "clean            - Cleans the local build"
	@echo "test             - Runs the unit tests"
	@echo
	@make -s overrides-help

build:
	GOOS=linux CGO_ENABLED=0 go build -mod=vendor .

check: gosec
	gofmt -w ./.
	golint -set_exit_status ./.
	go vet ./...

gosec:
	gosec -quiet -log gosec.log -out=gosecresults.csv -fmt=csv ./...

test:
	rm -rf /tmp/csi_retriever_test.sock
	go clean -cache; cd ./retriever; go test -race -cover -coverprofile=coverage.out -coverpkg ./... ./...
	rm -rf /tmp/csi_retriever_test.sock

coverage:
	cd ./retriever; go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf vendor
	rm -f csm-common.mk
	go clean

