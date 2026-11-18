#!/usr/bin/env python3
"""Tool-call replay set for tool-call recall (huginn task n7jbmx).

Situations are the agent's own tool calls from jcode sessions in a date
window, as jcode's post_tool_feedback hook would see them: the tool, the
command (bash) and the touched paths (absolute, as JCODE_HOOK_TOUCHED_PATHS
gives them). "message" is the same call rendered as text, for the
message-recall arms (the old "default" baseline, muninn/docs/RECALL-TOOL.md).

Sampling: stratified by tool group (bash, edit, read) with fixed quotas, at
most --per-session calls per session per group, seeded. recall4 runs it with
-asof, so each call sees only memories written before it.

Usage: build_tool_set.py --from 2026-11-16 --to 2026-11-19 --n 300 \\
         --seed 1118 --out $JCODE_SCRATCH_DIR/recall4/n7jbmx/dev/replay.json
--to is exclusive; a bound is a date or an RFC 3339 time. Homes: $JCODE_SCRATCH_DIR/recall3/homes.json."""
import argparse, collections, datetime, glob, json, os, pathlib, random, re

ap = argparse.ArgumentParser()
ap.add_argument("--from", dest="start", required=True)
ap.add_argument("--to", dest="end", required=True)
ap.add_argument("--n", type=int, default=300)
ap.add_argument("--seed", type=int, default=1118)
ap.add_argument("--per-session", type=int, default=6)
ap.add_argument("--out", required=True)
ap.add_argument("--first-id", type=int, default=300000)
args = ap.parse_args()

S = pathlib.Path(os.environ["JCODE_SCRATCH_DIR"])
homes = json.loads((S / "recall3" / "homes.json").read_text())
ts = lambda s: datetime.datetime.fromisoformat(s.replace("Z", "+00:00"))
# A bound is a date (midnight UTC) or a time, e.g. the snapshot's freeze.
bound = lambda s: ts(s) if "T" in s else ts(s + "T00:00:00Z")
start, end = bound(args.start), bound(args.end)
GROUP = {"bash": "bash", "edit": "edit", "write": "edit", "apply_patch": "edit", "multiedit": "edit", "read": "read"}
QUOTA = {"bash": 0.5, "edit": 0.3, "read": 0.2}
patch_file = re.compile(r"^\*\*\* (?:Update|Add|Delete) File: (.+)$|^\+\+\+ b/(.+)$", re.M)

def touched(name, inp, cwd):
    ps = [inp[k] for k in ("file_path", "path") if isinstance(inp.get(k), str)]
    if isinstance(inp.get("paths"), list):
        ps += [p for p in inp["paths"] if isinstance(p, str)]
    if name == "apply_patch":
        ps += [a or b for a, b in patch_file.findall(str(inp.get("patch_text") or inp.get("patch") or ""))]
    out = []
    for p in ps:
        p = os.path.expanduser(p.strip())
        out.append(p if os.path.isabs(p) else os.path.normpath(os.path.join(cwd, p)))
    return out

def render(name, cmd, paths, inp):
    if name == "bash":
        return f"bash {cmd[:400]}"
    first = str(inp.get("new_string") or inp.get("content") or "").strip().splitlines()
    return f"{name} {' '.join(paths)} {first[0][:200] if first else ''}".strip()

def messages(f):
    try:
        d = json.load(open(f))
    except Exception:
        return None, []
    msgs, seen = list(d.get("messages") or []), set()
    seen = {m.get("id") for m in msgs}
    j = f[:-5] + ".journal.jsonl"
    if os.path.exists(j):
        for line in open(j):
            try:
                rec = json.loads(line)
            except Exception:
                continue
            for m in rec.get("append_messages") or []:
                if m.get("id") not in seen:
                    seen.add(m.get("id")); msgs.append(m)
    return d, msgs

pool = collections.defaultdict(list)  # (group, session) -> calls
for inst, home in homes.items():
    for f in glob.glob(os.path.join(home, "sessions", "session_*.json")):
        d, msgs = messages(f)
        # Swarm workers carry is_debug but are real work, and the hook runs
        # in them: keep them. A debug session without a parent is a test.
        if not d or d.get("is_debug") and not d.get("parent_id") or "/scratch/recall" in (d.get("working_dir") or ""):
            continue
        cwd = d.get("working_dir") or os.path.dirname(home)
        for m in msgs:
            if m.get("role") != "assistant" or not isinstance(m.get("content"), list) or not m.get("timestamp"):
                continue
            at = ts(m["timestamp"])
            if not (start <= at < end):
                continue
            for b in m["content"]:
                if b.get("type") != "tool_use" or b.get("name") not in GROUP:
                    continue
                inp = b.get("input") or {}
                cmd = str(inp.get("command", "")).strip() if b["name"] == "bash" else ""
                paths = touched(b["name"], inp, cwd)
                if b["name"] == "bash" and len(cmd) < 4 or b["name"] != "bash" and not paths:
                    continue
                g = GROUP[b["name"]]
                pool[(g, d["id"])].append({"instance": inst, "home": home, "session": d["id"], "cwd": cwd, "tool": b["name"],
                                            "command": cmd[:4000], "paths": paths, "message": render(b["name"], cmd, paths, inp),
                                            "timestamp": m["timestamp"], "stratum": g})
rng = random.Random(args.seed)
chosen = []
for g, share in QUOTA.items():
    keys = sorted(k for k in pool if k[0] == g)
    per = [rng.sample(pool[k], min(args.per_session, len(pool[k]))) for k in keys]
    flat = [c for cs in per for c in cs]
    rng.shuffle(flat)
    chosen += flat[: round(args.n * share)]
chosen.sort(key=lambda x: x["timestamp"])
for i, m in enumerate(chosen):
    m["id"], m["split"] = args.first_id + i, "tool"
out = pathlib.Path(args.out)
out.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
out.write_text(json.dumps(chosen, ensure_ascii=False))
inv = {"window": [args.start, args.end], "seed": args.seed, "chosen": len(chosen),
       "sessions": len({m["session"] for m in chosen}),
       "available": {g: sum(len(v) for k, v in pool.items() if k[0] == g) for g in QUOTA},
       "by_tool": collections.Counter(m["tool"] for m in chosen), "by_instance": collections.Counter(m["instance"] for m in chosen)}
(out.parent / "inventory.json").write_text(json.dumps(inv, indent=2))
print(json.dumps(inv, indent=2))
