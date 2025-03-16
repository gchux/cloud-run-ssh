# syntax=docker/dockerfile:1.4
#
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

ARG SSH_SERVER_VERSION=amd64-noble-cdec982b-ls31

FROM ghcr.io/linuxserver/baseimage-ubuntu:${SSH_SERVER_VERSION}
# published as: ghcr.io/gchux/cloud-run-ssh:ubuntu-latest

ARG DEBIAN_FRONTEND=noninteractive

ARG VERSION=1:9.6p1-3ubuntu13

LABEL build_version="${VERSION}"
LABEL maintainer="chux"

RUN \
  echo "**** install runtime packages ****" && \
  apt-get -qq update > /dev/null && \
  apt-get -qq -yy install --no-install-recommends \
  logrotate \
  nano \
  netcat-openbsd \
  sudo \
  > /dev/null && \
  echo "**** install openssh-server ****" && \
  apt-get -qq -yy install --no-install-recommends \
  openssh-server=${VERSION} \
  openssh-client=${VERSION} \
  openssh-sftp-server=${VERSION} \
  > /dev/null && \
  apt-get -qq clean > /dev/null

RUN \
  echo "**** setup openssh environment ****" && \
  echo "Cloud Run SSH server version: ${VERSION}" > /build_version && \
  sed -i 's/#PasswordAuthentication yes/PasswordAuthentication no/g' /etc/ssh/sshd_config && \
  usermod --shell /bin/bash abc && \
  rm -rf /tmp/* "${HOME}/.cache"

# add local files
COPY /root /

EXPOSE 2222

VOLUME /config
