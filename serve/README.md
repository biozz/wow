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

### 1. Configure

Configuration is read from (in order of precedence): CLI flags, environment variables, then a YAML config file. Default config file path is `serve.yaml` in the current directory. Override with `--config` / `-c` or the `SERVE_CONFIG` environment variable.

**Config file** — Create `serve.yaml` (or copy from `serve.example.yaml`):

```yaml
serve:
  etcd_endpoint: "localhost:2379"
  etcd_user: ""
  etcd_password: ""
  target_ip: "127.0.0.1"
  domain_template: "%s.example.com"
  cert_resolver: "lecf"
  key_prefix: "serve"
```

| YAML key (under `serve:`) | Description | Default |
|---------------------------|-------------|---------|
| `etcd_endpoint` | etcd server endpoint | `localhost:2379` |
| `etcd_user` | etcd username | (empty) |
| `etcd_password` | etcd password | (empty) |
| `target_ip` | Tailscale IP of your local machine (for Traefik to reach) | `127.0.0.1` |
| `domain_template` | Domain template; use `%s` for app name (required for `run`) | (empty) |
| `cert_resolver` | Traefik cert resolver name | `lecf` |
| `key_prefix` | Prefix for router/service names in etcd (e.g. `serve-myapp`) | `serve` |

**Override via env** — Same keys as before: `SERVE_ETCD_ENDPOINT`, `SERVE_ETCD_USER`, `SERVE_ETCD_PASSWORD`, `SERVE_ETCD_TARGET_IP`, `SERVE_DOMAIN_TEMPLATE`, `SERVE_CERT_RESOLVER`, `SERVE_KEY_PREFIX`. Env overrides the config file.

**Override via CLI** — Global flags: `--config` / `-c`, `--etcd-endpoint`, `--etcd-user`, `--etcd-password`, `--target-ip`, `--domain-template`, `--cert-resolver`, `--key-prefix`.

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

**Subdomains (wildcard TLS)** — If you need subdomains (e.g. `myapp.example.com` from `domain_template: "%s.example.com"`), configure a default TLS store with a wildcard cert in Traefik:

```yaml
tls:
  stores:
    default:
      defaultGeneratedCert:
        resolver: "acme"
        domain:
          main: "example.com"
          sans:
            - "*.example.com"
            - "*.app.example.com"
```

Use the same `resolver` name as in your `certificatesResolvers` and in serve's `cert_resolver` config.

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
- `--domain` (optional): A specific domain to use. If omitted, it uses `domain-template` (from config) with the app name.

This command will create entries in etcd under the `traefik/http/` prefix for routers and services (resource names use `{key_prefix}-{slug}` when the prefix is set).

### `stop`

Remove a service.

```bash
serve stop my-cool-app
# or stop by port
serve stop 8080
```

- `<slug|port>` (required): The name of the application or the port number to stop exposing.

This will delete the corresponding configuration from etcd, and Traefik will automatically stop routing traffic for it.

### `clean`

Remove all Traefik config for services managed by this utility (only those whose router/service name matches the key prefix).

```bash
serve clean
# or
serve reset
```

Use this to wipe all serve-managed entries from etcd in one go.

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

Services are stored in etcd with the following key structure. The resource name is `{prefix}-{slug}` when `key_prefix` is set (e.g. `serve-myapp`), or just `{slug}` when the prefix is empty.

```
traefik/http/routers/{prefix}-{slug}/entrypoints = "https"
traefik/http/routers/{prefix}-{slug}/tls = "true"
traefik/http/routers/{prefix}-{slug}/tls/certresolver = "{cert_resolver}"
traefik/http/routers/{prefix}-{slug}/rule = "Host(`{domain from domain_template}`)"
traefik/http/routers/{prefix}-{slug}/service = "{prefix}-{slug}"
traefik/http/services/{prefix}-{slug}/loadbalancer/servers/0/url = "http://{target_ip}:{port}"
``` 
