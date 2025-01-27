# Copyright © 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
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

# for variables override
-include vars.mk
include overrides.mk

docker: download-csm-common
	$(eval include csm-common.mk)
	echo "Building: $(REGISTRY)/$(IMAGENAME):v$(MAJOR).$(MINOR).$(PATCH) RELNOTE $(RELNOTE)"
	echo "$(DOCKER_FILE)"
	$(BUILDER) build --pull -f $(DOCKER_FILE) -t "$(REGISTRY)/$(IMAGENAME):v$(MAJOR).$(MINOR).$(PATCH)$(RELNOTE)" --build-arg BASEIMAGE=$(CSM_BASEIMAGE) --build-arg GOIMAGE=$(DEFAULT_GOIMAGE) .

docker-no-cache: download-csm-common
	$(eval include csm-common.mk)
	echo "Building: $(REGISTRY)/$(IMAGENAME):$(MAJOR).$(MINOR).$(PATCH) RELNOTE $(RELNOTE)"
	echo "$(DOCKER_FILE) --no-cache"
	$(BUILDER) build --pull --no-cache --pull -f $(DOCKER_FILE) -t "$(REGISTRY)/$(IMAGENAME):v$(MAJOR).$(MINOR).$(PATCH)$(RELNOTE)" --build-arg BASEIMAGE=$(CSM_BASEIMAGE) --build-arg GOIMAGE=$(DEFAULT_GOIMAGE) .

push:
	echo "Pushing MAJOR $(MAJOR) MINOR $(MINOR) PATCH $(PATCH) RELNOTE $(RELNOTE)"
	$(BUILDER) push "$(REGISTRY)/$(IMAGENAME):v$(MAJOR).$(MINOR).$(PATCH)$(RELNOTE)"

download-csm-common:
	curl -O -L https://raw.githubusercontent.com/dell/csm/main/config/csm-common.mk
