NAME:=csi-metadata-retriever

# Dockerfile defines which base image to use [Dockerfile.centos, Dockerfile.ubi, Dockerfile.ubi.min, Dockerfile.ubi.alt]
# e.g.:$ make docker DOCKER_FILE=Dockerfile.ubi.alt
ifndef DOCKER_FILE
    DOCKER_FILE = Dockerfile.ubi.min
endif

# Tag parameters
ifndef MAJOR
    MAJOR=1
endif
ifndef MINOR
    MINOR=0
endif
ifndef PATCH
    PATCH=0
endif
ifndef NOTES
	NOTES=
endif


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
	
# Generates the docker container (but does not push)
docker:
	go generate .
	make -f docker.mk DOCKER_FILE=docker-files/$(DOCKER_FILE) docker

# Same as `docker` but without cached layers and will pull latest version of base image
docker-no-cache:
	go generate .
	make -f docker.mk DOCKER_FILE=docker-files/$(DOCKER_FILE) docker-no-cache


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
