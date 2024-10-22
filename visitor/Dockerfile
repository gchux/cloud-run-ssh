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

FROM --platform=linux/amd64 ubuntu:22.04

ARG DEBIAN_FRONTEND=noninteractive
ARG GOST_VERSION=3.0.0-rc10

RUN apt-get -qq update  > /dev/null \
    && apt-get -qq -y install openssl wget curl jq > /dev/null \
    && apt-get -qq clean > /dev/null

COPY ./start_ssh_proxy_visitor /start_ssh_proxy_visitor
COPY ./ssh_proxy_visitor.yaml /ssh_proxy_visitor.yaml

RUN wget -O gost.tar.gz \
    https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_amd64.tar.gz \
    && tar -xzvf gost.tar.gz && chmod +x gost && rm -vf gost.tar.gz

ENTRYPOINT ["/start_ssh_proxy_visitor"]

CMD ["/start_ssh_proxy_visitor"]
