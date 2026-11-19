#!/usr/bin/env python3
"""Build a fresh confirmation set: user messages from jcode sessions that
started after the recall3 freeze (2026-11-06T10:13Z), none of which any
recall4 setting was tuned on.

Strata follow recall3: 'human' = a top-level session's typed message of at
most 600 bytes that is not a harness notice; everything else 'long'.
Writes real/fresh/replay.json (same fields recall4 reads) and inventory.json.
Usage: build_fresh.py [--per-session N] (default 4 per session per stratum)"""
import collections, datetime, glob, json, os, pathlib, random, sys

S = pathlib.Path(os.environ["JCODE_SCRATCH_DIR"])
OUT = S / "recall4" / "real" / "fresh"
OUT.mkdir(parents=True, exist_ok=True, mode=0o700)
CUT = datetime.datetime.fromisoformat("2026-11-06T10:13:13.988164+00:00")
homes = json.loads((S / "recall3" / "homes.json").read_text())
per = int(sys.argv[sys.argv.index("--per-session") + 1]) if "--per-session" in sys.argv else 4
per_human = 25  # typed messages are scarce; long ones are capped harder
NOTICE = ("**Background task**", "🐝 **Swarm await", "[Scheduled task]", "[auto] ", "<system-reminder>",
          # jcode's image notices in the user role, the same exact prefixes as
          # recall's harnessNotices (w4qmr1): not typed by a person.
          "[Older tool screenshot omitted from history to bound transcript size: ",
          "[Attached image associated with the preceding tool result: ",
          "[Image omitted during request-size recovery: ",
          "[Image omitted during emergency context recovery: ",
          "[Image omitted: the provider rejected an image in this conversation ",
          "[Image omitted: this provider/model does not support image input; ")
MACHINE = ("DM from", "Cross-swarm DM", "You are the engineer", "You are a worker", "You are an engineer", "You are a mechanic", "[transcription]")

def ts(s):
    return datetime.datetime.fromisoformat(s.replace("Z", "+00:00"))

pool = []
for inst, home in homes.items():
    for f in glob.glob(os.path.join(home, "sessions", "session_*.json")):
        try:
            d = json.load(open(f))
        except Exception:
            continue
        if not d.get("created_at") or ts(d["created_at"]) <= CUT:
            continue
        if d.get("is_debug") or "/scratch/" in (d.get("working_dir") or ""):
            continue
        child = bool(d.get("parent_id"))
        for m in d.get("messages", []):
            if m.get("role") != "user" or m.get("display_role") is not None or not isinstance(m.get("content"), list):
                continue
            text = "".join(b.get("text", "") for b in m["content"] if b.get("type") == "text").strip()
            if not text or text.startswith(NOTICE) or (text.startswith("You have ") and " incomplete todo" in text):
                continue
            if "-- AI memory (muninn):" in text or "-- muninn recall:" in text:
                continue
            machine = child or text.startswith(MACHINE)
            stratum = "long" if machine or len(text.encode()) > 600 else "human"
            pool.append({"instance": inst, "home": home, "session": d["id"], "cwd": d.get("working_dir") or home,
                         "message": text, "timestamp": m.get("timestamp") or d["created_at"], "stratum": stratum})

# Same-text duplicates (pasted twice, resent DMs) count once.
seen, uniq = set(), []
for m in sorted(pool, key=lambda x: x["timestamp"]):
    k = (m["session"], m["message"])
    if k not in seen:
        seen.add(k); uniq.append(m)

# Cap per session and stratum so one long session cannot dominate.
rng = random.Random(20261114)
by = collections.defaultdict(list)
for m in uniq:
    by[(m["session"], m["stratum"])].append(m)
chosen = []
for k in sorted(by):
    ms = by[k]
    cap = per_human if k[1] == "human" else per
    chosen += sorted(rng.sample(ms, min(cap, len(ms))), key=lambda x: x["timestamp"])
chosen.sort(key=lambda x: x["timestamp"])
for i, m in enumerate(chosen):
    m["id"] = 100000 + i
    m["split"] = "fresh"
(OUT / "replay.json").write_text(json.dumps(chosen, ensure_ascii=False))
inv = {"cut": CUT.isoformat(), "candidates": len(uniq), "chosen": len(chosen),
       "by_stratum": collections.Counter(m["stratum"] for m in chosen),
       "human_sessions": len({m["session"] for m in chosen if m["stratum"] == "human"}),
       "by_instance": collections.Counter(m["instance"] for m in chosen),
       "first": chosen[0]["timestamp"] if chosen else None, "last": chosen[-1]["timestamp"] if chosen else None}
(OUT / "inventory.json").write_text(json.dumps(inv, indent=2))
print(json.dumps(inv, indent=2))
