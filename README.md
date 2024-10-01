# Cloud Run SSH image

## Building the image

### Using Docker

Populate the variables and run the command:

```sh
./docker_build <lite-or-full> <[no-]root> <docker-repo> <ssh-user> <ssh-pass> <web-port>
```

alternatively, you may run the `docker` command directly:

```sh
# update versions of dependencies
docker buildx build --no-cache \
  --platform=linux/amd64 \
  --file=Dockerfile.<lite-or-full>-<[no-]root> \
  --tag=<docker-repo>/<image-name>:<image-tag> \
  --build-arg=GCSFUSE_VERSION=2.1.0 \
  --build-arg=CLOUDSDK_VERSION=459.0.0 \
  --build-arg=CSQL_PROXY_VERSION=2.8.1 \
  --build-arg=ALLOYDB_PROXY_VERSION=1.6.1 \
  --build-arg=USQL_VERSION=0.17.5 \
  --build-arg=WEB_PORT=8080 \
  --build-arg=SSH_USER=user \
  --build-arg=SSH_PASS=pass \
  $(pwd)
```

### Using Cloud Build

Adjust environment variables as per your requirements:

```sh
export DOCKERFILE='<lite-or-fill>-[no-]root'

export CLOUDSDK_VERSION='...'      # see: https://console.cloud.google.com/storage/browser/cloud-sdk-release
export GCSFUSE_VERSION='...'       # see: https://github.com/GoogleCloudPlatform/gcsfuse/releases
export CSQL_PROXY_VERSION='...'    # see: https://github.com/GoogleCloudPlatform/cloud-sql-proxy/releases
export ALLOYDB_PROXY_VERSION='...' # see: https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/releases

export USQL_VERSION='...'          # see: https://github.com/xo/usql/releases

export REPO_LOCATION='...'         # Artifact Registry docker repository location
export REPO_NAME='...'             # Artifact Registry docker repository name
export IMAGE_NAME='cloud-run-ssh'
export IMAGE_TAG='latest'
export BUILD_TAG='...'             # whatever tag you may/need to use
export SSH_USER='user'             # whatever user you want to use to login into the SSH server
export SSH_USER='pass'             # whatever password you want to use to login into the SSH server

gcloud builds submit --config cloudbuild.yaml \
--substitutions "_REPO_LOCATION=${REPO_LOCATION},_REPO_NAME=${REPO_LOCATION},_IMAGE_NAME=${IMAGE_NAME},_IMAGE_TAG=${IMAGE_TAG},_BUILD_TAG=${BUILD_TAG},_WEB_PORT=8080,_SSH_USER=${SSH_USER},_SSH_PASS=${SSH_PASS},_CLOUDSDK_VERSION=${CLOUDSDK_VERSION},_GCSFUSE_VERSION=${GCSFUSE_VERSION},_CSQL_PROXY_VERSION=${CSQL_PROXY_VERSION},_ALLOYDB_PROXY_VERSION=${ALLOYDB_PROXY_VERSION},_USQL_VERSION=${USQL_VERSION},_DOCKERFILE=${DOCKERFILE}" .
```

## Deploying the image to Cloud Run

```sh
export SERVICE_NAME='<service-name>'
export SERVICE_REGION='<service-region>'
export IMAGE_URI='<image-uri>'

gcloud run deploy ${SERVICE_NAME} --image=${IMAGE_URI} \
--region=${SERVICE_REGION} --port=8080 --min-instances=0 \
--max-instances=1 --timeout=3600s --no-use-http2 \
--session-affinity --memory=2Gi --cpu=2 --cpu-boost \
--no-cpu-throttling --execution-environment=gen2 \
--set-env-vars='SUDO_ACCESS=<true-or-false>,PASSWORD_ACCESS=true,LOG_STDOUT=true' \
--no-allow-unauthenticated
```

> [!IMPORTANT]
> it is strongly recommended to use **`--no-allow-unauthenticated`** in order to prevent unauthorized access to the SSH server.

## SSHing into the container

1. `gcloud run services proxy ${SERVICE_NAME} --region=${SERVICE_REGION} --port=8080`

   > see: https://cloud.google.com/sdk/gcloud/reference/run/services/proxy

2. Use a WEB Browser, got to: `http://127.0.0.1:8080/`

3. Fill in the following fields:

   - Hostname: `127.0.0.1` _(fixed)_
   - Port: `2222` _(fixed)_
   - Username: `$_USER_NAME` # identical to the value in cloudbuild.yaml
   - Password: `$_USER_PASS` # identical to the value in cloudbuild.yaml

4. Click `Connect`
