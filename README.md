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

```sh
export DOCKERFILE='<lite-or-fill>-[no-]root'

gcloud builds submit --config cloudbuild.yaml \
--substitutions "_REPO_LOCATION=<repo-location>,_REPO_NAME=<repo-name>,_IMAGE_NAME=<image-name>,_IMAGE_TAG=<image-tag>,_BUILD_TAG=<build-tag>,_WEB_PORT=8080,_SSH_USER=<username>,_SSH_PASS=<password>,_DOCKERFILE=${DOCKERFILE}" .
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
