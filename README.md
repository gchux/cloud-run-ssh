# cloud-run-ssh

```shell
docker build run-ssh-server:latest \
--build-arg="CLOUDSDK_VERSION=..." \
--build-arg="CSQL_PROXY_VERSION=..." \
--build-arg="ALLOYDB_PROXY_VERSION=..." \
--build-arg="USQL_VERSION=..." \
--build-arg="SERVICE_PORT=..." \
--build-arg="USER_NAME=..." \
--build-arg="USER_PASS=..." \
-t  .
```