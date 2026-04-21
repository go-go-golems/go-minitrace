#!/usr/bin/env python3
"""
reconstruct_files.py — Recover files from a minitrace archive by replaying
write/edit tool calls against a target directory.

Usage:
    # Reconstruct all files from a minitrace archive
    python reconstruct_files.py \
        --archive ./analysis/jellyfin-session/active/2026-04/session.minitrace.json \
        --target /tmp/recovered

    # Reconstruct only files matching a path pattern
    python reconstruct_files.py \
        --archive ./analysis/session.minitrace.json \
        --target /tmp/recovered \
        --filter "jellyfin-001"

    # Dry run — show what would be recovered without writing
    python reconstruct_files.py \
        --archive ./analysis/session.minitrace.json \
        --target /tmp/recovered \
        --dry-run

    # Also replay bash commands that write files (e.g. docmgr, tee, heredocs)
    python reconstruct_files.py \
        --archive ./analysis/session.minitrace.json \
        --target /tmp/recovered \
        --include-bash-writes
"""

import argparse
import json
import os
import re
import sys
from pathlib import Path


def expand_home(path: str) -> str:
    """Expand ~ and $HOME in paths."""
    home = os.environ.get("HOME", "/tmp")
    return path.replace("~/", home + "/").replace("$HOME/", home + "/")


# Patterns that indicate a tool call actually failed despite success=True
_FAILED_RESULT_PATTERNS = [
    "File not found",
    "No such file",
    "Permission denied",
    "error:",
    "Error:",
]


def _tool_actually_succeeded(output: dict) -> bool:
    """Check if a tool call really succeeded.
    
    Pi's edit/write tools can return success=True even when they report
    'File not found' or other errors in the result text. This function
    cross-checks the result string for known failure patterns.
    """
    if not output.get("success"):
        return False
    result = output.get("result", "") or ""
    for pat in _FAILED_RESULT_PATTERNS:
        if pat in result:
            return False
    return True


def apply_edit(content: str, edits: list[dict]) -> str:
    """Apply a list of {oldText, newText} edits to content, sequentially."""
    for edit in edits:
        old = edit.get("oldText", "")
        new = edit.get("newText", "")
        if old not in content:
            raise ValueError(
                f"oldText not found in content (len={len(content)}, "
                f"oldText starts with: {old[:80]!r}...)"
            )
        content = content.replace(old, new, 1)
    return content


def reconstruct_from_archive(archive_path: str, target_dir: str, path_filter: str | None = None,
                              dry_run: bool = False, include_bash: bool = False) -> None:
    """Reconstruct files by replaying write/edit tool calls from a minitrace archive."""
    with open(archive_path) as f:
        data = json.load(f)

    tool_calls = data.get("tool_calls", [])
    if not tool_calls:
        print("No tool_calls found in archive.")
        return

    # Track file states: {abs_path -> current_content}
    file_states: dict[str, str] = {}
    operations: list[dict] = []

    for tc in tool_calls:
        tool = tc.get("tool_name", "")
        inp = tc.get("input", {})
        out = tc.get("output", {})
        args = inp.get("arguments", {})
        turn = tc.get("emitting_turn_index", "?")

        if tool == "write":
            path = args.get("path") or inp.get("file_path", "")
            content = args.get("content", "")
            if not path:
                continue
            path = expand_home(path)
            if path_filter and path_filter not in path:
                continue
            actually_ok = _tool_actually_succeeded(out)
            operations.append({
                "turn": turn,
                "tool": "write",
                "path": path,
                "content": content,
                "success": actually_ok,
                "ts": tc.get("timestamp"),
            })
            if actually_ok:
                file_states[path] = content

        elif tool == "edit":
            path = args.get("path") or inp.get("file_path", "")
            edits = args.get("edits", [])
            if not path or not edits:
                continue
            path = expand_home(path)
            if path_filter and path_filter not in path:
                continue
            actually_ok = _tool_actually_succeeded(out)
            operations.append({
                "turn": turn,
                "tool": "edit",
                "path": path,
                "edits": edits,
                "success": actually_ok,
                "ts": tc.get("timestamp"),
            })
            if actually_ok:
                # Apply edit to current state (or start from empty if unknown)
                current = file_states.get(path, "")
                try:
                    file_states[path] = apply_edit(current, edits)
                except ValueError as e:
                    operations[-1]["error"] = str(e)

        elif tool == "bash" and include_bash:
            cmd = inp.get("command", "")
            result = out.get("result", "") or ""
            # Detect bash writes: commands that create files
            # Pattern: docmgr, tee, cat > file, heredoc, cp, mv
            bash_write_patterns = [
                # docmgr ticket/doc creation
                r"docmgr\s+(?:ticket\s+create-ticket|doc\s+add)",
            ]
            for pat in bash_write_patterns:
                if re.search(pat, cmd) and out.get("success"):
                    operations.append({
                        "turn": turn,
                        "tool": "bash",
                        "command": cmd,
                        "result_preview": result[:200],
                        "ts": tc.get("timestamp"),
                    })
                    break

    # Display operations timeline
    print(f"\n{'='*80}")
    print(f"RECONSTRUCTION REPORT")
    print(f"Archive: {archive_path}")
    print(f"Session: {data.get('id', 'unknown')}")
    print(f"Target:  {target_dir}")
    if path_filter:
        print(f"Filter:  {path_filter}")
    print(f"{'='*80}\n")

    print(f"Found {len(operations)} file operations:\n")

    # Group by file
    files_ops: dict[str, list[dict]] = {}
    for op in operations:
        p = op.get("path", "(bash)")
        files_ops.setdefault(p, []).append(op)

    for filepath, ops in sorted(files_ops.items()):
        print(f"\n--- {filepath} ---")
        for op in ops:
            status = "✅" if op.get("success") else "❌"
            err = f" ERROR: {op['error']}" if op.get("error") else ""
            if op["tool"] == "write":
                print(f"  {status} Turn {op['turn']:>3} | WRITE  | {len(op['content']):>6} bytes | {op['ts']}{err}")
            elif op["tool"] == "edit":
                n_edits = len(op.get("edits", []))
                print(f"  {status} Turn {op['turn']:>3} | EDIT   | {n_edits} edits    | {op['ts']}{err}")
            elif op["tool"] == "bash":
                print(f"  {status} Turn {op['turn']:>3} | BASH   | {op.get('command', '')[:60]}")

    # Write final state of each file
    print(f"\n{'='*80}")
    print("FINAL FILE STATES")
    print(f"{'='*80}\n")

    recovered = 0
    skipped = 0
    for filepath, content in sorted(file_states.items()):
        if path_filter and path_filter not in filepath:
            continue

        # Map absolute path to target directory
        # Strip the home prefix and write under target
        home = os.environ.get("HOME", "/tmp")
        rel = filepath
        if rel.startswith(home + "/"):
            rel = rel[len(home) + 1:]
        # Also try stripping leading /home/user/
        if rel.startswith("/home/"):
            parts = rel.split("/", 3)
            if len(parts) >= 4:
                rel = parts[3]

        outpath = os.path.join(target_dir, rel)
        nbytes = len(content.encode("utf-8"))

        if dry_run:
            print(f"  [DRY] Would write {nbytes:>6} bytes -> {outpath}")
        else:
            os.makedirs(os.path.dirname(outpath), exist_ok=True)
            with open(outpath, "w") as f:
                f.write(content)
            print(f"  ✅ Wrote {nbytes:>6} bytes -> {outpath}")
            recovered += 1

    # Also generate a per-version dump of each file (every write/edit creates a version)
    if not dry_run:
        versions_dir = os.path.join(target_dir, ".versions")
        os.makedirs(versions_dir, exist_ok=True)

        for filepath, ops_list in sorted(files_ops.items()):
            if filepath == "(bash)":
                continue
            if path_filter and path_filter not in filepath:
                continue

            # Replay operations to get each version
            state = None
            version = 0
            for op in operations:
                if op.get("path") != filepath:
                    continue
                if not op.get("success"):
                    # Skip failed operations — they didn't actually modify the file
                    continue
                if op["tool"] == "write":
                    state = op["content"]
                    version += 1
                elif op["tool"] == "edit":
                    if state is None:
                        state = ""
                    try:
                        state = apply_edit(state, op["edits"])
                        version += 1
                    except ValueError:
                        pass

                if state is not None:
                    home = os.environ.get("HOME", "/tmp")
                    rel = filepath
                    if rel.startswith(home + "/"):
                        rel = rel[len(home) + 1:]
                    if rel.startswith("/home/"):
                        parts = rel.split("/", 3)
                        if len(parts) >= 4:
                            rel = parts[3]

                    basename = os.path.basename(rel)
                    dirname = os.path.dirname(rel).replace("/", "_")
                    vername = f"v{version:03d}-turn{op['turn']:03d}-{basename}"
                    verpath = os.path.join(versions_dir, f"{dirname}__{vername}")
                    with open(verpath, "w") as f:
                        f.write(state)

    print(f"\nRecovered: {recovered} files, Skipped: {skipped}")
    if not dry_run:
        print(f"Version history written to: {os.path.join(target_dir, '.versions/')}")


def main():
    parser = argparse.ArgumentParser(
        description="Reconstruct files from a minitrace archive by replaying tool calls",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument(
        "--archive", "-a",
        required=True,
        help="Path to a .minitrace.json file",
    )
    parser.add_argument(
        "--target", "-t",
        required=True,
        help="Directory to write recovered files into",
    )
    parser.add_argument(
        "--filter", "-f",
        default=None,
        help="Only reconstruct files whose path contains this string",
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Show what would be recovered without writing files",
    )
    parser.add_argument(
        "--include-bash-writes", "-b",
        action="store_true",
        help="Also detect file-writing bash commands (e.g. docmgr, tee)",
    )

    args = parser.parse_args()

    if not os.path.exists(args.archive):
        print(f"Error: archive not found: {args.archive}", file=sys.stderr)
        sys.exit(1)

    reconstruct_from_archive(
        archive_path=args.archive,
        target_dir=args.target,
        path_filter=args.filter,
        dry_run=args.dry_run,
        include_bash=args.include_bash_writes,
    )


if __name__ == "__main__":
    main()
