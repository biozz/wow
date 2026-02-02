# /// script
# requires-python = ">=3.10"
# dependencies = ["pyyaml"]
# ///

from pathlib import Path

import yaml

CONFIG_PATH = Path(__file__).with_name("config.yaml")

# hostname -> { address, private_key, public_key, peers, endpoint? }
# endpoint: if set, peer blocks for this host will include Endpoint (for hosts that others initiate to)
def load_config() -> dict[str, dict]:
    with open(CONFIG_PATH, "r") as f:
        data = yaml.safe_load(f) or {}
    hosts = data.get("hosts") or {}
    if not isinstance(hosts, dict):
        raise ValueError("config.yaml: hosts must be a mapping")
    return hosts


HOSTS = load_config()
HOSTNAMES = list(HOSTS.keys())


def allowed_ips_for_peer(_hostname: str, peer: str) -> list[str]:
    """AllowedIPs: just the peer's /32."""
    return [HOSTS[peer]["address"]]


def listen_port_from_endpoint(endpoint: str | None) -> str | None:
    if not endpoint:
        return None
    if ":" not in endpoint:
        return None
    return endpoint.rsplit(":", 1)[1]


def render_interface_block(hostname: str) -> str:
    h = HOSTS[hostname]
    lines = [
        "[Interface] # " + hostname,
        "PrivateKey = " + h["private_key"],
        "Address = " + h["address"],
    ]
    if (listen_port := listen_port_from_endpoint(h.get("endpoint"))):
        lines.append("ListenPort = " + listen_port)
    return "\n".join(lines)


def render_peer_block(hostname: str, peer: str) -> str:
    allowed = ", ".join(allowed_ips_for_peer(hostname, peer))
    lines = [
        "[Peer] # " + peer,
        "PublicKey = " + HOSTS[peer]["public_key"],
        "AllowedIPs = " + allowed,
    ]
    if (endpoint := HOSTS[peer].get("endpoint")):
        lines.append("Endpoint = " + endpoint)
    lines.append("PersistentKeepalive = 25")
    return "\n".join(lines)


def render_host_config(hostname: str) -> str:
    parts = [render_interface_block(hostname)]
    for peer in HOSTS[hostname]["peers"]:
        parts.append(render_peer_block(hostname, peer))
    return "\n\n".join(parts)


def validate():
    for h in HOSTNAMES:
        seen: set[str] = set()
        for peer in HOSTS[h]["peers"]:
            assert peer in HOSTS, f"unknown peer {peer} on {h}"
            for cidr in allowed_ips_for_peer(h, peer):
                assert cidr not in seen, f"duplicate AllowedIPs {cidr} on {h}"
                seen.add(cidr)


def main():
    validate()
    for hostname in HOSTNAMES:
        path = hostname + ".wg0.conf"
        with open(path, "w") as f:
            f.write(render_host_config(hostname) + "\n")


if __name__ == "__main__":
    main()
