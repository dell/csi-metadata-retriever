#
#
# Copyright © 2022 - 2023 Dell Inc. or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#      http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
#

NAME:=csi-metadata-retriever

# Tag parameters
ifndef MAJOR
    MAJOR=1
endif
ifndef MINOR
    MINOR=6
endif
ifndef PATCH
    PATCH=0
endif
ifndef NOTES
	NOTES=
endif


.PHONY: all

all: help

# include an overrides file, which sets up default values and allows user overrides
include overrides.mk

# Help target, prints usefule information
help:
	@echo
	@echo "The following targets are commonly used:"
	@echo
	@echo "go-build         - Builds the code locally"
	@echo "check            - Runs the suite of code checking tools: lint, format, etc"
	@echo "clean            - Cleans the local build"
	@echo "docker           - Builds the code within a golang container and then creates the driver image"
	@echo "push             - Pushes the built container to a target registry"
	@echo "test             - Runs the unit tests"
	@echo
	@make -s overrides-help

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
	
# Generates the docker container (but does not push)
docker:
	go generate .
	make -f docker.mk DOCKER_FILE=docker-files/Dockerfile docker

# Same as `docker` but without cached layers and will pull latest version of base image
docker-no-cache:
	go generate .
	make -f docker.mk DOCKER_FILE=docker-files/Dockerfile docker-no-cache


# Pushes container to the repository
push:	docker
		make -f docker.mk push

check:	gosec
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
