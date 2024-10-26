# Cloud Run SSH server

![cloud_run_ssh](https://github.com/gchux/cloud-run-ssh/blob/main/pix/cloud_run_ssh.png?raw=true)

## Motivation

During development and while troubleshooting issues, it is often very useful to test some preconditions in the same execution environment as the microservice while having more maneuverability.

The **`Cloud Run SSH server`** aims to provide full linux shell running in a Cloud Run service instance to allow developers and administrators to test and troubleshoot all kinds of scenarios.

## Building Blocks:

- [docker-openssh-server](https://github.com/linuxserver/docker-openssh-server): provides the SSH server.
- [webssh](https://pypi.org/project/webssh/): provides web access to the SSH server and a browser based shell to execute commands.

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

export CREATE_KEY_PAIR='false'     # when using `docker_build`, whether to create a new KEY pair
export PASSWORD_ACCESS='true'      # allows to disable password authentication if KEY based auth is preferred
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
./docker_build
```

alternatively, you may run the `docker` command directly:

```sh
# update versions of dependencies
docker buildx build \
  --no-cache --push \
  --platform=linux/amd64 \
  --file=$(pwd)/${DOCKERFILE} \
  --tag="${IMAGE_URI_FULL}" \
  --build-arg="SSH_USER=${SSH_USER}" \
  --build-arg="SSH_PASS=${SSH_PASS}" \
  --build-arg="WEB_PORT=${WEB_PORT}" \
  --build-arg="PASSWORD_ACCESS=${PASSWORD_ACCESS}" \
  --build-arg="SUDO_ACCESS=${SUDO_ACCESS}" \
  --build-arg="GCSFUSE_VERSION=${GCSFUSE_VERSION}" \
  --build-arg="CLOUDSDK_VERSION=${CLOUDSDK_VERSION}" \
  --build-arg="CSQL_PROXY_VERSION=${CSQL_PROXY_VERSION}" \
  --build-arg="ALLOYDB_PROXY_VERSION=${ALLOYDB_PROXY_VERSION}" \
  --build-arg="USQL_VERSION=${USQL_VERSION}" \
  $(pwd)
```

> [!NOTE]
> see: https://cloud.google.com/artifact-registry/docs/docker/authentication

### Using Cloud Build

Adjust environment variables as per your requirements:

```sh
gcloud builds submit --config $(pwd)/cloudbuild.yaml \
--substitutions "_REPO_LOCATION=${REPO_LOCATION},_REPO_NAME=${REPO_NAME},_IMAGE_NAME=${IMAGE_NAME},_IMAGE_TAG=${IMAGE_TAG},_BUILD_TAG=${BUILD_TAG},_WEB_PORT=${WEB_PORT},_SSH_USER=${SSH_USER},_SSH_PASS=${SSH_PASS},_PASSWORD_ACCESS=${PASSWORD_ACCESS},_SUDO_ACCESS=${SUDO_ACCESS},_CLOUDSDK_VERSION=${CLOUDSDK_VERSION},_GCSFUSE_VERSION=${GCSFUSE_VERSION},_CSQL_PROXY_VERSION=${CSQL_PROXY_VERSION},_ALLOYDB_PROXY_VERSION=${ALLOYDB_PROXY_VERSION},_USQL_VERSION=${USQL_VERSION},_DOCKERFILE=${DOCKERFILE}" \
$(pwd)
```

### Using pre-built images

```sh
docker pull ghcr.io/gchux/cloud-run-ssh:latest
docker tag ghcr.io/gchux/cloud-run-ssh:latest ${IMAGE_URI_FULL}
docker push ${IMAGE_URI_FULL}
```

> [!NOTE]
> Docker image tag `latest` points to `CONTENT_FLAVOR=lite` and `ACCESS_LEVEL=no-root`

## Deploying the image to Cloud Run

```sh
export SERVICE_NAME='...'
export SERVICE_REGION='...'

gcloud run deploy ${SERVICE_NAME} \
   --image=${IMAGE_URI_FULL} \
   --region=${SERVICE_REGION} \
   --port=8080 --execution-environment=gen2 \
   --min-instances=0 --max-instances=1 \
   --memory=2Gi --cpu=2 --cpu-boost \
   --timeout=3600s --no-use-http2 \
   --session-affinity --no-cpu-throttling \
   --set-env-vars="SUDO_ACCESS=${SUDO_ACCESS},PASSWORD_ACCESS=${PASSWORD_ACCESS},LOG_STDOUT=true" \
   --no-allow-unauthenticated
```

> [!CAUTION]
> it is strongly recommended to use **`--no-allow-unauthenticated`** in order to prevent unauthorized access to the SSH server.

> [!IMPORTANT]
> if `gen1` is needed, then both `ACCESS_LEVEL`, `SSH_USER`/`USER_NAME` must be set to `root`.

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

## Advanced Configurations

- Use `SUDO_ACCESS=true` if you want to allow users to escalate privileges without allowing `root` to login.

- At container execution time, it is possible to override the following parameters:

  - `SSH_PASS`: expose a secret using the environment variable `USER_PASSWORD`,

    - alternatively, mount a secret volume for a file containing the password at directory: `/wssh/secrets/2/`, and then:

      - define the environment variable `USER_PASSWORD_FILE` containing the exact file path; i/e: `/wssh/secrets/2/user_password`.

  - `SSH_USER`: use the environment variable `USER_NAME`; this environment variable may also be defined using a secret.

- When using public key authentication, you may use the following alternatives to provide Public keys:

  - Mount a secret volume for the Public key file that should have access at directory `/wssh/secrets/3/`, and then:

    - define the environment variable `PUBLIC_KEY_FILE` containing the exact file path; i/e: `/wssh/secrets/3/public_key`.

    > see: https://cloud.google.com/run/docs/configuring/services/secrets

  - Mount a GCS volume containing all Public keys that should have access, and then:

    - define its path using the environment variable `PUBLIC_KEY_DIR`.

    > see: https://cloud.google.com/run/docs/configuring/services/cloud-storage-volume-mounts

  - Serve an HTTP(S) accessible key file containing all the Public keys that should have access,

    - and define its full URL using the environment variable `PUBLIC_KEY_URL`.

  - Leverage the SSH server `AuthorizedKeysFile` config by mounting a secret volume at: `/wssh/secrets/1/authorized_keys`

> [!TIP]
> In order to avoid having to handle access management for all users, it is better/simpler to provide `user`, `password`, `Public key` and `authorized_keys` as secrets via environment variable or volume mounts.

---

# Cloud Run SSH server sidecar

As of version `2.0.0` the Cloud Run SSH Server can also be deployed as a sidecar whch will allow to troubleshoot Cloud Run instances running application code.

This operation mode does not require any modifications to the main –ingress– application container so that you can perform tests using its default configuration.

In order to SSH into the sidecar on a running instance, you'll need:

- The SSH Proxy Server: `ghcr.io/gchux/cloud-run-ssh:proxy-server-latest`
- and the SSH client: `ghcr.io/gchux/cloud-run-ssh:client-latest`
- Cloud Run service/revision with VPC connectivity: https://cloud.google.com/run/docs/configuring/connecting-vpc

This setup works in the following manner:

1. The Cloud Run SSH server sidecar creates TLS tunnel against the `SSH Proxy Server` via the `SSH Proxy Server API`.

   - The `SSH Proxy Server API` is served over `HTTPS` and requires the `SSH server sidecar` to provide an ID token using the [Cloud Run service identity](https://cloud.google.com/run/docs/securing/service-identity).
   - The `SSH Proxy Server API` enforces access controls on the project hosting the Cloud Run instances and the identity used by the service.

2. The `SSH Proxy Server` registers the Cloud Run instance(s) and enables access via the reserved TLS tunnel.

   - The `SSH Proxy Server` enforced access controls on the tunnel by restrict access to it only to specific hosts or networks; this may be the [`CIDR` ranges](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) assigned to VPC Connectors or Direct VPC.

3. The `SSH Proxy Server` may be hosted wherever a container can be executed, provided that it has access to the `Metadata Server`; i/e: Compute Engine VM, Kubernetes Engine cluster, etc...

   - The `SSH Proxy Server` identity is a user defined valid [`UUID`](https://en.wikipedia.org/wiki/Universally_unique_identifier) which is verified by both the `SSH server sidecar` and the `SSH Client` before registering and using the TLS tunnel.
   - The `SSH Proxy Server` API and Tunnel ports can be adjusted; the default values are: `5000` for the API, and `5555` for the Tunnel.

4. The [`SSH Client`](https://github.com/gchux/cloud-run-ssh/blob/main/visitor/ssh) ( aka the **SSH Proxy Visitor** ) uses the `SSH Proxy Server API` to resolve a Cloud Run instance id into the correct TLS tunnel to be used.
5. The [`SSH Client`](https://github.com/gchux/cloud-run-ssh/blob/main/visitor/ssh) uses the TLS tunnel to connect to a running **Cloud Run instance SSH sidecar** via the `SSH Proxy Server`.

![cloud_run_ssh_proxy](https://github.com/gchux/cloud-run-ssh/blob/main/pix/cloud_run_ssh_proxy.png?raw=true)

Since all networking happens within the VPC and tunnels are encrypted end to end, connecting into running instances is safe and secure.

## SSH sidecar configuration

A pre-built Docker image is available as: `ghcr.io/gchux/cloud-run-ssh:latest`.

The `SSH server sidecar` accepts the following environment variables:

- `ENABLE_SSH_PROXY`: _boolean_ ; default value is `false`.
- `SSH_PROXY_SERVER_HOST`: _IPv4_ ; no default value is assigned.
- `SSH_PROXY_SERVER_API_PORT` _uint16_ ; default value is `5000`.
- `SSH_PROXY_SERVER_TUNNEL_PORT`: _uint16_ ; default value is `5555`.
- `SSH_PROXY_SERVER_ID`: _uuid_ ; default value is `00000000-0000-0000-0000-000000000000`.

## SSH Proxy Server

A pre-built Docker image is available as: `ghcr.io/gchux/cloud-run-ssh:proxy-server-latest`.

### Configuration

The `SSH Proxy Server` requires a [`YAML` configuraiton file](https://github.com/gchux/cloud-run-ssh/blob/main/proxy/sample_config.yaml) which must be mounted at `/etc/ssh_proxy_server/config.yaml`.

- `id`: _uuid_ ; the `SSH Proxy Server` identity.
- `project_id`: _string_ ; the GCP Project ID hosting the `SSH Proxy Server`.
- `access_control.allowed_projects`: _list[string]_ ; Cloud Run allowed projects.
- `access_control.allowed_identities`: _list[email]_ ; identities allowed to consume the `SSH Proxy Server` API.
- `access_control.allowed_hosts`: _list[IPv4|IPv6|CIDR]_ ; hosts which are allowed to consume the `SSH Proxy Server` API and Tunnels.
  - **Must include**: VPC Connectors or Direct VPC [`CIDR` ranges](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing), and the allowed subnets where `SSH Clients` will be connecting from.

#### Sample Configuration

```yaml
id: 00000000-0000-0000-0000-000000000000
project_id: my-project-1
access_control:
  allowed_projects:
    - my-project-1
    - my-project-2
  allowed_identities:
    - ssh@my-project-1.iam.gserviceaccount.com
    - ssh@my-project-2.iam.gserviceaccount.com
  allowed_hosts:
    - 127.0.0.1
    - 169.254.0.0/16
```

### Execution

```sh
docker kill ssh-proxy-server

docker rm ssh-proxy-server

docker pull ghcr.io/gchux/cloud-run-ssh:proxy-server-latest

docker tag ghcr.io/gchux/cloud-run-ssh:proxy-server-latest cloud-run-ssh-proxy-server

docker run -d \
        --restart=unless-stopped \
        --name=cloud-run-ssh-proxy-server \
        -p 5555:${SSH_PROXY_SERVER_TUNNEL_PORT} \
        -p 5000:${SSH_PROXY_SERVER_API_PORT} \
        -p 127.0.0.1:8888:8888 \
        -e PROJECT_ID=${PROJECT_ID} \
        -v ./config.yaml:/etc/ssh_proxy_server/config.yaml:ro \
        cloud-run-ssh-proxy-server
```

> [!NOTE]
> Port `8888` exposes the `SSH Proxy Server` API (same as `${SSH_PROXY_SERVER_API_PORT}`), but is does NOT enforce access controls on `project` nor `identity`.

## SSH Client ( aka SSH Proxy Visitor )

A pre-built Docker image is available as: `ghcr.io/gchux/cloud-run-ssh:client-latest`

### Environment

The `SSH Client` container requies the following [environment variables](https://github.com/gchux/cloud-run-ssh/blob/main/visitor/sample_ssh.env):

- `PROJECT_ID`: _string_ ; GCP Project ID hosting the client.
- `SSH_PROXY_SERVER_HOST`: _IPv4_ ; `IP` assigned to the host running the `SSH Proxy Server`.
- `SSH_PROXY_SERVER_API_PORT` _uint16_ ; `SSH Proxy Server` API port.
- `SSH_PROXY_SERVER_TUNNEL_PORT`: _uint16_ ; `SSH Proxy Server` Tunnel port.
- `SSH_PROXY_SERVER_ID`: _uuid_ ; `SSH Proxy Server` identity.

#### Sample Environment

```sh
PROJECT_ID=my-project-1
SSH_PROXY_SERVER_HOST=${SSH_PROXY_SERVER_HOST}
SSH_PROXY_SERVER_API_PORT=5000
SSH_PROXY_SERVER_TUNNEL_PORT=5555
SSH_PROXY_SERVER_ID=00000000-0000-0000-0000-000000000000
```

### Execution

```sh
docker pull ghcr.io/gchux/cloud-run-ssh:client-latest

docker tag ghcr.io/gchux/cloud-run-ssh:client-latest cloud-run-ssh-client:latest

docker run -it --rm \
        --name=cloud-run-ssh-client \
        -e "INSTANCE_ID=${INSTANCE_ID}" \
        --env-file=ssh.env \
        cloud-run-ssh-client:latest
```

A [bash script that executed the `SSH Client` container](https://github.com/gchux/cloud-run-ssh/blob/main/visitor/ssh) is also available;

this script accepts onyl 1 argument: the Cloud Run `INSTANCE_ID` to connect; i/e: `./ssh ${INSTANCE_ID}`:

![cloud_run_ssh_client](https://github.com/gchux/cloud-run-ssh/blob/main/pix/cloud_run_ssh_client.png?raw=true)
