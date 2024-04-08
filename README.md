# Cloud Run SSH image

## Building the image

### Using Docker

Populate the variables and run the command:

```sh
docker build -t <image-name>:<image-tag> \
  --build-arg CLOUDSDK_VERSION=459.0.0 \
  --build-arg CSQL_PROXY_VERSION=2.8.1 \
  --build-arg ALLOYDB_PROXY_VERSION=1.6.1 \
  --build-arg USQL_VERSION=0.17.5 \
  --build-arg SERVICE_PORT=8080 \
  --build-arg SSH_USER=user \
  --build-arg SSH_PASS=123123 \
  . 
```

### Using Cloud Build

```sh
gcloud builds submit --config cloudbuild.yaml \
--substitutions '_REPO_LOCATION=<repo-location>,_REPO_NAME=<repo-name>,_IMAGE_NAME=<image-name>,_IMAGE_TAG=<image-tag>,_BUILD_TAG=<build-tag>' .
```
## (Optional) Using dynamic credentials

### Generating credentials
Instead of using the default (weak) provided values for the username and password, we can generate a username with `uuidgen` and a password with `openssl rand`.

```sh
chmod +x generate_credentials.sh
./generate_credentials.sh 
```
Place the output values in the `cloudbuild.yaml`'s `_USER_NAME` and `_USER_PASS`.

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
--no-allow-unauthenticated
```

## SSHing into the container

1. `gcloud run services proxy ${SERVICE_NAME} --region=${SERVICE_REGION} --port=8080`

    > see: https://cloud.google.com/sdk/gcloud/reference/run/services/proxy

2. Use a WEB Browser, got to: `http://127.0.0.1:8080/`

2. Fill in the following fields:
    - Hostname: `127.0.0.1` _(fixed)_
    - Port: `2222` _(fixed)_
    - Username: `$_USER_NAME` # identical to the value in cloudbuild.yaml
    - Password: `$_USER_PASS` # identical to the value in cloudbuild.yaml

3. Click `Connect`

## References
- https://www.openssl.org/docs/man1.1.1/man1/rand.html