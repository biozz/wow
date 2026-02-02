## whitelists-research-v2

`uv`-embedded Python scanner for **content-sharing fingerprints** + optional **WebSocket reachability**.

### Run

- Scan + group by shared signatures (no WS):

```bash
chmod +x main.py
./main.py scan path/to/domains.txt --top 25 --show-domains 10
```

- Same thing, explicitly via uv:

```bash
uv run --script main.py scan path/to/domains.txt --top 25 --show-domains 10
```

- Enable WebSocket probes (wss):

```bash
./main.py scan path/to/domains.txt --ws --concurrency 50 --progress 25
```

- Write outputs:

```bash
./main.py scan path/to/domains.txt --jsonl out.jsonl --summary summary.md --quiet
```

### What it does (signals)

- **DNS**: follows CNAME chain (no A/AAAA collection)
- **HTTP**: requests `https://<domain>/` and fingerprints headers + final host
- **TLS**: grabs cert subject/issuer + ALPN
- **WS (optional)**: tries `wss://<domain>/` and a few common paths


