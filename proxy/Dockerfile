# syntax=docker/dockerfile:1.4
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG GOLANG_VERSION=1.22.4

FROM --platform=linux/amd64 golang:${GOLANG_VERSION}-bookworm AS builder

ARG GOLANG_VERSION=1.22.4
ARG YQ_VERSION=4.44.3
ARG GOST_VERSION=3.0.0-nightly.20241022
ARG BIN_NAME=ssh_proxy_server

WORKDIR /app

COPY ./go.mod go.mod
COPY ./go.sum go.sum
COPY ./main.go main.go
COPY ./pkg pkg

RUN go install mvdan.cc/gofumpt@latest

ENV GO111MODULE=on
ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

RUN go mod tidy -compat=${GOLANG_VERSION}
RUN go mod download
RUN gofumpt -l -w ./main.go
RUN CGO_ENABLED=0 go build -a -v -o ${BIN_NAME} main.go && mv ${BIN_NAME} /${BIN_NAME} && chmod +x /${BIN_NAME}

RUN curl -sL -o yq https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_amd64 &&  mv -v yq /yq  && chmod +x /yq

RUN curl -sL -o gost.tar.gz \
    https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_amd64.tar.gz \
    && tar -xzvf gost.tar.gz && mv -v gost /gost && chmod +x /gost && rm -vf gost.tar.gz

FROM --platform=linux/amd64 ubuntu:22.04

ARG BIN_NAME=ssh_proxy_server

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /${BIN_NAME} /ssh_proxy_server
COPY --from=builder /gost /gost
COPY --from=builder /yq /yq
COPY ./ssh_proxy_server.yaml /ssh_proxy_server.yaml
COPY ./start_ssh_proxy_server /start_ssh_proxy_server

RUN mkdir -p /etc/ssh_proxy_server/

ENTRYPOINT ["/start_ssh_proxy_server"]

CMD ["/start_ssh_proxy_server"]
