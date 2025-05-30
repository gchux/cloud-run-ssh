![Cloud Run SSH server version](https://img.shields.io/badge/v3.3.1-green?style=flat&label=latest%20version&labelColor=gray&color=green&link=https%3A%2F%2Fgithub.com%2Fgchux%2Fcloud-run-ssh%2Fpkgs%2Fcontainer%2Fcloud-run-ssh%2Fversions)

# Cloud Run SSH server

![cloud_run_ssh](https://github.com/gchux/cloud-run-ssh/blob/main/pix/cloud_run_ssh.png?raw=true)

## Motivation

During development and while troubleshooting issues, it is often very useful to test some preconditions in the same execution environment as the microservice while having more maneuverability.

The **`Cloud Run SSH server`** aims to provide full linux shell running in a Cloud Run service instance to allow developers and administrators to test and troubleshoot all kinds of scenarios.

## Features

- SSH server for Cloud Run gen1 and gen2.
- Get a text transcript of the SSH session.
- Download files from the SSH server filesystem.
- Execute commands from the UI using a catalog with 2 clicks.
- Automatically log into the SSH server using [Cloud Run proxy](https://cloud.google.com/sdk/gcloud/reference/run/services/proxy).
- Edit files and code using [Visual Studio Code](https://github.com/microsoft/vscode).
- Run [Docker](https://docs.docker.com/engine/) containers on top of Cloud Run gen2.

![cloud_run_ssh_ui](https://github.com/gchux/cloud-run-ssh/blob/main/pix/cloud_run_ssh_shell.png?raw=true)

![cloud_run_ssh_vscode](https://github.com/gchux/cloud-run-ssh/blob/main/pix/cloud_run_ssh_vscode.png?raw=true)

## Building Blocks:

- [docker-openssh-server](https://github.com/linuxserver/docker-openssh-server): provides the SSH server.
- [webssh](https://pypi.org/project/webssh/): provides web access to the SSH server and a browser based shell to execute commands.
- [Supervisor](https://supervisord.org/): provides process control and orchestration.
- [Nginx](https://nginx.org/): provides the HTTP proxy for `dev` and `app` flavors.
- [Docker](https://www.docker.com/): provides containers execution engine.
- [Code Server](https://github.com/coder/code-server): provides the web based Visual Studio Code editor.

## Image flavors

### By contents

- `lite`: contains the bare minimum tools to perform network troubleshooting.
- `full`: contains additional Google Cloud SDK and third-party tools.
- `dev`: contains Google Cloud SDK, [Docker](https://www.docker.com/) and [Visual Studio Code](https://code.visualstudio.com/).
- `app`: same as `lite`, includes [Docker](https://www.docker.com/) and allows to import a container image into the Cloud Run SSH server filesystem.

> [!TIP]
> Choose `lite` if all you need to do is network troubleshooting; build-time and size will be reduced.

> [!NOTE]
> The `app` and `dev` flavors are only supported under Cloud Run gen2 environment.

### By access level

- `root`: whether to allow `sudo` to escalate privileges.
- `no-root`: `sudo` is disabled; the logged in user won't be able to escalate privileges.

> [!TIP]
> choose `no-root` if you are providing this image to others so that installing new software is disabled.

## Using pre-built images

```sh
docker pull ghcr.io/gchux/cloud-run-ssh:latest
docker tag ghcr.io/gchux/cloud-run-ssh:latest ${IMAGE_URI_FULL}
docker push ${IMAGE_URI_FULL}
```

Choose from one of the following pre-built image flavors:

- ghcr.io/gchux/cloud-run-ssh:lite-root
- ghcr.io/gchux/cloud-run-ssh:full-root
- ghcr.io/gchux/cloud-run-ssh:dev-root

> [!NOTE]
> Docker image tag `latest` points to `CONTENT_FLAVOR=lite` and `ACCESS_LEVEL=root`

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

export WEB_PORT=8022               # whatever TCP port you want to use to serve the WEB SSH terminal
export DEV_PORT=8088               # whatever TCP port you want to use to serve Visual Studio code
export APP_PORT=8080               # whatever TCP port you want to use to serve the APP under test

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
# pass the full path to your env file to override `.env`
./docker_build
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

## Deploying the image to Cloud Run

```sh
export SERVICE_NAME='...'
export SERVICE_REGION='...'

export PASSWORD_ACCESS='true|false' # allow users to login using user and password.
export SUDO_ACCESS='true|false'     # allow users to use `sudo` and login as `root`.
export SSH_AUTO_LOGIN='true|false'  # allow going straight to shell; requires `PASSWORD_ACCESS`.

# set PORT to `8888` for `dev` and `app` server flavors.
export CLOUD_RUN_PORT="${WEB_PORT}"

gcloud run deploy ${SERVICE_NAME} \
   --image=${IMAGE_URI_FULL} \
   --region=${SERVICE_REGION} \
   --port=${CLOUD_RUN_PORT} \
   --execution-environment=gen2 \
   --min-instances=0 --max-instances=1 \
   --memory=1Gi --cpu=1 --cpu-boost \
   --timeout=3600s --no-use-http2 \
   --session-affinity --no-cpu-throttling \
   --set-env-vars="SUDO_ACCESS=${SUDO_ACCESS},PASSWORD_ACCESS=${PASSWORD_ACCESS},SSH_AUTO_LOGIN=${SSH_AUTO_LOGIN},LOG_STDOUT=true" \
   --no-allow-unauthenticated
```

> [!CAUTION]
> It is **strongly recommended to use `--no-allow-unauthenticated`** in order to **prevent unauthorized access to the SSH server**.

> [!IMPORTANT]
> If `gen1` is needed, then both `ACCESS_LEVEL`, `SSH_USER`/`USER_NAME` must be set to `root`.

## SSHing into the container

1. `gcloud run services proxy ${SERVICE_NAME} --region=${SERVICE_REGION} --port=8080`

   > see: https://cloud.google.com/sdk/gcloud/reference/run/services/proxy

2. Use a WEB browser, got to: `http://127.0.0.1:8080/`

3. If `SSH_AUTO_LOGIN` is set to `false`, then fill in the following fields:

   - Hostname: `127.0.0.1` _(fixed)_
   - Port: `2222` _(fixed)_
   - Username: `${SSH_USER}`
   - Password: `${SSH_PASS}`

4. Click `Connect`

> [!TIP]
> The `gcloud` proxy command and ALL relevant URLs will be avaiable in Cloud Logging; filter logs using `[SSH Server]` to find them.
> Use any of the logged URLs to log into the **`SSH Server`** automatically and go straight to the shell.

## Advanced Configurations

- Use `PASSWORD_ACCESS=true` if you want to allow users to login with user and password.

- Use `SUDO_ACCESS=true` if you want to allow users to escalate privileges without allowing `root` to login.

- Use `SSH_AUTO_LOGIN=true` if you want to allow users to be automatically logged in and go straight to the `SHELL`.

> [!IMPORTANT]
> The option `SSH_AUTO_LOGIN=true` requires `PASSWORD_ACCESS=true`, and `SUDO_ACCESS=true` if the intended user is `root`.

- At container execution time, it is possible to override the following parameters:

  - `SSH_PASS`: expose a secret using the environment variable `USER_PASSWORD`,

    - alternatively, mount a secret volume for a file containing the password at directory: `/wssh/secrets/2/`, and then:

      - define the environment variable `USER_PASSWORD_FILE` containing the exact file path; i/e: `/wssh/secrets/2/user_password`.

  - `SSH_USER`: use the environment variable `USER_NAME`:

    - this environment variable may also be defined using a secret.

    - it must not be set if the expected user is `root`.

  - `WEB_PORT`: use the environment variable `WEBSSH_PORT`.

    - when using the `lite` flavor, it will be overriden by the `PORT` environment variable.

  - `DEV_PORT`: use the environment variable `WEBDEV_PORT`.

  - `APP_PORT`: use the environment variable `WEBAPP_PORT`.

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

As of version `v2.0.0` the **`Cloud Run SSH Server`** can also be deployed as a sidecar which allows to **individually access and troubleshoot Cloud Run instances** running application code.

This operation mode does not require any modifications to the main –ingress– application container so that you can perform the required troubleshooting using its default configuration; this effectively means that any other containers remain immutable.

In order to SSH into the sidecar in any **Cloud Run running instance**, you'll need:

- The SSH Proxy Server: `ghcr.io/gchux/cloud-run-ssh:proxy-server-latest`
- and the SSH client: `ghcr.io/gchux/cloud-run-ssh:client-latest`
- Cloud Run service/revision with VPC connectivity:
  - https://cloud.google.com/run/docs/configuring/connecting-vpc

This setup works by creating [bastion host](https://en.wikipedia.org/wiki/Bastion_host) ( the `SSH Proxy Server` ) through which **Cloud Run instances** are **individually accessible** but not directly reachable as the `SSH Proxy Server` cannot route traffic to any of them unless the **Cloud Run instances** establish a connection first; this connections is called a tunnel.

The flow to create and use an encrypted tunnel to connect to a **Cloud Run instance** is decribed as follows:

1. Upon startup, the `Cloud Run SSH server sidecar` creates a TLS tunnel via the `SSH Proxy Server` using the **`SSH Proxy Server` API**.

   - The TLS tunnel created by a **Cloud Run instance** is identified by a randomly generated [`UUID`](https://en.wikipedia.org/wiki/Universally_unique_identifier). This TLS tunnel ID is the **Cloud Run instance**'s identity.
   - The **`SSH Proxy Server` API** is served over `HTTPS` and requires the `SSH server sidecar` and `SSH Client` to provide a verifiable ID token.
   - The `Cloud Run SSH server sidecar` uses the [Cloud Run service identity](https://cloud.google.com/run/docs/securing/service-identity) to generate tokens.
   - The **`SSH Proxy Server` API** enforces access controls on the project hosting **Cloud Run instances** and the **identity used by the Cloud Run service**.
   - In order to be reachable, **Cloud Run instances** must register themselves with the `SSH Proxy Server`; otherwise, `SSH Clients` cannot connect.

2. The `SSH Proxy Server` registers the available **Cloud Run instance(s)**, and enables access via the reserved TLS tunnels.

   - The `SSH Proxy Server` enforces access controls on the TLS tunnel by restricting access to specific hosts or networks.
     - Allowed hosts must include the [`CIDR` ranges](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing) assigned to VPC Connectors or Direct VPC.

3. The `SSH Proxy Server` may be hosted wherever a container can be executed, provided that it has access to the `Metadata Server`; i/e: Compute Engine VM, Kubernetes Engine cluster, etc...

   - The **`SSH Proxy Server` identity** is a user defined and valid [`UUID`](https://en.wikipedia.org/wiki/Universally_unique_identifier) which is verified by both the `SSH server sidecar` and the `SSH Client` before registering and using the TLS tunnels.
   - The **`SSH Proxy Server` API & Tunnel** ports can be adjusted; the default values are: `5000` for the API, and `5555` for the Tunnel.

4. The [`SSH Client`](https://github.com/gchux/cloud-run-ssh/blob/main/visitor/ssh) ( aka the **SSH Proxy Visitor** ) uses the **`SSH Proxy Server` API** to resolve a **Cloud Run instance** ID into the correct TLS tunnel to be used.

5. The [`SSH Client`](https://github.com/gchux/cloud-run-ssh/blob/main/visitor/ssh) uses the discovered TLS tunnel to connect to a running **Cloud Run instance**'s `SSH server sidecar` via the `SSH Proxy Server`.

   - The `SSH Proxy Server` must also allow access to the host(s) `IPv4`/`IPv6` or `CIDR` ranges where the `SSH Clients` will be connecting from.

6. `Cloud Run SSH server sidear` pings the `SSH Proxy Server` API every 60 seconds to renew its registration.

   - Similarly, the `SSH Proxy Server` drops registrations if a **Cloud Run instance** has not renewed its registration within 15 minutes.

![cloud_run_ssh_proxy](https://github.com/gchux/cloud-run-ssh/blob/main/pix/cloud_run_ssh_proxy.png?raw=true)

Since all networking happens within the VPC and tunnels are encrypted end to end, connecting into running instances is safe and secure.

> [!IMPORTANT]
> While the `SSH Proxy Server` identity is known, the `Cloud Run SSH server sidecar` identity **must remain confidential** and only discovered via the **`SSH Proxy Server` API**.

## SSH server sidecar

A pre-built Docker image is available as: `ghcr.io/gchux/cloud-run-ssh:latest`.

In addition to the environment variables used when running as the main –ingress– application container, the `SSH server sidecar` accepts the following:

- `ENABLE_SSH_PROXY`: _boolean_ ; default value is `false`.
- `SSH_PROXY_SERVER_HOST`: _IPv4_ ; the `IPv4` of the host runnignt the `SSH Proxy Server`. No default value is assigned.
- `SSH_PROXY_SERVER_API_PORT` _uint16_ ; default value is `5000`.
- `SSH_PROXY_SERVER_TUNNEL_PORT`: _uint16_ ; default value is `5555`.
- `SSH_PROXY_SERVER_ID`: _uuid_ ; the `SSH Proxy Server` identity that will be verified by the **Cloud Run instance** before enabling a tunnel. Default value is `00000000-0000-0000-0000-000000000000`.

## SSH Proxy Server

A pre-built Docker image is available as: `ghcr.io/gchux/cloud-run-ssh:proxy-server-latest`.

### Configuration

The `SSH Proxy Server` requires a [`YAML` configuraiton file](https://github.com/gchux/cloud-run-ssh/blob/main/proxy/sample_config.yaml) which must be mounted at `/etc/ssh_proxy_server/config.yaml`.

- `id`: _uuid_ (**required**); the `SSH Proxy Server` identity.
- `project_id`: _string_ (**required**); the GCP Project ID hosting the `SSH Proxy Server`.
- `access_control.allowed_projects`: _list[string]_ (optional); **Cloud Run** projects allowed to register instances and accept connections. If empty, it allows all projects.
- `access_control.allowed_regions`: _list[string]_ (optionsl); **Cloud Run** regions allowed to register instances and accept connections. If empty, it allows all regions.
- `access_control.allowed_services`: _list[string]_ (optional); **Cloud Run** services allowed to register instances and accept connections. If empty, it allows all services.
- `access_control.allowed_revisions`: _list[string]_ (optional); **Cloud Run** revisions allowed to register instances and accept connections. If empty, it allows all revisions.
- `access_control.allowed_identities`: _list[email]_ ; identities allowed to consume the `SSH Proxy Server` API. Identities are enforced for the **Cloud Run instances** and the `SSH Clients`.
- `access_control.allowed_hosts`: _list[IPv4|IPv6|CIDR]_ (**required**); hosts which are allowed to consume the `SSH Proxy Server` API and Tunnels.
  - **Must include**: VPC Connectors or Direct VPC [`CIDR` ranges](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing), and the allowed subnets where `SSH Clients` will be connecting from.

#### Sample Configuration

```yaml
id: 00000000-0000-0000-0000-000000000000
project_id: my-project-1
access_control:
  allowed_projects:
    - my-project-1
    - my-project-2
  allowed_regions:
    - us-central1
    - us-west4
  allowed_services:
    - my-service-1
    - my-service-2
  allowed_revisions:
    - my-service-1-0001-abcd
    - my-service-2-0001-abcd
  allowed_identities:
    - ssh@my-project-1.iam.gserviceaccount.com
    - ssh@my-project-2.iam.gserviceaccount.com
  allowed_hosts:
    - ::1
    - 127.0.0.1
    - 169.254.0.0/16
```

### Execution

```sh
docker kill cloud-run-ssh-proxy-server

docker rm cloud-run-ssh-proxy-server

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
> Port `8888` exposes the `SSH Proxy Server` API ( same as `${SSH_PROXY_SERVER_API_PORT}` ),
> but it does NOT enforce `access_control` other than `allowed_hosts`, nor is it served over `HTTPS`.
> It is **strongly recommended to no expose port `8888` outisde loopback ( `localhost` ) interface**.

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
        -e "INSTANCE_ID=${1}" \
        --env-file=ssh.env \
        --network=host \
        cloud-run-ssh-client:latest \
        "${@:1}"
```

A [bash script that executes the `SSH Client` container](https://github.com/gchux/cloud-run-ssh/blob/main/visitor/ssh) is also available;
this script requires the 1st argument to be the **Cloud Run `INSTANCE_ID`** to connect; i/e: `./ssh ${INSTANCE_ID}`.

After the **Cloud Run instance ID** you may use any valid [`OpenSSH client` arguments](https://man.openbsd.org/ssh),
except for [`-p` or **port**](https://man.openbsd.org/ssh#p) as it is reserved for the TLS tunnel ( `TCP::2222` ) to deliver all traffic to the **Cloud Run instance**.

In general, the most useful flag is local port forward or [`-L`](https://man.openbsd.org/ssh#L) which you may use to forward traffic
from a local TCP port into a remote TCP port available in the **Cloud Run instance** itself or any other remote host reachable/routable from the **Cloud Run instance**.

#### Sample `SSH Client` Execution

- Simple SSH connection into a running **Cloud Run instance**:

  ![cloud_run_ssh_client](https://github.com/gchux/cloud-run-ssh/blob/main/pix/cloud_run_ssh_client.png?raw=true)

- SSH connection into a running **Cloud Run instance** with local port forwarding:

  ![cloud_run_ssh_client_port_forward](https://github.com/gchux/cloud-run-ssh/blob/main/pix/cloud_run_ssh_client_2.png?raw=true)
