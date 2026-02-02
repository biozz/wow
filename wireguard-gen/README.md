# wireguard-gen

Generates WireGuard `*.wg0.conf` files from a single YAML config. One host = one config file; peers are derived from the `peers` list.

## Setup

1. Copy `config.example.yaml` to `config.yaml`.
2. Fill in each host: `address`, `private_key`, `public_key`, `peers`. Optionally set `endpoint` (host:port) for peers that others initiate to (e.g. server with static IP).

## Run

```bash
uv run wg_gen.py
```

Writes `<hostname>.wg0.conf` for each host in the config.

## Config

- **hosts** — map of hostname → { address, private_key, public_key, peers [, endpoint ] }.
- **address** — CIDR for this host (e.g. `10.0.0.1/32`).
- **peers** — list of other hostnames this host connects to.
- **endpoint** — optional; if set, peer blocks for this host will include `Endpoint` (for the host others connect to).

Do not commit `config.yaml` (private keys). It is ignored via `.gitignore`.
