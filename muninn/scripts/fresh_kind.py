#!/usr/bin/env python3
"""Fresh-set check of the kind rule (the confirmation test for jrj1vq step 2).

Arms on the 147 post-freeze messages (real/fresh/replay.json):
  shipped   today's recall (v0.7.2, BM25), real/default-fresh.jsonl
  kindrule  from the top-20 ungated pool (real/union-fresh.jsonl): own
            project, under 2000 chars, vector >= 0.60, CASE agrees, and the
            extracted kind is not 'record'; top 5 by vector, session dedup.
Every pick is judged blind; extraction only sees the memory.
Paired session-cluster bootstrap (2000 reps), per stratum."""
import collections, hashlib, json, os, pathlib, random, re, subprocess, sys

S = pathlib.Path(os.environ["JCODE_SCRATCH_DIR"]); R = S / "recall4" / "real"
msgs = {m["id"]: m for m in json.loads((R / "fresh" / "replay.json").read_text())}
mk = lambda t: hashlib.sha256(t.encode()).hexdigest()[:16]
PROJECTS = {"brovs": "brovs", "seid": "seid", "statbus": "statbus", "global": ""}

# 1. extract kind/case for any memory in the fresh pool that lacks it
subprocess.run([sys.executable, str(S / "recall4" / "extract_whenthen.py")], check=True, stdout=subprocess.DEVNULL)
WT = json.loads((R / "whenthen.json").read_text())

def load(v):
    out = {}
    for l in open(R / f"{v}-fresh.jsonl"):
        r = json.loads(l)
        out[r["id"]] = r["response"].get("hits") or []
    return out
shipped, pool = load("default"), load("union")

def kindrule(i):
    proj = PROJECTS.get(msgs[i]["instance"], msgs[i]["instance"].lower())
    picks = []
    for h in pool[i]:
        w = h.get("why", "")
        sem = float(re.search(r"semantic=([\d.]+)", w).group(1)) if "semantic=" in w else 0.0
        wt = WT.get(mk(h["text"])) or {}
        projs = [p.lower() for p in (wt.get("case") or {}).get("projects", [])]
        case_ok = not projs or not proj or any(proj in p or p in proj for p in projs)
        own = msgs[i]["instance"].lower() in h["scope"].lower()
        if own and len(h["text"]) <= 2000 and sem >= 0.60 and case_ok and wt.get("kind") != "record":
            picks.append((sem, h))
    return [h for _, h in sorted(picks, key=lambda x: -x[0])]

def dedup(pick):
    out, seen = {}, collections.defaultdict(set)
    for i in sorted(msgs, key=lambda i: msgs[i]["timestamp"]):
        s = msgs[i]["session"]
        got = [h for h in pick(i) if h["text"] not in seen[s]][:5]
        for h in got:
            seen[s].add(h["text"])
        out[i] = [h["text"] for h in got]
    return out
arms = {"shipped": dedup(lambda i: shipped[i]), "kindrule": dedup(kindrule)}

# 2. write arms in the jsonl shape judge_real.py reads, then judge
for name, rows in arms.items():
    if name == "shipped":
        continue
    with open(R / f"{name}-fresh.jsonl", "w") as f:
        for i, texts in rows.items():
            f.write(json.dumps({"id": i, "split": "fresh", "response": {"hits": [{"text": t} for t in texts]}}) + "\n")
subprocess.run([sys.executable, str(S / "recall4" / "judge_real.py"), "--variants=default,kindrule", "--only=fresh"], check=True, cwd=S / "recall4")
print((R / "report-fresh.md").read_text())
