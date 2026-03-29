#!/usr/bin/env python3
import json
import sys
import zipfile


def main() -> int:
    zip_path = (
        sys.argv[1]
        if len(sys.argv) > 1
        else "/home/manuel/Downloads/data-2026-03-29-11-53-11-batch-0000.zip"
    )

    with zipfile.ZipFile(zip_path) as zf:
        conversations = json.loads(zf.read("conversations.json"))

    print("count", len(conversations))
    if not conversations:
        return 0

    first = conversations[0]
    print("keys", sorted(first.keys()))
    print("uuid", first.get("uuid"))
    print("chat_messages", len(first.get("chat_messages", [])))

    first_message = (first.get("chat_messages") or [{}])[0]
    print("message keys", sorted(first_message.keys()))
    print("first sender", first_message.get("sender"))
    print("first content type", type(first_message.get("content")).__name__)
    print("first content sample", (first_message.get("content") or [])[:1])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
