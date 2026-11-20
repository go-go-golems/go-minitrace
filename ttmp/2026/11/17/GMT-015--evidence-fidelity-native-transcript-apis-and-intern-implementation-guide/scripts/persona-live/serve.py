#!/usr/bin/env python3
"""Serve a single private batch timeline over loopback."""
import argparse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
from urllib.parse import parse_qs, urlsplit

VIEWER = Path(__file__).with_name("viewer.html")


def timeline(batch, positions):
    updates = {}
    for path in sorted(batch.glob("*/events.jsonl")):
        run = path.parent.name
        offset = positions.get(run, 0)
        with path.open("rb") as log:
            log.seek(offset if 0 <= offset <= path.stat().st_size else 0)
            rows = []
            while len(rows) < 100:
                line = log.readline()
                if not line.endswith(b"\n"):
                    break
                rows.append(json.loads(line))
                offset = log.tell()
        updates[run] = {"events": rows, "offset": offset}
    return updates


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("batch", type=Path)
    parser.add_argument("--port", default=8794, type=int)
    args = parser.parse_args()
    batch = args.batch.resolve()

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            # Disallow Host aliases so a remote page cannot DNS-rebind this viewer.
            if self.headers.get("Host") not in (f"127.0.0.1:{args.port}", f"localhost:{args.port}"):
                return self.send_error(403)
            if self.headers.get("Sec-Fetch-Site") == "cross-site":
                return self.send_error(403)
            url = urlsplit(self.path)
            if url.path == "/":
                body, mime = VIEWER.read_bytes(), "text/html; charset=utf-8"
            elif url.path == "/events":
                try:
                    offsets = json.loads(parse_qs(url.query).get("after", ["{}"])[0])
                    if not isinstance(offsets, dict) or any(type(n) is not int for n in offsets.values()):
                        raise ValueError("offsets must be integer values")
                except ValueError:
                    return self.send_error(400)
                batch_path = batch / "batch.json"
                meta = json.loads(batch_path.read_text()) if batch_path.exists() else {}
                body = json.dumps({"meta": meta, "runs": timeline(batch, offsets)}).encode()
                mime = "application/json"
            else:
                return self.send_error(404)
            self.send_response(200)
            self.send_header("Content-Type", mime)
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            self.send_header("X-Content-Type-Options", "nosniff")
            self.send_header("Referrer-Policy", "no-referrer")
            self.send_header("Content-Security-Policy", "default-src 'none'; connect-src 'self'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; frame-ancestors 'none'")
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, format, *values):
            if len(values) > 1 and values[1] != "200":
                super().log_message(format, *values)

    with ThreadingHTTPServer(("127.0.0.1", args.port), Handler) as server:
        print(f"http://127.0.0.1:{args.port}/ · {batch.name}", flush=True)
        server.serve_forever()


if __name__ == "__main__":
    main()
