# Copyright © 2024 Dell Inc. or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#      http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

# overrides file
# this file, included from the Makefile, will overlay default values with environment variables
#

# DEFAULT values


# DEFAULT values
DEFAULT_GOIMAGE=$(shell sed -En 's/^go (.*)$$/\1/p' go.mod)
DEFAULT_REGISTRY="dellemc"
DEFAULT_IMAGENAME="csi-metadata-retriever"


# set the GOIMAGE if needed
ifeq ($(GOIMAGE),)
export GOIMAGE="$(DEFAULT_GOIMAGE)"
endif

# set the REGISTRY if needed
ifeq ($(REGISTRY),)
export REGISTRY="$(DEFAULT_REGISTRY)"
endif

# set the IMAGENAME if needed
ifeq ($(IMAGENAME),)
export IMAGENAME="$(DEFAULT_IMAGENAME)"
endif


# figure out if podman or docker should be used (use podman if found)
ifneq (, $(shell which podman 2>/dev/null))
export BUILDER=podman
else
export BUILDER=docker
endif

ifdef NOTES
	RELNOTE="$(NOTES)"
else
	RELNOTE=
endif

ifndef MAJOR
	MAJOR=1
endif
ifndef MINOR
	MINOR=12
endif
ifndef PATCH
	PATCH=0
endif

# target to print some help regarding these overrides and how to use them
overrides-help:
	@echo
	@echo "The following environment variables can be set to control the build"
	@echo
	@echo "GOIMAGE   - The version of Go to build with, default is: $(DEFAULT_GOIMAGE)"
	@echo "              Current setting is: $(GOIMAGE)"
	@echo "REGISTRY    - The registry to push images to, default is: $(DEFAULT_REGISTRY)"
	@echo "              Current setting is: $(REGISTRY)"
	@echo "IMAGENAME   - The image name to be built, defaut is: $(DEFAULT_IMAGENAME)"
	@echo "              Current setting is: $(IMAGENAME)"
	@echo

