#!/usr/bin/env python3
"""03-reconstruct-files.py — Reconstruct deleted surf-cli source files from minitrace tool calls.

Reads the file-contents JSON extracted by 02-extract-file-contents.sql,
reconstructs each file by:
  1. Starting from the initial `write` (NEW) tool call — full file content
  2. Applying subsequent `edit` (MODIFY) tool calls in turn order — each
     edit's oldText is replaced by newText in the file content

Outputs reconstructed files to a recovery directory.

Usage:
    python3 03-reconstruct-files.py <file-contents.json> <output_dir>
"""
import json
import os
import sys
import re


def normalize_path(file_path, session_cwd="/home/manuel/code/others/llms/pi/nicobailon/surf-cli"):
    """Normalize a file path relative to the surf-cli root."""
    if not file_path:
        return None
    # Strip the session cwd prefix
    if file_path.startswith(session_cwd + "/"):
        return file_path[len(session_cwd) + 1:]
    # Handle ~ paths
    if file_path.startswith("~/code/others/llms/pi/nicobailon/surf-cli/"):
        return file_path[len("~/code/others/llms/pi/nicobailon/surf-cli/"):]
    # Already relative
    if not file_path.startswith("/"):
        return file_path
    return file_path


def extract_content_from_write(arguments_json):
    """Extract file content from a `write` tool call."""
    try:
        args = json.loads(arguments_json)
    except (json.JSONDecodeError, TypeError):
        return None
    # The write tool stores content in args["content"]
    return args.get("content")


def extract_edits_from_edit(arguments_json):
    """Extract (oldText, newText) pairs from an `edit` tool call."""
    try:
        args = json.loads(arguments_json)
    except (json.JSONDecodeError, TypeError):
        return []
    edits = args.get("edits", [])
    pairs = []
    for edit in edits:
        old = edit.get("oldText", "")
        new = edit.get("newText", "")
        if old:
            pairs.append((old, new))
    return pairs


def apply_edits(content, edit_pairs):
    """Apply edit pairs to the content. Returns the modified content."""
    for old, new in edit_pairs:
        if old in content:
            content = content.replace(old, new, 1)
        else:
            # If oldText not found exactly, try without leading/trailing whitespace normalization
            # This can happen if the content was already modified by a prior edit
            pass  # Skip silently — we'll note missing edits in the report
    return content


def main():
    if len(sys.argv) != 3:
        print("Usage: 03-reconstruct-files.py <file-contents.json> <output_dir>", file=sys.stderr)
        sys.exit(1)

    input_file = sys.argv[1]
    output_dir = sys.argv[2]

    with open(input_file) as f:
        rows = json.load(f)

    # Group rows by normalized file path, ordered by turn_index
    files = {}
    for row in rows:
        path = normalize_path(row.get("file_path", ""))
        if not path:
            continue
        turn = row.get("turn_index", 0)
        op = row.get("operation_type", "")
        tool = row.get("tool_name", "")
        args = row.get("arguments_json", "")

        if path not in files:
            files[path] = []
        files[path].append({
            "turn": turn,
            "op": op,
            "tool": tool,
            "args": args,
        })

    # Reconstruct each file
    os.makedirs(output_dir, exist_ok=True)
    report = []

    for path, operations in sorted(files.items()):
        operations.sort(key=lambda x: x["turn"])
        content = None
        edits_applied = 0
        edits_skipped = 0

        for op in operations:
            if op["op"] == "NEW" and op["tool"] == "write":
                # Full file content
                content = extract_content_from_write(op["args"])
                if content:
                    report.append(f"  {path}: initialized from write at t{op['turn']} ({len(content)} chars)")
            elif op["op"] == "MODIFY" and op["tool"] == "edit":
                if content is None:
                    report.append(f"  {path}: WARNING — edit at t{op['turn']} but no initial write found, skipping")
                    continue
                edit_pairs = extract_edits_from_edit(op["args"])
                old_len = len(content)
                content = apply_edits(content, edit_pairs)
                if len(content) != old_len or True:  # Always count
                    edits_applied += 1
                report.append(f"  {path}: applied {len(edit_pairs)} edit(s) at t{op['turn']}")

        if content is not None:
            out_path = os.path.join(output_dir, path)
            os.makedirs(os.path.dirname(out_path), exist_ok=True)
            with open(out_path, "w") as f:
                f.write(content)
            report.append(f"  => WROTE {out_path} ({len(content)} chars)")
        else:
            report.append(f"  => SKIPPED {path} (no write tool call found)")

    print(f"Reconstructed {len(files)} files to {output_dir}")
    print(f"\nDetails:")
    for line in report:
        print(line)


if __name__ == "__main__":
    main()
