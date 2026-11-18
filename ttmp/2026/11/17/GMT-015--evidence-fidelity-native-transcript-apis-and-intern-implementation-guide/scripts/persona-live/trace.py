"""Project Pi phase order into the viewer without including reasoning text."""
import argparse
import json
from pathlib import Path


class Thinking:
    def __init__(self):
        self.starts = {}

    def event(self, raw, now=None):
        if raw.get("type") != "message_update":
            return None
        update = raw.get("assistantMessageEvent", {})
        kind = update.get("type")
        index = update.get("contentIndex")
        if kind == "thinking_start":
            self.starts[index] = now
            return "thinking start", "Thinking started", None
        if kind == "thinking_end":
            start = self.starts.pop(index, None)
            data = {"elapsed_seconds": round(now - start, 2)} if now is not None and start is not None else None
            return "thinking end", "Thinking ended", data
        return None


def public_message(raw):
    message = raw.get("message") or {}
    if raw.get("type") != "message_end" or message.get("role") != "assistant":
        return ""
    return "".join(block.get("text", "") for block in message.get("content", [])
                   if isinstance(block, dict) and block.get("type") == "text")


def projected_phase(raw_path):
    thinking = Thinking()
    with raw_path.open() as handle:
        for line in handle:
            try:
                raw = json.loads(line)
            except ValueError:
                continue
            activity = thinking.event(raw)
            if activity:
                yield ("thinking", activity)
            elif raw.get("type") in {"tool_execution_start", "tool_execution_end"}:
                kind = ("tool call" if raw["type"] == "tool_execution_start" else
                        "tool failure" if raw.get("isError") else "tool result")
                yield ("observed", (kind, raw["toolName"], raw.get("toolCallId")))
            elif text := public_message(raw):
                yield ("observed", ("message", text, None))


def project_run(directory):
    log = directory / "events.jsonl"
    events = [json.loads(line) for line in log.read_text().splitlines()]
    if any("tool_call_id" in event or event["kind"] in ("thinking start", "thinking end") for event in events):
        yield from events
        return
    phase = None
    role = None
    template = None

    def add_thinking(payload):
        kind, text, data = payload
        return dict(time=None, run=template["run"], role=role, kind=kind, text=text, data=data)

    for event in events:
        kind = event["kind"]
        if kind in {"waiting", "replay", "failure", "verdict", "finish"} and phase is not None:
            for origin, payload in phase:
                if origin != "thinking":
                    raise ValueError(f"Unmatched Pi event in {directory.name}: {payload[0]}")
                yield add_thinking(payload)
            phase = None
        if kind == "waiting":
            template = event
            role = event["role"]
            name = event["text"].split(" · ")[-1]
            raw_path = directory / (name + ".pi.jsonl")
            if raw_path.is_file():
                phase = projected_phase(raw_path)
        elif phase is not None and event["role"] == role and kind in {"message", "tool call", "tool result", "tool failure"}:
            for origin, payload in phase:
                if origin == "thinking":
                    yield add_thinking(payload)
                    continue
                expected_kind, expected_text, tool_id = payload
                if (expected_kind, expected_text) != (kind, event["text"]):
                    raise ValueError(f"Pi order mismatch in {directory.name}: {expected_kind} vs {kind}")
                if tool_id:
                    event["tool_call_id"] = tool_id
                break
            else:
                raise ValueError(f"Missing Pi event in {directory.name}: {kind}")
        yield event


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("original", type=Path, help="Completed batch with raw Pi phase files")
    parser.add_argument("destination", type=Path, help="Fresh read-only viewer projection")
    args = parser.parse_args()
    args.destination.mkdir(mode=0o700, parents=True, exist_ok=False)
    (args.destination / "batch.json").write_bytes((args.original / "batch.json").read_bytes())
    for log in sorted(args.original.glob("*/events.jsonl")):
        target = args.destination / log.parent.name
        target.mkdir()
        with (target / "events.jsonl").open("w") as out:
            for event in project_run(log.parent):
                out.write(json.dumps(event) + "\n")


if __name__ == "__main__":
    main()
