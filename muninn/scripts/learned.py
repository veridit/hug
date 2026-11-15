#!/usr/bin/env python3
"""Learned triggers: does a memory's judged history predict its relevance in
a new situation?

Every judged (message, memory) pair is a past situation where the memory was
shown and was relevant (+) or not (-). For a new message, a memory's learned
score is how similar the new message is to the past messages where it helped,
minus how similar it is to those where it did not.

Leakage control: sessions are ordered by first message time and folded by
session. For a test message only judged pairs from strictly EARLIER sessions
count as history. The test message's own session never contributes.

Candidates are recall3's union pools (top 20 ungated per message). Output:
for each arm, relevant and irrelevant memories injected per message (top 5,
session de-duplicated), using the same blind judgments.

Message similarity = cosine of nomic embeddings (cached, local Ollama)."""
import collections, hashlib, json, math, os, pathlib, re, struct, sys, urllib.request

S = pathlib.Path(os.environ["JCODE_SCRATCH_DIR"])
R3, R = S / "recall3", S / "recall4"
msgs = {m["id"]: m for m in json.loads((R3 / "replay.json").read_text())}
J = json.loads((R / "real" / "judgments.json").read_text())
key = lambda i, t: hashlib.sha256((str(i) + "\0" + t).encode()).hexdigest()[:20]
QV = R / "real" / "msgvec"
QV.mkdir(exist_ok=True)

def vec(text):
    text = text[:8000]
    h = hashlib.sha256(("nomic-embed-text\0" + text).encode()).hexdigest()
    p = QV / h
    if not p.exists():
        req = urllib.request.Request("http://localhost:11434/api/embed", data=json.dumps({"model": "nomic-embed-text", "input": [text]}).encode())
        v = json.load(urllib.request.urlopen(req, timeout=120))["embeddings"][0]
        n = math.sqrt(sum(x * x for x in v)) or 1
        p.write_bytes(struct.pack(f"<{len(v)}f", *[x / n for x in v]))
    b = p.read_bytes()
    return struct.unpack(f"<{len(b)//4}f", b)

def cos(a, b):
    return sum(x * y for x, y in zip(a, b))

# Pools: union candidates with their recall features; baseline: shipped recall.
pools, base = {}, {}
for s in ("development", "heldout"):
    for l in open(R3 / f"union-{s}.jsonl"):
        r = json.loads(l)
        c = []
        for p, h in enumerate(r["response"].get("hits") or []):
            w = h.get("why", "")
            sem = float(re.search(r"semantic=([\d.]+)", w).group(1)) if "semantic=" in w else 0.0
            j = J.get(key(r["id"], h["text"]))
            c.append({"text": h["text"], "rank": p, "sem": sem, "own": msgs[r["id"]]["instance"].lower() in h["scope"].lower(),
                      "rel": None if j is None else bool(j["relevant"])})
        pools[r["id"]] = c
    for l in open(R3 / f"baseline-{s}.jsonl"):
        r = json.loads(l)
        base[r["id"]] = [{"text": h["text"], "rel": (J.get(key(r["id"], h["text"])) or {}).get("relevant")} for h in r["response"].get("hits") or []]

# History: for every judged pair, the memory text, the message, and the label.
# Judged pairs include pairs from every recall3/recall4 arm, not only the pool.
history = collections.defaultdict(list)  # memory text -> [(msg id, rel)]
for s in ("development", "heldout"):
    for f in list(R3.glob(f"*-{s}.jsonl")) + list((R / "real").glob(f"*-{s}.jsonl")):
        if f.name.startswith(("longmem",)):
            continue
        for l in f.read_text().splitlines():
            try:
                r = json.loads(l)
            except Exception:
                continue
            if r.get("id") not in msgs:
                continue
            for h in r["response"].get("hits") or []:
                j = J.get(key(r["id"], h["text"]))
                if j is not None:
                    history[h["text"]].append((r["id"], bool(j["relevant"])))
for t in history:
    history[t] = sorted(set(history[t]))

start = {}
for m in msgs.values():
    start[m["session"]] = min(start.get(m["session"], m["timestamp"]), m["timestamp"])
order = sorted(start, key=start.get)
rank_of = {s: n for n, s in enumerate(order)}

ids = sorted(msgs)
print(f"messages {len(ids)}, sessions {len(order)}, memories with history {len(history)}, judged pairs {sum(map(len, history.values()))}", flush=True)
V = {}
for n, i in enumerate(ids):
    V[i] = vec(msgs[i]["message"])
    if n % 50 == 0:
        print(f"JCODE_PROGRESS {json.dumps({'current': n, 'total': len(ids), 'unit': 'message vectors'})}", flush=True)

def learned(i, text, k=3):
    """Earlier-session evidence for this memory: mean of the k most similar
    positive past messages minus the same for negatives (0 if none)."""
    me = rank_of[msgs[i]["session"]]
    pos, neg = [], []
    for j, rel in history.get(text, []):
        if rank_of[msgs[j]["session"]] >= me:
            continue
        (pos if rel else neg).append(cos(V[i], V[j]))
    top = lambda xs: sum(sorted(xs, reverse=True)[:k]) / min(k, len(xs)) if xs else 0.0
    return top(pos), top(neg), len(pos), len(neg)

def run(pick, sub):
    rel = irr = unk = empty = 0
    seen = collections.defaultdict(set)
    for i in sorted(sub, key=lambda i: msgs[i]["timestamp"]):
        s = msgs[i]["session"]
        got = [c for c in pick(i) if c["text"] not in seen[s]][:5]
        for c in got:
            seen[s].add(c["text"])
        empty += not got
        for c in got:
            if c["rel"] is None:
                unk += 1
            elif c["rel"]:
                rel += 1
            else:
                irr += 1
    n = len(sub)
    return rel / n, irr / n, unk / n, empty / n

def rule(i, ms=0.64):
    return [c for c in sorted(pools[i], key=lambda c: -c["sem"]) if c["sem"] >= ms and len(c["text"]) <= 2000 and c["own"]]

def with_learning(i, ms=0.64, veto=0.05, boost=0.03, rescue=0.80):
    """rule + learned triggers: drop a candidate whose closest earlier
    negative situation is clearly closer than its closest positive one
    (veto); admit a below-threshold candidate when an earlier positive
    situation is very similar (rescue); order by sem + boost * evidence."""
    out = []
    for c in pools[i]:
        p, q, np_, nq = learned(i, c["text"])
        if nq and q - p > veto:
            continue
        ok = c["sem"] >= ms and len(c["text"]) <= 2000 and c["own"]
        if not ok and np_ and p >= rescue and c["own"]:
            ok = True
        if ok:
            out.append((c["sem"] + boost * (p - q), c))
    return [c for _, c in sorted(out, key=lambda x: -x[0])]

def coverage(sub):
    with_hist = sum(1 for i in sub for c in pools[i] if any(rank_of[msgs[j]["session"]] < rank_of[msgs[i]["session"]] for j, _ in history.get(c["text"], [])))
    return with_hist / max(1, sum(len(pools[i]) for i in sub))

lines = ["# Learned triggers (recall4)", "", "Earlier-session judged history only; same blind judgments. Top 5 per message.", ""]
for st in ("human", "long"):
    sub = [i for i in ids if msgs[i]["stratum"] == st]
    lines += [f"## {st} (n={len(sub)}); candidates with earlier-session history: {coverage(sub):.0%}", "",
              "| arm | rel/msg | irr/msg | unjudged/msg | empty |", "|---|---:|---:|---:|---:|"]
    arms = [("current recall", lambda i: base[i]), ("rule sem>=0.64", rule)]
    for veto in (0.0, 0.05):
        for rescue in (0.80, 0.85, 1.1):
            arms.append((f"rule + learned veto>{veto} rescue>={rescue if rescue < 1 else 'off'}", lambda i, v=veto, r=rescue: with_learning(i, veto=v, rescue=r)))
    for name, f in arms:
        a, b, u, e = run(f, sub)
        lines.append(f"| {name} | {a:.2f} | {b:.2f} | {u:.2f} | {e:.2f} |")
    lines.append("")
text = "\n".join(lines)
(R / "real" / "report-learned.md").write_text(text)
print(text)

def per_msg(pick, sub):
    out = {}
    seen = collections.defaultdict(set)
    for i in sorted(sub, key=lambda i: msgs[i]["timestamp"]):
        s = msgs[i]["session"]
        got = [c for c in pick(i) if c["text"] not in seen[s]][:5]
        for c in got:
            seen[s].add(c["text"])
        out[i] = (sum(c["rel"] is True for c in got), sum(c["rel"] is False for c in got))
    return out

import random
def boot(a, b, sub, j, reps=2000):
    sess = collections.defaultdict(list)
    for i in sub:
        sess[msgs[i]["session"]].append(i)
    cl = list(sess); rng = random.Random(813); d = []
    for _ in range(reps):
        smp = [i for s in rng.choices(cl, k=len(cl)) for i in sess[s]]
        d.append(sum(a[i][j] - b[i][j] for i in smp) / len(smp))
    d.sort(); return d[50], d[1949]

print("\n## paired deltas, 95% session-cluster CI")
for st in ("human", "long"):
    sub = [i for i in ids if msgs[i]["stratum"] == st]
    cur = per_msg(lambda i: base[i], sub); ru = per_msg(rule, sub)
    le = per_msg(lambda i: with_learning(i, veto=0.05, rescue=1.1), sub)
    for name, x, y in (("learned vs current", le, cur), ("learned vs rule", le, ru), ("rule vs current", ru, cur)):
        for j, lab in ((0, "rel"), (1, "irr")):
            lo, hi = boot(x, y, sub, j)
            print(f"{st:5s} {name:20s} {lab}: {sum(x[i][j]-y[i][j] for i in sub)/len(sub):+.2f} [{lo:+.2f}, {hi:+.2f}]")
