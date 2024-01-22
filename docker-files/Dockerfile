#
#
# Copyright © 2023 - 2024 Dell Inc. or its subsidiaries. All Rights Reserved.
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
# Dockerfile to build CSI Metadata Retriever sidecar
# some arguments that must be supplied
ARG GOVERSION
ARG BASEIMAGE
ARG VERSION
# Stage to build the driver
FROM golang:${GOVERSION} as builder

RUN mkdir -p /go/src
COPY ./ /go/src/
WORKDIR /go/src/
RUN CGO_ENABLED=0 \
    make go-build

# Stage to build the driver image
FROM $BASEIMAGE AS final

COPY --from=builder /go/src/csi-metadata-retriever /
LABEL vendor="Dell Inc." \
      name="csi-metadata-retriever" \
      summary="CSI Metadata Retriever sidecar" \
      description="CSI Metadata Retriever sidecar for metadata retrievel via kubeapi" \
      version=$(VERSION) \
      license="Apache-2.0"
COPY licenses /licenses
ENTRYPOINT ["/csi-metadata-retriever"]
