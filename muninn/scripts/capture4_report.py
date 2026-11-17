#!/usr/bin/env python3
"""Pooled comparison for ez50rb check 3: each arm's per-item value is the mean
of its two replicate runs (temperature 0 still varies on DeepSeek), compared
with a paired bootstrap over items (2000 reps). Also lists every memory the
judge or the Go matcher flags as naming an agent, per arm. Offline only.
Run in the folder holding old.jsonl, old2.jsonl, new.jsonl, new2.jsonl
(muninn/cmd/capture4 output per arm; items from capture4_sample.py, verdicts
from capture4_judge.py). See muninn/docs/CAPTURE4.md.

  capture4_report.py judged.json
"""
import collections, hashlib, json, random, sys

judged = json.load(open(sys.argv[1]))
def mkey(iid, text):
    return hashlib.sha256((iid + "\x00" + text).encode()).hexdigest()[:12]
runs = {}
for a in ["old", "old2", "new", "new2"]:
    runs[a] = {json.loads(l)["id"]: json.loads(l) for l in open(a + ".jsonl")}
def strip(r):
    return dict(r, candidates=[c for c in r["candidates"] if not c.get("agents")])
for a in ["old", "old2", "new", "new2"]:
    runs[a + "+check"] = {i: strip(r) for i, r in runs[a].items()}
arms = {"A old prompt": ["old", "old2"], "B old+post-check": ["old+check", "old2+check"], "C new prompt": ["new", "new2"], "D new+post-check": ["new+check", "new2+check"]}

def m(run, iid):
    r = runs[run][iid]
    out = collections.Counter()
    for c in r["candidates"]:
        j = judged[mkey(iid, c["content"])]
        out["mem"] += 1
        out["name"] += bool(j["agent_name"])
        out["status"] += bool(j["agent_status"])
        out["agent_any"] += bool(j["agent_name"] or j["agent_status"])
        out["durable"] += bool(j["durable"])
        out["durable_clean"] += bool(j["durable"] and not j["agent_name"] and not j["agent_status"])
        out["go"] += bool(c.get("agents"))
    return out

ok = lambda iid: all(not runs[r][iid].get("error") for r in ["old", "old2", "new", "new2"])
kinds = collections.defaultdict(list)
for iid, r in runs["old"].items():
    if ok(iid):
        kinds[r["kind"]].append(iid)

def val(arm, iid, k):
    return sum(m(r, iid)[k] for r in arms[arm]) / len(arms[arm])

def boot(a, b, ids, k, reps=2000, seed=1117):
    rng = random.Random(seed)
    pre = {i: val(a, i, k) - val(b, i, k) for i in ids}
    d = sorted(sum(pre[i] for i in rng.choices(ids, k=len(ids))) / len(ids) for _ in range(reps))
    return sum(pre.values()) / len(ids), d[int(.025 * reps)], d[int(.975 * reps)]

keys = ["mem", "name", "status", "agent_any", "durable", "durable_clean", "go"]
for kind in ["agent", "control"]:
    ids = sorted(kinds[kind])
    print(f"## {kind} items (n={len(ids)}), per item, mean of 2 runs")
    for arm in arms:
        print(f"{arm:18} " + "  ".join(f"{k} {sum(val(arm, i, k) for i in ids)/len(ids):.2f}" for k in keys))
    # noise floor: replicate vs replicate of the same prompt
    for x, y in [("old", "old2"), ("new", "new2")]:
        line = []
        for k in ["agent_any", "durable"]:
            pre = {i: m(x, i)[k] - m(y, i)[k] for i in ids}
            rng = random.Random(7); d = sorted(sum(pre[i] for i in rng.choices(ids, k=len(ids))) / len(ids) for _ in range(2000))
            line.append(f"{k} {sum(pre.values())/len(ids):+.2f} [{d[50]:+.2f}, {d[1949]:+.2f}]")
        print(f"  noise {x} vs {y}: " + "  ".join(line))
    for a, b in [("B old+post-check", "A old prompt"), ("C new prompt", "A old prompt"), ("D new+post-check", "A old prompt"), ("C new prompt", "B old+post-check")]:
        print(f"  {a} vs {b}:")
        for k in ["agent_any", "name", "status", "durable", "durable_clean", "mem"]:
            d, lo, hi = boot(a, b, ids, k)
            print(f"    {k:13} {d:+.3f}  95% [{lo:+.3f}, {hi:+.3f}]  {'CI-clear' if lo > 0 or hi < 0 else 'not clear'}")
    print()

print("## flagged memories (judge agent_name/agent_status, or Go matcher)")
for run in ["old", "old2", "new", "new2"]:
    for iid in sorted(runs[run]):
        for c in runs[run][iid]["candidates"]:
            j = judged[mkey(iid, c["content"])]
            if j["agent_name"] or j["agent_status"] or c.get("agents"):
                print(f"{run:5} {runs[run][iid]['kind']:7} {iid:7} judge name={int(j['agent_name'])} status={int(j['agent_status'])} durable={int(j['durable'])} go={','.join(c.get('agents') or []) or '-'} | {c['content'][:170]}")
