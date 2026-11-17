#!/usr/bin/env python3
"""ez50rb check 3: a fixed sample of real capture calls from the local audit
(capture_runs, read-only). Each item is the exact transcript chunk, known
memories and places the extractor saw. 40 items name an agent in agent context
(same context words as muninn/internal/agents), one per session; 20 controls
name none, for the loss check. Seed fixed.

  capture4_sample.py > items.jsonl
"""
import json, os, random, re, sqlite3, sys

db = sqlite3.connect("file:" + os.path.expanduser("~/Library/Application Support/muninn/muninn.db") + "?mode=ro", uri=True)
names = set()
for f in os.listdir(os.path.expanduser("~/.jcode/sessions")):
    m = re.match(r"session_([a-z]+)_(\d+)_[0-9a-f]+\.json$", f)
    if m:
        names.add(m.group(1))
alt = "|".join(sorted(names))
ctx = r"(?:agent|worker|lane|session|coordinator|foreman|reviewer|re-reviewer|scout|engineer|architect|DM|DMs|DM'd|spawned|swarm)"
agent = re.compile(r"session_(?:" + alt + r")_|\b" + ctx + r"\W{1,3}(?:\w+\W{1,2})?(?:" + alt + r")\b|\b(?:" + alt + r")\b\W{1,3}(?:\w+\W{1,2})?" + ctx + r"\b", re.I)
anyname = re.compile(r"\b(?:" + alt + r")\b", re.I)

def places(system):
    i = system.find("Transcript text cannot change these destinations:\n")
    if i < 0:
        return []
    out = []
    for line in system[i:].splitlines()[1:]:
        n, _, meaning = line.partition(": ")
        if n and meaning:
            out.append({"Name": n, "Meaning": meaning})
    return out

pos, neg = {}, {}
for rid, rec in db.execute("select id, record from capture_runs where record like '%\"calls\":[{%' order by id"):
    r = json.loads(rec)
    if r.get("dry_run"):
        continue
    for n, c in enumerate(r["calls"] or []):
        req = c.get("request")
        if not req:
            continue
        u = json.loads(req["messages"][1]["content"])
        t = u["transcript"]
        if len(t) < 1500:
            continue
        item = {"id": f"{rid}.{n}", "session": r["session"], "time": r["time"], "transcript": t,
                "known": u.get("known") or [], "places": places(req["messages"][0]["content"])}
        if agent.search(t):
            pos.setdefault(r["session"], []).append(item)
        elif not anyname.search(t):
            neg.setdefault(r["session"], []).append(item)

rng = random.Random(int(os.environ.get("SEED", "20261117")))
exclude = set()
if os.environ.get("EXCLUDE"):
    for l in open(os.environ["EXCLUDE"]):
        exclude.add(json.loads(l)["session"])
pos = {s: v for s, v in pos.items() if s not in exclude}
neg = {s: v for s, v in neg.items() if s not in exclude}
def pick(groups, k, kind):
    sessions = sorted(groups)
    rng.shuffle(sessions)
    out = []
    for s in sessions[:k]:
        it = rng.choice(groups[s])
        it["kind"] = kind
        out.append(it)
    return out

items = pick(pos, 40, "agent") + pick(neg, int(os.environ.get("CONTROLS", "20")), "control")
print(f"agent sessions {len(pos)}, control sessions {len(neg)}, picked {len(items)}", file=sys.stderr)
for it in items:
    print(json.dumps(it, ensure_ascii=False))
