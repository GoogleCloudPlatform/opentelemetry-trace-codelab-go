# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.18.3-bullseye as builder
WORKDIR /build
COPY . .
ENV CGO_ENABLED=0
RUN go build -o server .
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.11
ARG PROBE_BIN_URL=https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64
RUN wget -qO /build/grpc_health_probe $PROBE_BIN_URL && \
    chmod +x /build/grpc_health_probe

FROM gcr.io/distroless/base-debian11
WORKDIR /svc
COPY --from=builder /build/server /svc/server
COPY --from=builder /build/grpc_health_probe /bin/grpc_health_probe
EXPOSE 5050
ENTRYPOINT ["/svc/server"]
