"""One-shot private preservation. Does not forget jobs or change service settings."""
import base64
import contextlib
import datetime
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import sqlite3
import stat
import uuid

EXPECTED_STORE = "dfbd69b42df2453aae7bfa1c5a7af704"
BATCH_SIZE = 32
os.umask(0o077)


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def private_dir(path):
    info = path.lstat()
    require(path.resolve(strict=True) == path, "Symlinked directory refused")
    require(stat.S_ISDIR(info.st_mode) and info.st_uid == os.getuid(), "Invalid directory ownership/type")
    require(stat.S_IMODE(info.st_mode) == 0o700, "Directory is not private")


def digest_file(path):
    result = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            result.update(chunk)
    return result.hexdigest()


def persist_json(path, data):
    encoded = (json.dumps(data, indent=2, sort_keys=True) + "\n").encode("utf-8")
    with path.open("xb") as handle:
        handle.write(encoded)
        handle.flush()
        os.fsync(handle.fileno())
    require(path.read_bytes() == encoded, "JSON readback mismatch")


def main():
    state = Path.home() / "Library/Application Support/EkoWriteBridge/commands"
    private_dir(state)
    db_path = state / "commands.sqlite3"
    info = db_path.lstat()
    require(stat.S_ISREG(info.st_mode) and info.st_uid == os.getuid() and stat.S_IMODE(info.st_mode) == 0o600, "Invalid source database")
    source = sqlite3.connect(db_path.as_uri() + "?mode=ro", uri=True, timeout=5)
    source.row_factory = sqlite3.Row
    source.execute("PRAGMA query_only=ON")
    require(source.execute("SELECT value FROM metadata WHERE key='store_id'").fetchone()[0] == EXPECTED_STORE, "Unexpected execution store")
    require(source.execute("PRAGMA quick_check").fetchall()[0][0] == "ok", "Source database failed integrity check")
    selected = source.execute("SELECT * FROM jobs WHERE state IN ('exited','timed_out','cancelled') AND output_complete=1 AND finished IS NOT NULL ORDER BY created,job_id LIMIT ?", (BATCH_SIZE,)).fetchall()
    require(len(selected) == BATCH_SIZE, "Insufficient complete terminal jobs; no retirement authorised by this script")
    required_bytes = sum(row["stdout_bytes"] + row["stderr_bytes"] for row in selected) + db_path.stat().st_size
    require(shutil.disk_usage(state).free > required_bytes + 128 * 1024 * 1024, "Insufficient backup disk space")
    maintenance = state.parent / "maintenance"
    maintenance.mkdir(mode=0o700, exist_ok=True)
    private_dir(maintenance)
    archive = maintenance / ("20260920-" + uuid.uuid4().hex)
    archive.mkdir(mode=0o700)
    private_dir(archive)
    entries = []
    with contextlib.ExitStack() as stack:
        for row in selected:
            identity = row["job_id"]
            require(re.fullmatch(r"[a-f0-9]{32}", identity) is not None, "Invalid job identity")
            job_dir = state / identity
            private_dir(job_dir)
            require({p.name for p in job_dir.iterdir()} <= {"supervisor.lock", "stdout", "stderr"}, "Unexpected job-directory material")
            fd = os.open(job_dir / "supervisor.lock", os.O_RDWR | os.O_NOFOLLOW)
            stack.callback(os.close, fd)
            fcntl.flock(fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
        snapshot = archive / "commands.sqlite3"
        with sqlite3.connect(snapshot) as destination:
            source.backup(destination)
        with snapshot.open("rb") as handle:
            os.fsync(handle.fileno())
        copied = sqlite3.connect(snapshot.as_uri() + "?mode=ro", uri=True)
        copied.row_factory = sqlite3.Row
        stack.callback(copied.close)
        require([r[0] for r in copied.execute("PRAGMA integrity_check")] == ["ok"], "Backup database failed integrity check")
        require(copied.execute("SELECT value FROM metadata WHERE key='store_id'").fetchone()[0] == EXPECTED_STORE, "Backup store identity mismatch")
        for selected_row in selected:
            identity = selected_row["job_id"]
            row = dict(source.execute("SELECT * FROM jobs WHERE job_id=?", (identity,)).fetchone())
            saved = dict(copied.execute("SELECT * FROM jobs WHERE job_id=?", (identity,)).fetchone())
            require(row == saved == dict(selected_row), "Selected job changed during backup")
            require(row["state"] in ("exited", "timed_out", "cancelled") and row["output_complete"] == 1, "Job no longer eligible")
            spec = json.loads(row["spec"])
            fingerprint = hashlib.sha256(json.dumps(spec, sort_keys=True, ensure_ascii=True, separators=(",", ":")).encode()).hexdigest()
            require(fingerprint == row["request_hash"], "Command fingerprint mismatch")
            inputs = [tuple(r) for r in source.execute("SELECT * FROM inputs WHERE job_id=? ORDER BY seq", (identity,))]
            saved_inputs = [tuple(r) for r in copied.execute("SELECT * FROM inputs WHERE job_id=? ORDER BY seq", (identity,))]
            require(inputs == saved_inputs, "Input archive mismatch")
            destination_dir = archive / identity
            destination_dir.mkdir(mode=0o700)
            entry = {"job_id": identity, "request_id": row["request_id"], "request_hash": row["request_hash"], "original_state": row["state"], "exit_code": row["exit_code"], "input_count": len(inputs), "streams": {}}
            for stream in ("stdout", "stderr"):
                original = state / identity / stream
                with os.fdopen(os.open(original, os.O_RDONLY | os.O_NOFOLLOW), "rb") as reader:
                    before = os.fstat(reader.fileno())
                    require(stat.S_ISREG(before.st_mode) and before.st_uid == os.getuid(), "Invalid output file")
                    require(before.st_size == row[stream + "_bytes"], "Output length does not match completed record")
                    target = destination_dir / stream
                    source_hash = hashlib.sha256()
                    total = 0
                    with target.open("xb") as writer:
                        while True:
                            chunk = reader.read(1024 * 1024)
                            if not chunk:
                                break
                            source_hash.update(chunk)
                            writer.write(chunk)
                            total += len(chunk)
                        writer.flush()
                        os.fsync(writer.fileno())
                    after = os.fstat(reader.fileno())
                    require((before.st_ino, before.st_size, before.st_mtime_ns) == (after.st_ino, after.st_size, after.st_mtime_ns), "Output changed during copy")
                require(total == row[stream + "_bytes"] and digest_file(target) == source_hash.hexdigest(), "Output readback mismatch")
                entry["streams"][stream] = {"bytes": total, "sha256": source_hash.hexdigest()}
            entries.append(entry)
        counts = dict(source.execute("SELECT state,count(*) FROM jobs GROUP BY state").fetchall())
        manifest = {"record_type": "private_bridge_command_batch_backup", "at_brisbane": datetime.datetime.now(datetime.timezone(datetime.timedelta(hours=10))).isoformat(), "store_id": EXPECTED_STORE, "source": str(state), "archive": str(archive), "batch_size": len(entries), "database_sha256": digest_file(snapshot), "database_integrity_check": "ok", "job_rows_and_inputs_match": True, "output_readbacks_match": True, "source_counts_at_backup": counts, "entries": entries, "retired_jobs": [], "restore_warning": "Read this snapshot as historical evidence only. It includes other job rows whose output is not copied, including the active backup command. Do not replace the live store or replay commands."}
        persist_json(archive / "MANIFEST.json", manifest)
        for path in [archive / e["job_id"] for e in entries] + [archive, maintenance]:
            fd = os.open(path, os.O_RDONLY)
            try:
                os.fsync(fd)
            finally:
                os.close(fd)
    source.close()
    summary = {"archive": str(archive), "manifest_sha256": digest_file(archive / "MANIFEST.json"), "store_id": EXPECTED_STORE, "batch_size": len(entries), "verified_output_bytes": sum(s["bytes"] for e in entries for s in e["streams"].values()), "jobs": [{"job_id": e["job_id"], "request_hash": e["request_hash"]} for e in entries], "counts": counts, "retirement_performed": False, "credentials_read": False, "service_or_config_changed": False}
    persist_json(Path(__file__).with_name("2026-09-20_bridge_batch_backup.json"), summary)
    print(json.dumps(summary, sort_keys=True))


if __name__ == "__main__":
    main()
