# Cloud Run SSH image

## Image flavors

### By contents

- `lite`: contains the bare minimum tools to perform network troubleshooting.
- `full`: contains additional Google Cloud SDK and third-party tools.

> [!TIP]
> choose `lite` if all you need to do is network troubleshooting; build-time and size will be reduced.

### By access level

- `root`: whether to allow `sudo` to escalate privileges.
- `no-root`: `sudo` is disabled; the logged in user won't be able to escalate privileges.

> [!TIP]
> choose `no-root` if you are providing this image to others so that installing new software is disabled.

## Building the image

### Configure environment

```sh
export CONTENT_FLAVOR='...'        # `lite` or `full` depending on the required tooling
export ACCESS_LEVEL='...'          # `root` or `no-root` depending on the required access level
export SUDO_ACCESS='false'         # `true`/`false` depending on access level selection

export CLOUDSDK_VERSION='...'      # see: https://console.cloud.google.com/storage/browser/cloud-sdk-release
export GCSFUSE_VERSION='...'       # see: https://github.com/GoogleCloudPlatform/gcsfuse/releases
export CSQL_PROXY_VERSION='...'    # see: https://github.com/GoogleCloudPlatform/cloud-sql-proxy/releases
export ALLOYDB_PROXY_VERSION='...' # see: https://github.com/GoogleCloudPlatform/alloydb-auth-proxy/releases

export USQL_VERSION='...'          # see: https://github.com/xo/usql/releases

export SSH_USER='user'             # whatever user you want to use to login into the SSH server
export SSH_PASS='pass'             # whatever password you want to use to login into the SSH server

export WEB_PORT=8080               # whatever TCP port you want to use to server the WEB SSH server

export IMAGE_NAME='cloud-run-ssh'  # or whatever name you need/require to use for the Docker image
export IMAGE_TAG="${CONTENT_FLAVOR}-${ACCESS_LEVEL}"

export DOCKERFILE="Dockerfile.${IMAGE_TAG}"

export PROJECT_ID='...'            # GCP Project ID ( this is an alphanumeric name, not a number )
export REPO_LOCATION='...'         # Artifact Registry docker repository location
export REPO_NAME='...'             # Artifact Registry docker repository name

export IMAGE_URI_BASE="${REPO_LOCATION}-docker.pkg.dev/${PROJECT_ID}/${REPO_NAME}"
export IMAGE_URI_FULL="${IMAGE_URI_BASE}/${IMAGE_NAME}:${IMAGE_TAG}"

export BUILD_TAG='...'             # whatever tag you may/need to use
```

or create a copy of `env.sample` named `.env`, edit it with your desired values, and source it:

```sh
cp -v env.sample .env
# edit `.env` to contain your desired values
source $(pwd)/.env
```

> [!NOTE]
> creating a copy of `env.sample` with your custom configuration is the better approach as you can expand this pattern to multiple `env files` in order to create various builds.

### Using Docker

Populate the variables and run the command:

```sh
./docker_build ${CONTENT_FLAVOR} ${ACCESS_LEVEL} ${IMAGE_URI_BASE} ${SSH_USER} ${SSH_PASS} ${WEB_PORT}
```

alternatively, you may run the `docker` command directly:

```sh
# update versions of dependencies
docker buildx build --no-cache \
  --platform=linux/amd64 \
  --file=$(pwd)/${DOCKERFILE} \
  --tag="${IMAGE_URI_FULL}" \
  --build-arg="SSH_USER=${SSH_USER}" \
  --build-arg="SSH_PASS=${SSH_PASS}" \
  --build-arg="WEB_PORT=8080" \
  --build-arg="GCSFUSE_VERSION=${GCSFUSE_VERSION}" \
  --build-arg="CLOUDSDK_VERSION=${CLOUDSDK_VERSION}" \
  --build-arg="CSQL_PROXY_VERSION=${CSQL_PROXY_VERSION}" \
  --build-arg="ALLOYDB_PROXY_VERSION=${ALLOYDB_PROXY_VERSION}" \
  --build-arg="USQL_VERSION=${USQL_VERSION}" \
  $(pwd)
```

### Using Cloud Build

Adjust environment variables as per your requirements:

```sh
gcloud builds submit --config cloudbuild.yaml \
--substitutions "_REPO_LOCATION=${REPO_LOCATION},_REPO_NAME=${REPO_NAME},_IMAGE_NAME=${IMAGE_NAME},_IMAGE_TAG=${IMAGE_TAG},_BUILD_TAG=${BUILD_TAG},_WEB_PORT=${WEB_PORT},_SSH_USER=${SSH_USER},_SSH_PASS=${SSH_PASS},_CLOUDSDK_VERSION=${CLOUDSDK_VERSION},_GCSFUSE_VERSION=${GCSFUSE_VERSION},_CSQL_PROXY_VERSION=${CSQL_PROXY_VERSION},_ALLOYDB_PROXY_VERSION=${ALLOYDB_PROXY_VERSION},_USQL_VERSION=${USQL_VERSION},_DOCKERFILE=${DOCKERFILE}" \
$(pwd)
```

## Deploying the image to Cloud Run

```sh
export SERVICE_NAME='...'
export SERVICE_REGION='...'

gcloud run deploy ${SERVICE_NAME} --image=${IMAGE_URI_FULL} \
--region=${SERVICE_REGION} --port=8080 --min-instances=0 \
--max-instances=1 --timeout=3600s --no-use-http2 \
--session-affinity --memory=2Gi --cpu=2 --cpu-boost \
--no-cpu-throttling --execution-environment=gen2 \
--set-env-vars="SUDO_ACCESS=${SUDO_ACCESS},PASSWORD_ACCESS=true,LOG_STDOUT=true" \
--no-allow-unauthenticated
```

> [!CAUTION]
> it is strongly recommended to use **`--no-allow-unauthenticated`** in order to prevent unauthorized access to the SSH server.

## SSHing into the container

1. `gcloud run services proxy ${SERVICE_NAME} --region=${SERVICE_REGION} --port=8080`

   > see: https://cloud.google.com/sdk/gcloud/reference/run/services/proxy

2. Use a WEB browser, got to: `http://127.0.0.1:8080/`

3. Fill in the following fields:

   - Hostname: `127.0.0.1` _(fixed)_
   - Port: `2222` _(fixed)_
   - Username: `${SSH_USER}`
   - Password: `${SSH_PASS}`

4. Click `Connect`
