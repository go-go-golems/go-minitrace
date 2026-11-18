"""Regression probes for the ledger comparison and incremental viewer cursor."""
import argparse
from pathlib import Path
import shutil
from tempfile import TemporaryDirectory

from run import HERE, command_json, fixture, input_glob, verify
from serve import timeline


def test_ledger(binary, root):
    archives = root / "input/archives"
    archives.mkdir(parents=True)
    fixture(archives)
    work = root / "fixture"
    artifacts = work / "artifacts"
    artifacts.mkdir(parents=True)
    source = (HERE / "fixture-ledger.js").read_text()
    command = artifacts / "ledger.js"
    command.write_text(source)
    for name in ("probe_test.js", "probe-test.js"):
        (artifacts / name).write_text('function probe() { return []; } __verb__("probe", {name:"probe"});')
    try:
        command_json([str(binary), "query", "commands", "--query-repository", str(artifacts),
            "ledger", "--archive-glob", input_glob(root, "all"), "--output", "json"], work)
    except RuntimeError as error:
        assert "duplicate js verb path" in str(error), error
    else:
        raise AssertionError("Scratch command collision did not reproduce")
    original = verify(root, work, binary, "all", offline=True)
    assert original["passed"] and original["observed_rows"] == 2, original

    # Preserve the row count while replacing a genuine record with a duplicate.
    command.write_text(source.replace("return result.rows;", "return [result.rows[0], result.rows[0]];"))
    duplicate = verify(root, work, binary, "all", offline=True)
    assert duplicate["observed_rows"] == 2 and not duplicate["passed"], duplicate
    assert duplicate["unexpected_count"] == duplicate["missing_count"] == 1, duplicate

    command.write_text(source.replace("return result.rows;", "return result.rows.slice(0, 1);"))
    missing = verify(root, work, binary, "all", offline=True)
    assert not missing["passed"] and missing["missing_count"] == 1, missing

    command.write_text(source)
    narrowed = verify(root, work, binary, "codex", offline=True)
    assert narrowed["passed"] and narrowed["observed_rows"] == 1, narrowed
    assert narrowed["preview"][0]["evidence_status"] == "attempted", narrowed


def test_cursor(root):
    work = root / "events-test"
    work.mkdir()
    log = work / "events.jsonl"
    log.write_bytes(b'{"kind":"start"}\n{"kind":"')
    first = timeline(root, {})["events-test"]
    assert first["events"] == [{"kind": "start"}], first
    with log.open("ab") as handle:
        handle.write(b'finish"}\n')
    second = timeline(root, {"events-test": first["offset"]})["events-test"]
    assert second["events"] == [{"kind": "finish"}], second
    assert not timeline(root, {"events-test": second["offset"]})["events-test"]["events"]


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    args = parser.parse_args()
    with TemporaryDirectory(prefix="persona-lab-test-") as tmp:
        root = Path(tmp).resolve()
        binary = root / "go-minitrace"
        shutil.copy2(args.binary, binary)
        test_ledger(binary, root)
        test_cursor(root)
    print("Ledger contract and partial-event cursor checks passed")
