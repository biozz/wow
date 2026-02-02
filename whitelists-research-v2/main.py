#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "aiohttp>=3.9.5",
#   "dnspython>=2.6.1",
# ]
# ///

from __future__ import annotations

import argparse
import asyncio
import hashlib
import json
import socket
import ssl
import sys
from dataclasses import asdict, dataclass, field
from typing import Any, Iterable
from urllib.parse import urlparse

import aiohttp
import dns.exception
import dns.resolver


def _norm_domain(line: str) -> str | None:
    s = line.strip()
    if not s or s.startswith("#"):
        return None
    # Allow scheme + paths in input; normalize to hostname.
    if "://" in s:
        p = urlparse(s)
        host = p.hostname
        if not host:
            return None
        return host.strip(".").lower()
    # Strip paths if present.
    s = s.split("/", 1)[0]
    # Strip port if present.
    if ":" in s and s.count(":") == 1:
        s = s.split(":", 1)[0]
    return s.strip(".").lower() or None


def read_domains(path: str) -> list[str]:
    domains: list[str] = []
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            d = _norm_domain(line)
            if d:
                domains.append(d)
    return domains


def _safe_get_header(headers: aiohttp.typedefs.LooseHeaders, key: str) -> str:
    try:
        v = headers.get(key)  # type: ignore[attr-defined]
    except Exception:
        v = None
    if v is None:
        return ""
    return str(v)


def _sha256_hex(s: str) -> str:
    return hashlib.sha256(s.encode("utf-8", errors="replace")).hexdigest()


@dataclass(slots=True)
class HTTPFingerprint:
    ok: bool = False
    status: int | None = None
    final_url: str = ""
    final_host: str = ""
    content_type: str = ""
    server: str = ""
    via: str = ""
    cache: str = ""
    cf_ray: str = ""
    cf_cache_status: str = ""
    x_cache: str = ""
    x_served_by: str = ""
    x_amz_cf_pop: str = ""
    x_amz_cf_id: str = ""
    x_vercel_id: str = ""
    x_nf_request_id: str = ""
    error: str = ""


@dataclass(slots=True)
class TLSFingerprint:
    ok: bool = False
    alpn: str = ""
    sni: str = ""
    peer_ip: str = ""
    subject_cn: str = ""
    issuer_cn: str = ""
    san: list[str] = field(default_factory=list)
    not_before: str = ""
    not_after: str = ""
    error: str = ""


@dataclass(slots=True)
class WebSocketProbe:
    url: str
    ok: bool = False
    negotiated_subprotocol: str = ""
    error: str = ""
    ms: float = 0.0


@dataclass(slots=True)
class DomainScanResult:
    domain: str
    cname_chain: list[str] = field(default_factory=list)
    http: HTTPFingerprint = field(default_factory=HTTPFingerprint)
    tls: TLSFingerprint = field(default_factory=TLSFingerprint)
    websockets: list[WebSocketProbe] = field(default_factory=list)
    signature: str = ""
    errors: list[str] = field(default_factory=list)


async def resolve_cname_chain(domain: str, timeout_s: float) -> list[str]:
    # dnspython is sync; run it in a thread to keep the scanner async.
    def _do() -> list[str]:
        r = dns.resolver.Resolver()
        r.timeout = timeout_s
        r.lifetime = timeout_s

        chain: list[str] = []
        cur = domain.rstrip(".")
        seen: set[str] = set()

        for _ in range(10):  # prevent loops
            if cur in seen:
                break
            seen.add(cur)
            try:
                ans = r.resolve(cur, "CNAME")
            except (dns.resolver.NoAnswer, dns.resolver.NXDOMAIN, dns.resolver.NoNameservers, dns.exception.Timeout):
                break
            except Exception:
                break

            try:
                target = str(ans[0].target).rstrip(".")
            except Exception:
                break
            chain.append(target)
            cur = target

        return chain

    return await asyncio.to_thread(_do)


async def fetch_http_fingerprint(
    session: aiohttp.ClientSession, domain: str, timeout_s: float
) -> HTTPFingerprint:
    fp = HTTPFingerprint()

    # Prefer https:// (most infra signals are there). We also allow redirects.
    url = f"https://{domain}/"
    try:
        timeout = aiohttp.ClientTimeout(total=timeout_s)
        async with session.get(url, allow_redirects=True, timeout=timeout) as resp:
            fp.ok = True
            fp.status = resp.status
            fp.final_url = str(resp.url)
            fp.final_host = (resp.url.host or "").lower()
            fp.content_type = resp.headers.get("content-type", "")

            # Keep a curated set of headers for fingerprinting.
            fp.server = resp.headers.get("server", "")
            fp.via = resp.headers.get("via", "")
            fp.cache = resp.headers.get("cache-control", "")
            fp.cf_ray = resp.headers.get("cf-ray", "")
            fp.cf_cache_status = resp.headers.get("cf-cache-status", "")
            fp.x_cache = resp.headers.get("x-cache", "")
            fp.x_served_by = resp.headers.get("x-served-by", "")
            fp.x_amz_cf_pop = resp.headers.get("x-amz-cf-pop", "")
            fp.x_amz_cf_id = resp.headers.get("x-amz-cf-id", "")
            fp.x_vercel_id = resp.headers.get("x-vercel-id", "")
            fp.x_nf_request_id = resp.headers.get("x-nf-request-id", "")

            # Drain a small amount so some servers actually send headers fully.
            try:
                await resp.content.read(256)
            except Exception:
                pass
    except Exception as e:
        fp.error = str(e)

    return fp


async def fetch_tls_fingerprint(domain: str, timeout_s: float) -> TLSFingerprint:
    fp = TLSFingerprint(sni=domain)

    # We do a small TLS handshake ourselves to retrieve certificate metadata.
    # NOTE: We intentionally do not validate the certificate chain here; we're fingerprinting.
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    ctx.set_alpn_protocols(["h2", "http/1.1"])

    loop = asyncio.get_running_loop()

    def _get_cert_cn(name: str, cert: dict[str, Any]) -> str:
        try:
            subj = cert.get(name, ())
            for rdn in subj:
                for k, v in rdn:
                    if k.lower() == "commonname":
                        return str(v)
        except Exception:
            pass
        return ""

    try:
        infos = await loop.getaddrinfo(domain, 443, type=socket.SOCK_STREAM)
        if not infos:
            fp.error = "getaddrinfo returned no results"
            return fp
        # pick first
        ip = infos[0][4][0]
        fp.peer_ip = ip

        raw_sock = socket.socket(socket.AF_INET6 if ":" in ip else socket.AF_INET, socket.SOCK_STREAM)
        raw_sock.settimeout(timeout_s)
        raw_sock.connect((ip, 443))

        tls_sock = ctx.wrap_socket(raw_sock, server_hostname=domain)
        fp.ok = True
        fp.alpn = tls_sock.selected_alpn_protocol() or ""

        cert = tls_sock.getpeercert()
        fp.subject_cn = _get_cert_cn("subject", cert)
        fp.issuer_cn = _get_cert_cn("issuer", cert)
        fp.not_before = str(cert.get("notBefore", ""))
        fp.not_after = str(cert.get("notAfter", ""))
        try:
            san = cert.get("subjectAltName", ())
            fp.san = [str(v) for (k, v) in san if str(k).lower() == "dns"]
        except Exception:
            fp.san = []
        tls_sock.close()
    except Exception as e:
        fp.error = str(e)

    return fp


async def probe_websockets(
    session: aiohttp.ClientSession,
    domain: str,
    paths: list[str],
    timeout_s: float,
) -> list[WebSocketProbe]:
    out: list[WebSocketProbe] = []
    timeout = aiohttp.ClientTimeout(total=timeout_s)

    for p in paths:
        if not p.startswith("/"):
            p = "/" + p
        url = f"wss://{domain}{p}"
        probe = WebSocketProbe(url=url)
        start = asyncio.get_running_loop().time()
        try:
            async with session.ws_connect(
                url,
                timeout=timeout,
                ssl=False,  # keep consistent w/ TLS fingerprinting (no validation)
                autoping=True,
                autoclose=True,
                max_msg_size=1024 * 1024,
            ) as ws:
                probe.ok = True
                probe.negotiated_subprotocol = ws.protocol or ""
        except Exception as e:
            probe.error = str(e)
        finally:
            probe.ms = (asyncio.get_running_loop().time() - start) * 1000.0
            out.append(probe)
    return out


def compute_signature(domain: str, cname_chain: list[str], http: HTTPFingerprint, tls: TLSFingerprint) -> str:
    parts = [
        "cname=" + (cname_chain[-1].lower() if cname_chain else ""),
        "final_host=" + (http.final_host or ""),
        "status=" + (str(http.status) if http.status is not None else ""),
        "content_type=" + (http.content_type or "").lower(),
        "server=" + (http.server or "").lower(),
        "via=" + (http.via or "").lower(),
        "cache_control=" + (http.cache or "").lower(),
        "x_cache=" + (http.x_cache or "").lower(),
        "x_served_by=" + (http.x_served_by or "").lower(),
        "issuer=" + (tls.issuer_cn or "").lower(),
        "alpn=" + (tls.alpn or "").lower(),
    ]
    return _sha256_hex("|".join(parts))[:16]


async def scan_domain(
    sem: asyncio.Semaphore,
    session: aiohttp.ClientSession,
    domain: str,
    *,
    dns_timeout_s: float,
    http_timeout_s: float,
    tls_timeout_s: float,
    ws_timeout_s: float,
    ws_paths: list[str],
) -> DomainScanResult:
    async with sem:
        res = DomainScanResult(domain=domain)

        # DNS CNAME chain (content-sharing/CDN clue) — not IP resolution.
        try:
            res.cname_chain = await resolve_cname_chain(domain, timeout_s=dns_timeout_s)
        except Exception as e:
            res.errors.append(f"cname: {e}")

        # HTTP fingerprint
        res.http = await fetch_http_fingerprint(session, domain, timeout_s=http_timeout_s)
        if not res.http.ok and res.http.error:
            res.errors.append(f"http: {res.http.error}")

        # TLS fingerprint (certificate issuer often clusters CDNs)
        res.tls = await fetch_tls_fingerprint(domain, timeout_s=tls_timeout_s)
        if not res.tls.ok and res.tls.error:
            res.errors.append(f"tls: {res.tls.error}")

        # WebSocket probes (optional; fast fail by timeout)
        if ws_paths:
            res.websockets = await probe_websockets(session, domain, paths=ws_paths, timeout_s=ws_timeout_s)

        res.signature = compute_signature(domain, res.cname_chain, res.http, res.tls)
        return res


def print_result(res: DomainScanResult) -> None:
    ws_ok = any(p.ok for p in res.websockets)
    ws_any = len(res.websockets) > 0

    print(f"\nDomain: {res.domain}")
    if res.cname_chain:
        print(f"  CNAME: {' -> '.join(res.cname_chain)}")
    if res.http.ok:
        print(f"  HTTP: {res.http.status}  final={res.http.final_url}")
        if res.http.server:
            print(f"    server: {res.http.server}")
        if res.http.via:
            print(f"    via: {res.http.via}")
        if res.http.x_cache:
            print(f"    x-cache: {res.http.x_cache}")
        if res.http.x_served_by:
            print(f"    x-served-by: {res.http.x_served_by}")
    else:
        print(f"  HTTP: error={res.http.error}")

    if res.tls.ok:
        print(f"  TLS: alpn={res.tls.alpn} issuer={res.tls.issuer_cn} subject={res.tls.subject_cn}")
    else:
        print(f"  TLS: error={res.tls.error}")

    if ws_any:
        print(f"  WS: {'ok' if ws_ok else 'no'}")
        for p in res.websockets:
            if p.ok:
                print(f"    ok  {p.url} ({p.ms:.1f}ms) proto={p.negotiated_subprotocol}")
            else:
                print(f"    no  {p.url} ({p.ms:.1f}ms) err={p.error}")

    print(f"  Signature: {res.signature}")
    if res.errors:
        print(f"  Errors: {', '.join(res.errors)}")


def group_by_signature(results: Iterable[DomainScanResult]) -> list[tuple[str, list[DomainScanResult]]]:
    groups: dict[str, list[DomainScanResult]] = {}
    for r in results:
        groups.setdefault(r.signature, []).append(r)
    out = sorted(groups.items(), key=lambda kv: len(kv[1]), reverse=True)
    return out


def write_jsonl(path: str, results: Iterable[DomainScanResult]) -> None:
    with open(path, "w", encoding="utf-8") as f:
        for r in results:
            f.write(json.dumps(asdict(r), ensure_ascii=False) + "\n")


def write_summary(path: str, grouped: list[tuple[str, list[DomainScanResult]]]) -> None:
    with open(path, "w", encoding="utf-8") as f:
        for sig, rs in grouped:
            f.write(f"# {sig} ({len(rs)} domains)\n")
            host = {}
            for r in rs:
                if r.http.final_host:
                    host[r.http.final_host] = host.get(r.http.final_host, 0) + 1
            if host:
                top = sorted(host.items(), key=lambda x: x[1], reverse=True)[:5]
                f.write("final_host: " + ", ".join([f"{k}({v})" for k, v in top]) + "\n")
            f.write("\n")
            for r in rs:
                f.write(f"- {r.domain}\n")
            f.write("\n")


async def scan_action(args: argparse.Namespace) -> int:
    domains = read_domains(args.domains_file)
    if not domains:
        print("No domains found.", file=sys.stderr)
        return 2

    ws_paths = []
    if args.ws:
        ws_paths = args.ws_paths or ["/", "/ws", "/websocket", "/socket.io/?EIO=4&transport=websocket"]
    else:
        ws_paths = []

    connector = aiohttp.TCPConnector(ssl=False, limit=0)
    headers = {"User-Agent": args.user_agent}
    sem = asyncio.Semaphore(args.concurrency)

    results: list[DomainScanResult] = []

    async with aiohttp.ClientSession(connector=connector, headers=headers) as session:
        tasks = [
            asyncio.create_task(
                scan_domain(
                    sem,
                    session,
                    d,
                    dns_timeout_s=args.dns_timeout,
                    http_timeout_s=args.http_timeout,
                    tls_timeout_s=args.tls_timeout,
                    ws_timeout_s=args.ws_timeout,
                    ws_paths=ws_paths,
                )
            )
            for d in domains
        ]

        for i, fut in enumerate(asyncio.as_completed(tasks), start=1):
            r = await fut
            results.append(r)
            if not args.quiet:
                print_result(r)
            if args.progress and i % args.progress == 0:
                print(f"\n-- progress: {i}/{len(domains)} --")

    grouped = group_by_signature(results)

    print("\n" + "=" * 80)
    print("GROUPED SIGNATURES (top)")
    print("=" * 80)
    for sig, rs in grouped[: args.top]:
        print(f"{sig}  n={len(rs)}")
        if args.show_domains:
            for r in rs[: args.show_domains]:
                print(f"  - {r.domain}")

    if args.jsonl:
        write_jsonl(args.jsonl, results)
        print(f"\nWrote JSONL: {args.jsonl}")
    if args.summary:
        write_summary(args.summary, grouped)
        print(f"Wrote summary: {args.summary}")

    return 0


def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="whitelists-research-v2",
        description="Scan domains for CDN/content-sharing fingerprints and WebSocket reachability.",
    )
    sub = p.add_subparsers(dest="cmd", required=True)

    scan = sub.add_parser("scan", help="scan domains from a file")
    scan.add_argument("domains_file", help="txt file with one domain per line")
    scan.add_argument("--concurrency", type=int, default=50)

    scan.add_argument("--dns-timeout", type=float, default=2.5)
    scan.add_argument("--http-timeout", type=float, default=8.0)
    scan.add_argument("--tls-timeout", type=float, default=5.0)
    scan.add_argument("--ws-timeout", type=float, default=6.0)

    scan.add_argument("--ws", action="store_true", help="enable WebSocket probes (wss)")
    scan.add_argument(
        "--ws-paths",
        nargs="*",
        default=[],
        help="override WS paths (default: /, /ws, /websocket, /socket.io/?EIO=4&transport=websocket)",
    )

    scan.add_argument("--jsonl", default="", help="write per-domain results as JSONL")
    scan.add_argument("--summary", default="", help="write grouped summary (markdown-ish)")

    scan.add_argument("--top", type=int, default=25, help="top N signatures to print")
    scan.add_argument("--show-domains", type=int, default=0, help="print first N domains per signature")
    scan.add_argument("--quiet", action="store_true", help="do not print per-domain results")
    scan.add_argument("--progress", type=int, default=0, help="print a progress line every N results")
    scan.add_argument("--user-agent", default="whitelists-research-v2/0.1", help="HTTP User-Agent")

    return p


def main(argv: list[str]) -> int:
    args = build_parser().parse_args(argv)

    if args.cmd == "scan":
        return asyncio.run(scan_action(args))

    raise RuntimeError(f"unknown cmd: {args.cmd}")


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))


