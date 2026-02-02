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
  etcd_root_key: "traefik"
  target_ip: "127.0.0.1"
  domain_template: "%s.example.com"
  cert_resolver: "lecf"
  key_prefix: "serve"
  slug_length: 3
```

| YAML key (under `serve:`) | Description | Default |
|---------------------------|-------------|---------|
| `etcd_endpoint` | etcd server endpoint | `localhost:2379` |
| `etcd_user` | etcd username | (empty) |
| `etcd_password` | etcd password | (empty) |
| `etcd_root_key` | etcd key prefix for Traefik (e.g. `traefik-vortex`, `traefik-andromeda`) | `traefik` |
| `target_ip` | Tailscale IP of your local machine (for Traefik to reach) | `127.0.0.1` |
| `domain_template` | Domain template; use `%s` for app name (required for `run`) | (empty) |
| `cert_resolver` | Traefik cert resolver name | `lecf` |
| `key_prefix` | Prefix for router/service names in etcd (e.g. `serve-myapp`) | `serve` |
| `slug_length` | Length of auto-generated slug when `--slug` is not provided | `3` |

**Override via env** — `SERVE_ETCD_ENDPOINT`, `SERVE_ETCD_USER`, `SERVE_ETCD_PASSWORD`, `SERVE_ETCD_ROOT_KEY`, `SERVE_ETCD_TARGET_IP`, `SERVE_DOMAIN_TEMPLATE`, `SERVE_CERT_RESOLVER`, `SERVE_KEY_PREFIX`, `SERVE_SLUG_LENGTH`. Env overrides the config file.

**Override via CLI** — Global flags: `--config` / `-c`, `--etcd-endpoint`, `--etcd-user`, `--etcd-password`, `--etcd-root-key`, `--target-ip`, `--domain-template`, `--cert-resolver`, `--key-prefix`, `--slug-length`, `--slug`. Slug can be set globally (e.g. `serve --slug myapp run 8080`) or per-command.

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

### `run` (alias: `start`)

Add a new service to be exposed. Blocks until Ctrl+C unless `--detach` is used.

```bash
serve run 8080
# Generated app name: x7k
# Serving at https://x7k.example.com (forwarding to :8080). Press Ctrl+C to stop and remove from Traefik.

# with custom app name (flag can be global or on run)
serve run 8080 --slug my-cool-app
# or: serve --slug my-cool-app run 8080

# run in background (no cleanup on exit)
serve run 8080 --slug myapp --detach
```

- `<port>` (required): The port your local application is running on (e.g. `3000`, `8080`, `:8080`).
- `--slug` (optional): Name for your application. If not provided, a random alphanumeric slug of length `slug_length` (default 3) is generated.
- `--detach` / `-d` (optional): Don't block; leave config in etcd when the process exits (no cleanup on Ctrl+C).

This command will create entries in etcd under `{etcd_root_key}/http/` for routers and services (resource names use `{key_prefix}-{slug}` when the prefix is set).

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

### `status` (aliases: `ls`, `list`)

List all currently active services managed by this utility.

```bash
serve status
# or: serve ls   /   serve list
```

**Example Output:**

```
SLUG                 DOMAIN                                   PORT
-------------------- ---------------------------------------- ----
my-cool-app          https://my-cool-app.example.com          :8080
another-app          https://another-app.example.com          :3000
```

## etcd Key Structure

Services are stored in etcd with the following key structure. The root is `etcd_root_key` (default `traefik`). The resource name is `{key_prefix}-{slug}` when `key_prefix` is set (e.g. `serve-myapp`), or just `{slug}` when the prefix is empty.

```
{etcd_root_key}/http/routers/{res_name}/entrypoints = "https"
{etcd_root_key}/http/routers/{res_name}/tls = "true"
{etcd_root_key}/http/routers/{res_name}/tls/certresolver = "{cert_resolver}"
{etcd_root_key}/http/routers/{res_name}/rule = "Host(`{domain from domain_template}`)"
{etcd_root_key}/http/routers/{res_name}/service = "{res_name}"
{etcd_root_key}/http/services/{res_name}/loadbalancer/servers/0/url = "http://{target_ip}:{port}"
``` 
