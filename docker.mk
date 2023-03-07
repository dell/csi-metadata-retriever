#
#
# Copyright © 2022 Dell Inc. or its subsidiaries. All Rights Reserved.
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

# for variables override
-include vars.mk

# Includes the following generated file to get semantic version information
ifdef NOTES
	RELNOTE="$(NOTES)"
else
	RELNOTE=
endif

ifndef DOCKER_REGISTRY
	DOCKER_REGISTRY=dellemc
endif

ifndef DOCKER_IMAGE_NAME
    DOCKER_IMAGE_NAME=csi-metadata-retriever
endif

ifndef BASEIMAGE
	BASEIMAGE=ubi-minimal:8.7-1085
endif

# figure out if podman or docker should be used (use podman if found)
ifneq (, $(shell which podman 2>/dev/null))
	BUILDER=podman
else
	BUILDER=docker
endif

ifndef MAJOR
	MAJOR=1
endif

ifndef MINOR
	MINOR=0
endif

ifndef PATCH
	PATCH=0
endif

docker:
	echo "MAJOR $(MAJOR) MINOR $(MINOR) PATCH $(PATCH) RELNOTE $(RELNOTE)"
	echo "$(DOCKER_FILE)"
	$(BUILDER) build -f $(DOCKER_FILE) -t "$(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME):v$(MAJOR).$(MINOR).$(PATCH)$(RELNOTE)" --build-arg BASEIMAGE=$(BASEIMAGE) .

docker-no-cache:
	echo "MAJOR $(MAJOR) MINOR $(MINOR) PATCH $(PATCH) RELNOTE $(RELNOTE)"
	echo "$(DOCKER_FILE) --no-cache"
	$(BUILDER) build --no-cache --pull -f $(DOCKER_FILE) -t "$(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME):v$(MAJOR).$(MINOR).$(PATCH)$(RELNOTE)" --build-arg BASEIMAGE=$(BASEIMAGE) .


push:   
	echo "MAJOR $(MAJOR) MINOR $(MINOR) PATCH $(PATCH) RELNOTE $(RELNOTE)"
	$(BUILDER) push "$(DOCKER_REGISTRY)/$(DOCKER_IMAGE_NAME):v$(MAJOR).$(MINOR).$(PATCH)$(RELNOTE)"
