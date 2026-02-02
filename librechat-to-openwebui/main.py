#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#   "orjson",
# ]
# ///

from __future__ import annotations

import argparse
import datetime as dt
import sys
from pathlib import Path
from typing import Any, Iterable

ZERO_UUID = "00000000-0000-0000-0000-000000000000"
FOLDER_ID = "7cebb824-927d-4713-8bff-421518e9bb22"
IMPORTED_TAG = "Imported"

try:
    import orjson  # type: ignore

    def _json_loads(b: bytes) -> Any:
        return orjson.loads(b)

    def _json_dumps(obj: Any, *, pretty: bool) -> bytes:
        opts = 0
        if pretty:
            opts |= orjson.OPT_INDENT_2
        return orjson.dumps(obj, option=opts)

except Exception:  # pragma: no cover
    import json

    def _json_loads(b: bytes) -> Any:
        return json.loads(b.decode("utf-8"))

    def _json_dumps(obj: Any, *, pretty: bool) -> bytes:
        if pretty:
            return json.dumps(obj, ensure_ascii=False, indent=2).encode("utf-8")
        return json.dumps(obj, ensure_ascii=False, separators=(",", ":")).encode("utf-8")


def _read_json(path: Path) -> Any:
    return _json_loads(path.read_bytes())


def parse_mongodb_date(value: Any) -> dt.datetime | None:
    """
    Accepts common MongoDB export representations:
    - {"$date": "2025-03-15T12:57:23.586Z"}
    - {"$date": 1710000000000}
    - {"$date": {"$numberLong": "1710000000000"}}
    - ISO8601 string
    Returns aware datetime in UTC when possible.
    """
    if value is None:
        return None

    # {"$date": ...}
    if isinstance(value, dict) and "$date" in value:
        value = value["$date"]

    # {"$numberLong": "..."}
    if isinstance(value, dict) and "$numberLong" in value:
        value = value["$numberLong"]

    # epoch millis
    if isinstance(value, (int, float)):
        try:
            return dt.datetime.fromtimestamp(float(value) / 1000.0, tz=dt.UTC)
        except Exception:
            return None

    # string: epoch millis or ISO
    if isinstance(value, str):
        s = value.strip()
        if not s:
            return None
        if s.isdigit():
            try:
                return dt.datetime.fromtimestamp(int(s) / 1000.0, tz=dt.UTC)
            except Exception:
                return None
        # ISO8601 (with optional Z)
        try:
            if s.endswith("Z"):
                s = s[:-1] + "+00:00"
            d = dt.datetime.fromisoformat(s)
            if d.tzinfo is None:
                d = d.replace(tzinfo=dt.UTC)
            return d.astimezone(dt.UTC)
        except Exception:
            return None

    return None


def dt_to_unix_seconds(d: dt.datetime | None) -> int:
    if d is None:
        return 0
    return int(d.timestamp())


def dt_to_unix_millis(d: dt.datetime | None) -> int:
    if d is None:
        return 0
    return int(d.timestamp() * 1000)


def normalize_parent_id(parent_id: Any, existing_ids: set[str]) -> str | None:
    if parent_id in (None, "", ZERO_UUID):
        return None
    pid = str(parent_id)
    if pid not in existing_ids:
        # Avoid dangling references which can cause OpenWebUI to drop parts of the tree.
        return None
    return pid


def sort_messages(messages: list[dict[str, Any]]) -> list[dict[str, Any]]:
    def key(m: dict[str, Any]) -> tuple[int, int]:
        created = dt_to_unix_millis(parse_mongodb_date(m.get("createdAt")))
        updated = dt_to_unix_millis(parse_mongodb_date(m.get("updatedAt")))
        return (created or updated, updated)

    return sorted(messages, key=key)


def build_children_map(messages_sorted: list[dict[str, Any]]) -> dict[str, list[str]]:
    children: dict[str, list[str]] = {}
    for m in messages_sorted:
        mid = str(m.get("messageId") or "")
        if not mid:
            continue
        pid = m.get("parentMessageId")
        if pid in (None, "", ZERO_UUID):
            continue
        pid_s = str(pid)
        children.setdefault(pid_s, []).append(mid)
    return children


def _collect_content_text_fragments(value: Any, out: list[str]) -> None:
    if value is None:
        return
    if isinstance(value, str):
        if value.strip():
            out.append(value)
        return
    if isinstance(value, dict):
        text = value.get("text")
        if isinstance(text, str) and text.strip():
            out.append(text)
        tool_call = value.get("tool_call")
        if isinstance(tool_call, dict):
            output = tool_call.get("output")
            if isinstance(output, str) and output.strip():
                out.append(output)
        _collect_content_text_fragments(value.get("content"), out)
        return
    if isinstance(value, list):
        for item in value:
            _collect_content_text_fragments(item, out)


def _extract_message_content(msg: dict[str, Any]) -> str:
    if not isinstance(msg, dict):
        return ""
    fragments: list[str] = []
    text = msg.get("text")
    if isinstance(text, str) and text.strip():
        fragments.append(text)
    _collect_content_text_fragments(msg.get("content"), fragments)
    return "\n".join(fragments).strip()


def convert_message(
    msg: dict[str, Any],
    *,
    children_ids: list[str],
    parent_id: str | None,
    conv_model: str | None,
) -> dict[str, Any]:
    mid = str(msg.get("messageId") or "")
    is_user = bool(msg.get("isCreatedByUser"))
    role = "user" if is_user else "assistant"

    d_created = parse_mongodb_date(msg.get("createdAt")) or parse_mongodb_date(msg.get("updatedAt"))
    timestamp_s = dt_to_unix_seconds(d_created)

    model = msg.get("model") or conv_model or ""
    content = _extract_message_content(msg)

    out: dict[str, Any] = {
        "id": mid,
        "parentId": parent_id,
        "childrenIds": children_ids,
        "role": role,
        "content": content,
        "timestamp": timestamp_s,
    }

    if is_user:
        out["models"] = [str(model)] if model else []
    else:
        out["model"] = str(model) if model else ""
        out["modelName"] = str(model) if model else ""
        out["modelIdx"] = 0
        out["done"] = not bool(msg.get("unfinished"))

    return out


def convert_conversation(
    conv: dict[str, Any],
    *,
    messages: list[dict[str, Any]],
    tags_by_id: dict[str, str],
) -> dict[str, Any]:
    conv_id = str(conv.get("conversationId") or "")
    conv_title = str(conv.get("title") or "")
    conv_user = str(conv.get("user") or "")
    conv_model = conv.get("model")
    conv_model_s = str(conv_model) if conv_model else None

    d_created = parse_mongodb_date(conv.get("createdAt"))
    d_updated = parse_mongodb_date(conv.get("updatedAt"))

    created_s = dt_to_unix_seconds(d_created)
    updated_s = dt_to_unix_seconds(d_updated) or created_s

    # OpenWebUI export example uses ms for chat.timestamp.
    chat_timestamp_ms = dt_to_unix_millis(d_created) or (created_s * 1000)

    # Deduplicate (defensive) then sort.
    uniq: list[dict[str, Any]] = []
    seen: set[str] = set()
    for m in messages:
        mid = str(m.get("messageId") or "")
        if not mid or mid in seen:
            continue
        seen.add(mid)
        uniq.append(m)
    messages_sorted = sort_messages(uniq)

    ids = {str(m.get("messageId") or "") for m in messages_sorted if m.get("messageId")}
    children_map = build_children_map(messages_sorted)

    # Convert all messages for both "history.messages" map and "messages" array.
    history_messages: dict[str, dict[str, Any]] = {}
    messages_array: list[dict[str, Any]] = []
    for m in messages_sorted:
        mid = str(m.get("messageId") or "")
        if not mid:
            continue
        parent_id = normalize_parent_id(m.get("parentMessageId"), ids)
        out_msg = convert_message(
            m,
            children_ids=children_map.get(mid, []),
            parent_id=parent_id,
            conv_model=conv_model_s,
        )
        history_messages[mid] = out_msg
        messages_array.append(out_msg)

    current_id = messages_array[-1]["id"] if messages_array else None

    # Tags in your sample conversations are empty, but support mapping if present.
    conv_tag_ids = conv.get("tags") or []
    meta_tags: list[str] = []
    if isinstance(conv_tag_ids, list):
        for t in conv_tag_ids:
            if isinstance(t, dict) and "$oid" in t:
                tid = str(t["$oid"])
            else:
                tid = str(t)
            name = tags_by_id.get(tid)
            if name:
                meta_tags.append(name)
    if IMPORTED_TAG not in meta_tags:
        meta_tags.append(IMPORTED_TAG)
    # Always provide meta.tags as an array, OpenWebUI expects it even if empty.

    return {
        "id": conv_id,
        "user_id": conv_user,
        "title": conv_title,
        "chat": {
            "id": "",  # matches OpenWebUI export examples
            "title": conv_title,
            "models": [conv_model_s] if conv_model_s else [],
            "params": {},
            "history": {"messages": history_messages, "currentId": current_id},
            "messages": messages_array,
            "tags": [],
            "timestamp": chat_timestamp_ms,
            "files": conv.get("files") or [],
        },
        "updated_at": updated_s,
        "created_at": created_s,
        "share_id": None,
        "archived": bool(conv.get("isArchived", False)),
        "pinned": False,
        "meta": {"tags": meta_tags},
        "folder_id": FOLDER_ID,
    }


def build_tags_map(tags: Any) -> dict[str, str]:
    """
    LibreChat.conversationtags.json entries look like:
      { "_id": {"$oid": "..."} , "tag": "Name", ... }
    """
    out: dict[str, str] = {}
    if not isinstance(tags, list):
        return out
    for t in tags:
        if not isinstance(t, dict):
            continue
        _id = t.get("_id")
        tid = None
        if isinstance(_id, dict) and "$oid" in _id:
            tid = str(_id["$oid"])
        elif _id is not None:
            tid = str(_id)
        name = t.get("tag")
        if tid and isinstance(name, str) and name.strip():
            out[tid] = name.strip()
    return out


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(
        description="Convert LibreChat MongoDB export JSONs (conversations/messages/tags) into an OpenWebUI import JSON."
    )
    p.add_argument(
        "--data-dir",
        type=Path,
        default=Path(__file__).resolve().parent / "data",
        help="Directory containing LibreChat.conversations.json, LibreChat.messages.json, LibreChat.conversationtags.json",
    )
    p.add_argument(
        "--conversations",
        type=Path,
        default=None,
        help="Override path to LibreChat.conversations.json",
    )
    p.add_argument(
        "--messages",
        type=Path,
        default=None,
        help="Override path to LibreChat.messages.json",
    )
    p.add_argument(
        "--tags",
        type=Path,
        default=None,
        help="Override path to LibreChat.conversationtags.json",
    )
    p.add_argument(
        "-o",
        "--output",
        type=Path,
        default=None,
        help="Output file path (default: stdout)",
    )
    p.add_argument(
        "--pretty",
        action="store_true",
        help="Pretty-print JSON output (indent=2).",
    )
    args = p.parse_args(argv)

    conv_path = args.conversations or (args.data_dir / "LibreChat.conversations.json")
    msg_path = args.messages or (args.data_dir / "LibreChat.messages.json")
    tag_path = args.tags or (args.data_dir / "LibreChat.conversationtags.json")

    conversations = _read_json(conv_path)
    messages = _read_json(msg_path)
    tags = _read_json(tag_path) if tag_path.exists() else []

    if not isinstance(conversations, list):
        raise SystemExit("Expected conversations JSON to be a list")
    if not isinstance(messages, list):
        raise SystemExit("Expected messages JSON to be a list")

    tags_by_id = build_tags_map(tags)

    # Group messages by conversationId.
    by_conv: dict[str, list[dict[str, Any]]] = {}
    for m in messages:
        if not isinstance(m, dict):
            continue
        cid = m.get("conversationId")
        if not cid:
            continue
        by_conv.setdefault(str(cid), []).append(m)

    out_convs: list[dict[str, Any]] = []
    for conv in conversations:
        if not isinstance(conv, dict):
            continue
        cid = str(conv.get("conversationId") or "")
        if not cid:
            continue
        out_convs.append(
            convert_conversation(
                conv,
                messages=by_conv.get(cid, []),
                tags_by_id=tags_by_id,
            )
        )

    data = _json_dumps(out_convs, pretty=args.pretty)
    if args.output:
        args.output.write_bytes(data + b"\n")
    else:
        sys.stdout.buffer.write(data + b"\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
