#!/usr/bin/env -S uv run
import argparse
import json
import os
import sys
import urllib.request
from pathlib import Path

STATE_FILE = Path.home() / ".oneshot_state.json"

SYSTEM_PROMPT = (
    "You are a command-line assistant. "
    "Output ONLY the command, query, or code that answers the request. "
    "No explanations, no markdown, no code fences. "
    "For shell commands, output a single line; use backslash continuations if needed."
)


def load_state() -> list[dict]:
    try:
        return json.loads(STATE_FILE.read_text()).get("messages", [])
    except Exception:
        return []


def save_state(messages: list[dict]) -> None:
    STATE_FILE.write_text(json.dumps({"messages": messages}, ensure_ascii=False, indent=2))


def chat(messages: list[dict]) -> dict:
    api_key = os.environ.get("ONE_SHOT_API_KEY")
    if not api_key:
        sys.exit("error: ONE_SHOT_API_KEY is not set")

    base = os.environ.get("ONE_SHOT_BASE_URL", "https://api.openai.com/v1").rstrip("/")
    model = os.environ.get("ONE_SHOT_MODEL", "gpt-4o-mini")

    body = json.dumps({"model": model, "messages": messages, "temperature": 1}).encode()
    req = urllib.request.Request(
        f"{base}/chat/completions",
        data=body,
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
    )

    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            data = json.load(resp)
    except urllib.error.HTTPError as e:
        sys.exit(f"error: {e.code} {e.read().decode()[:200]}")
    except urllib.error.URLError as e:
        sys.exit(f"error: {e.reason}")

    msg = data["choices"][0]["message"]
    return {"role": msg["role"], "content": msg["content"]}


def main():
    p = argparse.ArgumentParser(prog="one-shot")
    p.add_argument("-c", "--continue", dest="cont", action="store_true", help="continue previous conversation")
    p.add_argument("prompt")
    args = p.parse_args()

    messages = load_state() if args.cont else []
    if not messages:
        messages = [{"role": "system", "content": SYSTEM_PROMPT}]
    messages.append({"role": "user", "content": args.prompt})

    reply = chat(messages)
    print(reply["content"].strip())

    messages.append(reply)
    save_state(messages)


if __name__ == "__main__":
    main()
