# Serve Utility

An attempt to re-invent ngrok. There are lots of options out there https://github.com/anderspitman/awesome-tunneling, and this is just my twist on the already solved problem.

Serve is a CLI utility to expose local development services running on a Tailscale network to the internet via a remote server running Traefik with etcd backend.

## Prerequisites

- A remote server with Traefik v2+ installed and configured with etcd backend.
- etcd server accessible from both your local machine and the remote Traefik server.
- Tailscale installed and configured on both the remote server and your local development machine.
- A domain name you can manage DNS for.

## How It Works

The `serve` utility stores Traefik configuration directly in etcd that creates routers and services. These services point to your development machine's Tailscale IP address. The remote Traefik instance uses its etcd Provider to watch for these configuration changes and update its routing table automatically.

## Setup

### 1. Configure Environment Variables

Set the following environment variables. You can add them to your `~/.bashrc` or `~/.zshrc` file.

-   `SERVE_ETCD_ENDPOINT`: (Optional) The etcd server endpoint. Defaults to `localhost:2379`.
    -   Example: `export SERVE_ETCD_ENDPOINT=etcd.example.com:2379`
-   `SERVE_ETCD_USER`: (Optional) Username for etcd authentication.
    -   Example: `export SERVE_ETCD_USER=myuser`
-   `SERVE_ETCD_PASSWORD`: (Optional) Password for etcd authentication.
    -   Example: `export SERVE_ETCD_PASSWORD=mypassword`
-   `SERVE_ETCD_TARGET_IP`: (Optional) The Tailscale IP address of your local development machine. Defaults to `127.0.0.1`.
    -   Example: `export SERVE_ETCD_TARGET_IP=100.1.2.3`
-   `SERVE_DOMAIN_TEMPLATE`: (Required) Template string for generating domains. Use `%s` as placeholder for the app name.
    -   Example: `export SERVE_DOMAIN_TEMPLATE=%s.example.com`
-   `SERVE_CERT_RESOLVER`: (Optional) The cert resolver to use for TLS certificates. Defaults to `lecf`.
    -   Example: `export SERVE_CERT_RESOLVER=letsencrypt`

### 2. Configure Traefik

Ensure your main `traefik.yml` (or `traefik.toml`) on the remote server has the etcd Provider enabled and is pointing to your etcd server.

**Example `traefik.yml`:**

```yaml
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

providers:
  etcd:
    endpoints:
      - "etcd.example.com:2379"
    username: "myuser"
    password: "mypassword"

certificatesResolvers:
  # your config

api:
  dashboard: true
```


## Usage

### `run`

Add a new service to be exposed.

```bash
serve run 8080
# Generated app name: x7k2m9qp
# Successfully configured x7k2m9qp.example.com to point to 100.1.2.3:8080

# or with custom app name
serve run 8080 --slug my-cool-app
```

- `<port>` (required): The port your local application is running on (e.g., `3000`, `8080`, `:8080`).
- `--slug` (optional): A unique name for your application. If not provided, an 8-character random alphanumeric slug will be generated.
- `--domain` (optional): A specific domain to use. If omitted, it uses `SERVE_DOMAIN_TEMPLATE` with the app name.

This command will create entries in etcd under the `traefik/http/` prefix for routers and services.

### `stop`

Remove a service.

```bash
serve stop my-cool-app
# or stop by port
serve stop 8080
```

- `<slug|port>` (required): The name of the application or the port number to stop exposing.

This will delete the corresponding configuration from etcd, and Traefik will automatically stop routing traffic for it.

### `status`

List all currently active services managed by this utility.

```bash
serve status
```

**Example Output:**

```
SLUG                 DOMAIN                                   PORT
-------------------- ---------------------------------------- ----
my-cool-app          https://my-cool-app.example.com          :8080
another-app          https://another-app.example.com          :3000
```

## etcd Key Structure

Services are stored in etcd with the following key structure:

```
traefik/http/routers/{slug}/entrypoints = "https"
traefik/http/routers/{slug}/tls = "true"
traefik/http/routers/{slug}/tls/certresolver = "{SERVE_CERT_RESOLVER}"
traefik/http/routers/{slug}/rule = "Host(`{domain from SERVE_DOMAIN_TEMPLATE}`)"
traefik/http/routers/{slug}/service = "{slug}"
traefik/http/services/{slug}/loadbalancer/servers/0/url = "http://{SERVE_ETCD_TARGET_IP}:{port}"
``` 