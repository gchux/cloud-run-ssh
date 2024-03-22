# Cloud Run SSH image

## Building the image

```sh
gcloud builds submit --config cloudbuild.yaml \
--substitutions '_REPO_LOCATION=<repo-location>,_REPO_NAME=<repo-name>,_IMAGE_NAME=<image-name>,_IMAGE_TAG=<image-tag>,_BUILD_TAG=<build-tag>' .
```

## Deploying the image

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

2. Fill in the following fields:
    - Hostname: `127.0.0.1` (fixed)
    - Port: `2222` (fixed)
    - Username: $_USER_NAME # identical to the value in cloudbuild.yaml
    - Password: $_USER_PASS # identical to the value in cloudbuild.yaml

3. Click `Connect`
