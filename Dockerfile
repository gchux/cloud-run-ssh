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

FROM --platform=linux/amd64 linuxserver/openssh-server
# the base image: https://github.com/linuxserver/docker-openssh-server

ARG WEB_PORT
ARG SSH_USER
ARG SSH_PASS

ARG GCSFUSE_VERSION=2.1.0

ENV USER_NAME=${SSH_USER}
ENV USER_PASSWORD=${SSH_PASS}
ENV HTTP_PORT=${WEB_PORT}

ENV SUDO_ACCESS=true
ENV PASSWORD_ACCESS=true
ENV LOG_STDOUT=true

ENV PUID=0
ENV PGID=0

USER 0:0

RUN apk update && apk add dpkg fuse vim busybox-extras net-tools bind-tools iproute2 \
    curl tmux git bc traceroute tcptraceroute tcpdump mtr nmap redis \
    python3-dev py3-pip gcc libc-dev libffi-dev
RUN wget -P /usr/bin http://www.vdberg.org/~richard/tcpping && chmod a+rx /usr/bin/tcpping
RUN python -m pip install --break-system-packages httpie webssh

RUN curl -o /gcsfuse.deb -L \
    https://github.com/GoogleCloudPlatform/gcsfuse/releases/download/v${GCSFUSE_VERSION}/gcsfuse_${GCSFUSE_VERSION}_amd64.deb \
    && dpkg -i --force-all /gcsfuse.deb && rm -vf /gcsfuse.deb

RUN echo -n "${HTTP_PORT}" > /http.port

EXPOSE ${HTTP_PORT}/tcp

RUN if [[ "${USER_NAME}" == "root" ]] ; then echo "root:${USER_PASSWORD}" | chpasswd && echo -e '\nPermitRootLogin yes' >> /etc/ssh/sshd_config; fi

# web ssh terminal: https://github.com/huashengdun/webssh
CMD ["/bin/bash", "-c", "export HTTP_PORT=$(cat /http.port | tr -d '\n') && exec env wssh --address='0.0.0.0' --port=${HTTP_PORT} --encoding='utf-8' --xheaders=True --debug=True --origin='*' --policy=autoadd --log-to-stderr --logging=debug --fbidhttp=False --maxconn=10 --xsrf=False"]
