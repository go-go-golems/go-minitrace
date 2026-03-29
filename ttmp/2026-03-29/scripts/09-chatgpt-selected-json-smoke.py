#!/usr/bin/env python3
import json
import os
import subprocess
import tempfile
import zipfile


REPO_DIR = "/home/manuel/code/wesen/corporate-headquarters/go-minitrace"
SOURCE_JSON = "/home/manuel/Downloads/mento-selected-conversations-2025-07-20T19-12-41.json"


def main() -> int:
    with open(SOURCE_JSON) as f:
        data = json.load(f)

    if not isinstance(data, list):
        raise SystemExit("expected a JSON list")

    with tempfile.TemporaryDirectory(prefix="go-minitrace-chatgpt-") as td:
        zip_path = os.path.join(td, "chatgpt-export.zip")
        with zipfile.ZipFile(zip_path, "w") as zf:
            zf.writestr("conversations.json", json.dumps(data))

        proc = subprocess.run(
            [
                "go",
                "run",
                "./cmd/go-minitrace",
                "convert",
                "chatgpt",
                "--source",
                zip_path,
                "--dry-run",
                "--output",
                "json",
            ],
            cwd=REPO_DIR,
            text=True,
            capture_output=True,
            check=False,
        )

        print("returncode", proc.returncode)
        print(proc.stdout[:4000])
        if proc.stderr:
            print(proc.stderr[:2000])
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
