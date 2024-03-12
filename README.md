# Cloud Run SSH image

## Building the image
```sh
gcloud builds submit --config cloudbuild.yaml
```

## Deploying the image
Replace the following variables below
- _SERVICE_NAME
- _PROJECT_ID
- _REGION

and run the command:
```sh
gcloud run deploy $_SERVICE_NAME --image=gcr.io/$_PROJECT_ID/$_SERVICE_NAME \
--region=$_REGION --port=8080 --min-instances=1 \
--max-instances=1 --timeout=3600s --no-use-http2 --session-affinity \
--memory=4Gi --cpu=2 --cpu-boost --no-cpu-throttling --execution-environment=gen2 \
--allow-unauthenticated
```

## SSHing into the container
1. Open the service's URL
1. Fill in the following fields:
 - Hostname: `127.0.0.1` (fixed)
 - Port: `2222` (fixed)
 - Username: $_USER_NAME # identical to the value in cloudbuild.yaml
 - Password: $_USER_PASS # identical to the value in cloudbuild.yaml
1. Click `Connect`