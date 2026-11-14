#!/usr/bin/env python3
"""Run recall4 variants on the frozen real-memory replay (both splits).
Usage: run_real.py VARIANT [VARIANT ...]   -> real/<variant>-<split>.jsonl"""
import json, os, pathlib, shutil, sqlite3, subprocess, sys

S = pathlib.Path(os.environ["JCODE_SCRATCH_DIR"])
R3, R = S / "recall3", S / "recall4" / "real"
R.mkdir(parents=True, exist_ok=True)
msgs = json.loads((R3 / "replay.json").read_text())
for split in ("development", "heldout"):
    p = R / f"replay-{split}.json"
    if not p.exists():
        p.write_text(json.dumps([m for m in msgs if m["split"] == split], ensure_ascii=False))

def snapshot(split, variant):
    db = R / f"db-{variant}-{split}.db"
    for ext in ("", "-wal", "-shm"):
        pathlib.Path(str(db) + ext).unlink(missing_ok=True)
    src = sqlite3.connect(f"file:{R3}/baseline-{split}.db?mode=ro", uri=True)
    dst = sqlite3.connect(db)
    src.backup(dst)
    dst.close(); src.close()
    return db

for v in sys.argv[1:]:
    for split in ("development", "heldout"):
        out = R / f"{v}-{split}.jsonl"
        if out.exists():
            print("have", out.name); continue
        db = snapshot(split, v)
        tmp = out.with_suffix(".tmp")
        with tmp.open("w") as f:
            subprocess.run(["nice", "-n", "19", str(S / "recall4/recall4"), "-db", str(db), "-input", str(R / f"replay-{split}.json"), "-variant", v],
                           stdout=f, stderr=open(R / f"{v}-{split}.log", "w"), check=True, env={**os.environ, "GOMAXPROCS": "2"})
        tmp.rename(out)
        for ext in ("", "-wal", "-shm"):
            pathlib.Path(str(db) + ext).unlink(missing_ok=True)
        print(f"JCODE_PROGRESS {json.dumps({'message': f'{v} {split} done'})}", flush=True)
