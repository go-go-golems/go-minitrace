#!/usr/bin/env python3
import collections
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

    senders = collections.Counter()
    block_types = collections.Counter()
    tools = collections.Counter()
    samples = []

    for conversation in conversations:
        for message in conversation.get("chat_messages", []):
            senders[message.get("sender")] += 1
            for block in message.get("content") or []:
                if not isinstance(block, dict):
                    continue
                block_type = block.get("type")
                block_types[block_type] += 1
                if block_type == "tool_use":
                    tools[block.get("name")] += 1
                    if len(samples) < 10:
                        samples.append(
                            (
                                conversation.get("uuid"),
                                message.get("sender"),
                                block.get("name"),
                                sorted(block.keys()),
                            )
                        )

    print("senders", senders)
    print("block_types", block_types)
    print("tools", tools)
    print("sample tool entries", samples)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
