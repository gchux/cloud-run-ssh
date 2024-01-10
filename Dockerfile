# the base image: https://github.com/linuxserver/docker-openssh-server
FROM linuxserver/openssh-server

ARG CLOUDSDK_VERSION
ARG CSQL_PROXY_VERSION
ARG ALLOYDB_PROXY_VERSION
ARG USQL_VERSION
ARG SERVICE_PORT
ARG SSH_USER
ARG SSH_PASS

ENV SUDO_ACCESS=true
ENV PASSWORD_ACCESS=true
ENV USER_NAME=${SSH_USER}
ENV USER_PASSWORD=${SSH_PASS}
ENV HTTP_PORT=${SERVICE_PORT}

RUN apk update
RUN apk add busybox-extras net-tools bind-tools iproute2 curl tmux git \
    bc traceroute tcptraceroute tcpdump mtr nmap redis python3 py3-pip
RUN wget -P /usr/bin http://www.vdberg.org/~richard/tcpping
RUN chmod a+rx /usr/bin/tcpping
RUN python -m pip install httpie webssh

RUN wget -P / https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${CLOUDSDK_VERSION}-linux-x86_64.tar.gz
RUN tar -xzvf /google-cloud-cli-${CLOUDSDK_VERSION}-linux-x86_64.tar.gz -C / \
    && rm -vf /google-cloud-cli-${CLOUDSDK_VERSION}-linux-x86_64.tar.gz
RUN /google-cloud-sdk/bin/gcloud components install cbt --quiet
RUN ln -s /google-cloud-sdk/bin/* /usr/bin/
RUN echo "export PATH=$PATH:/google-cloud-sdk/bin" >> ~/.bashrc
ENV PATH="$PATH:/google-cloud-sdk/bin"

RUN curl -o /cloud-sql-proxy https://storage.googleapis.com/cloud-sql-connectors/cloud-sql-proxy/v${CSQL_PROXY_VERSION}/cloud-sql-proxy.linux.amd64
RUN chmod a+x /cloud-sql-proxy \
    && ln -s /cloud-sql-proxy /usr/bin/cloud-sql-proxy

RUN wget https://storage.googleapis.com/alloydb-auth-proxy/v${ALLOYDB_PROXY_VERSION}/alloydb-auth-proxy.linux.amd64 -O /alloydb-auth-proxy \
    && chmod a+x /alloydb-auth-proxy \
    && ln -s /alloydb-auth-proxy /usr/bin/alloydb-auth-proxy

RUN wget -P / https://github.com/xo/usql/releases/download/v${USQL_VERSION}/usql_static-${USQL_VERSION}-linux-amd64.tar.bz2
RUN tar -xvf /usql_static-${USQL_VERSION}-linux-amd64.tar.bz2 \
    && rm -vf /usql_static-${USQL_VERSION}-linux-amd64.tar.bz2 \
    && ln -s /usql_static /usr/bin/usql

RUN echo -n "${HTTP_PORT}" > /http.port

EXPOSE ${HTTP_PORT}/tcp

# web ssh terminal: https://github.com/huashengdun/webssh
CMD ["/bin/bash", "-c", "export HTTP_PORT=$(cat /http.port) && exec wssh --port=${HTTP_PORT} --xheaders=False"]