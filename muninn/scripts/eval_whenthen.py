#!/usr/bin/env python3
"""Does matching the situation against a memory's WHEN (and WHEN NOT) pick
relevant memories better than matching against the memory text?

Candidates: recall3's union pools (top 20 ungated per real message).
Labels: the same blind judgments. Extraction never saw messages or labels.

Per candidate, local and free signals:
  sem_text   cosine(message, memory text)  (stored in the pool: today's signal)
  sem_when   cosine(message, WHEN)
  sem_not    cosine(message, WHEN NOT)
  bm_when    BM25 of message words in WHEN, per pool (FTS5)
  case_ok    1 if the CASE projects are empty or include the message's project
  kind       record | fact | rule | preference

Reported:
  AUC (pairwise: P(relevant candidate scores above an irrelevant one in the
  same message)) for each signal, by stratum.
  Top-5 picks per arm: relevant and irrelevant per message, session dedup.
  Paired session-cluster CIs against shipped recall and the text-only rule."""
import collections, hashlib, json, math, os, pathlib, random, re, sqlite3, struct, sys, urllib.request

S = pathlib.Path(os.environ["JCODE_SCRATCH_DIR"])
R3, R = S / "recall3", S / "recall4" / "real"
msgs = {m["id"]: m for m in json.loads((R3 / "replay.json").read_text())}
J = json.loads((R / "judgments.json").read_text())
WT = json.loads((R / "whenthen.json").read_text())
jk = lambda i, t: hashlib.sha256((str(i) + "\0" + t).encode()).hexdigest()[:20]
mk = lambda t: hashlib.sha256(t.encode()).hexdigest()[:16]
VD = R / "msgvec"; VD.mkdir(exist_ok=True)

def vec(text):
    text = (text or "")[:8000]
    if not text.strip():
        return None
    h = hashlib.sha256(("nomic-embed-text\0" + text).encode()).hexdigest()
    p = VD / h
    if not p.exists():
        req = urllib.request.Request("http://localhost:11434/api/embed", data=json.dumps({"model": "nomic-embed-text", "input": [text]}).encode())
        v = json.load(urllib.request.urlopen(req, timeout=120))["embeddings"][0]
        n = math.sqrt(sum(x * x for x in v)) or 1
        p.write_bytes(struct.pack(f"<{len(v)}f", *[x / n for x in v]))
    b = p.read_bytes()
    return struct.unpack(f"<{len(b)//4}f", b)

cos = lambda a, b: sum(x * y for x, y in zip(a, b)) if a and b else 0.0
STOP = set("the a an and or to of in is it for on with this that be are was i you we what how when do does can my me our".split())
words = lambda t: [w for w in re.findall(r"[a-z0-9][a-z0-9_.-]*", t.lower()) if len(w) > 1 and w not in STOP]

PROJECTS = {"brovs": "brovs", "seid": "seid", "statbus": "statbus", "global": "", "GenderedGames": "genderedgames", "truth_tribe_and_treasure": "truth"}

pools, base = {}, {}
for s in ("development", "heldout"):
    for l in open(R3 / f"union-{s}.jsonl"):
        r = json.loads(l); i = r["id"]; m = msgs[i]
        c = []
        for h in r["response"].get("hits") or []:
            w = h.get("why", "")
            sem = float(re.search(r"semantic=([\d.]+)", w).group(1)) if "semantic=" in w else 0.0
            j = J.get(jk(i, h["text"]))
            c.append({"text": h["text"], "sem_text": sem, "own": m["instance"].lower() in h["scope"].lower(),
                      "wt": WT.get(mk(h["text"])), "rel": None if j is None else bool(j["relevant"])})
        pools[i] = c
    for l in open(R3 / f"baseline-{s}.jsonl"):
        r = json.loads(l)
        base[r["id"]] = [{"text": h["text"], "rel": (J.get(jk(r["id"], h["text"])) or {}).get("relevant")} for h in r["response"].get("hits") or []]

missing = sum(1 for cs in pools.values() for c in cs if c["wt"] is None)
print(f"candidates {sum(map(len, pools.values()))}, without extraction {missing}", flush=True)

# Features
for n, (i, cs) in enumerate(pools.items()):
    mv = vec(msgs[i]["message"])
    proj = PROJECTS.get(msgs[i]["instance"], msgs[i]["instance"].lower())
    # BM25 of the message against this pool's WHEN texts
    db = sqlite3.connect(":memory:")
    db.execute("create virtual table w using fts5(t, tokenize='porter unicode61')")
    for k, c in enumerate(cs):
        db.execute("insert into w(rowid,t) values (?,?)", (k, (c["wt"] or {}).get("when", "")))
    q = " OR ".join('"' + x.replace('"', '""') + '"' for x in list(dict.fromkeys(words(msgs[i]["message"])))[:64]) if words(msgs[i]["message"]) else ""
    bm = {}
    if q:
        try:
            bm = dict(db.execute("select rowid, -bm25(w) from w where w match ?", (q,)).fetchall())
        except sqlite3.OperationalError:
            bm = {}
    for k, c in enumerate(cs):
        wt = c["wt"] or {}
        c["sem_when"] = cos(mv, vec(wt.get("when", "")))
        c["sem_not"] = cos(mv, vec(wt.get("when_not", ""))) if wt.get("when_not") else 0.0
        c["bm_when"] = bm.get(k, 0.0)
        projs = [p.lower() for p in (wt.get("case") or {}).get("projects", [])]
        c["case_ok"] = (not projs) or (not proj) or any(proj in p or p in proj for p in projs)
        c["kind"] = wt.get("kind", "?")
    if n % 25 == 0:
        print(f"JCODE_PROGRESS {json.dumps({'current': n, 'total': len(pools), 'unit': 'messages'})}", flush=True)

def auc(sub, f):
    num = den = 0
    for i in sub:
        pos = [f(c) for c in pools[i] if c["rel"] is True]
        neg = [f(c) for c in pools[i] if c["rel"] is False]
        for p in pos:
            for q in neg:
                den += 1; num += 1 if p > q else (0.5 if p == q else 0)
    return num / den if den else float("nan")

def run(pick, sub):
    out = {}; seen = collections.defaultdict(set)
    for i in sorted(sub, key=lambda i: msgs[i]["timestamp"]):
        s = msgs[i]["session"]
        got = [c for c in pick(i) if c["text"] not in seen[s]][:5]
        for c in got:
            seen[s].add(c["text"])
        out[i] = (sum(c["rel"] is True for c in got), sum(c["rel"] is False for c in got), sum(c["rel"] is None for c in got))
    return out

def boot(a, b, sub, j, reps=2000):
    sess = collections.defaultdict(list)
    for i in sub:
        sess[msgs[i]["session"]].append(i)
    cl = list(sess); rng = random.Random(813); d = []
    for _ in range(reps):
        smp = [i for s in rng.choices(cl, k=len(cl)) for i in sess[s]]
        d.append(sum(a[i][j] - b[i][j] for i in smp) / len(smp))
    d.sort(); return d[50], d[1949]

eligible = lambda c: c["own"] and len(c["text"]) <= 2000
ARMS = {
    "current recall": lambda i: base[i],
    "text rule (sem_text>=0.64)": lambda i: [c for c in sorted(pools[i], key=lambda c: -c["sem_text"]) if eligible(c) and c["sem_text"] >= 0.64],
}
for t in (0.60, 0.64, 0.68):
    ARMS[f"when rule (sem_when>={t})"] = lambda i, t=t: [c for c in sorted(pools[i], key=lambda c: -c["sem_when"]) if eligible(c) and c["sem_when"] >= t]
    ARMS[f"when rule {t} + case + not"] = lambda i, t=t: [c for c in sorted(pools[i], key=lambda c: -c["sem_when"])
                                                         if eligible(c) and c["case_ok"] and c["sem_when"] >= t and c["sem_not"] <= c["sem_when"]]
ARMS["text rule, no records"] = lambda i: [c for c in sorted(pools[i], key=lambda c: -c["sem_text"]) if eligible(c) and c["sem_text"] >= 0.64 and c["kind"] != "record"]
ARMS["text rule, no records, case"] = lambda i: [c for c in sorted(pools[i], key=lambda c: -c["sem_text"]) if eligible(c) and c["case_ok"] and c["sem_text"] >= 0.64 and c["kind"] != "record"]
ARMS["text 0.60, no records, case"] = lambda i: [c for c in sorted(pools[i], key=lambda c: -c["sem_text"]) if eligible(c) and c["case_ok"] and c["sem_text"] >= 0.60 and c["kind"] != "record"]
ARMS["text rule, records need bm_when"] = lambda i: [c for c in sorted(pools[i], key=lambda c: -c["sem_text"]) if eligible(c) and c["case_ok"] and c["sem_text"] >= 0.64 and (c["kind"] != "record" or c["bm_when"] > 0)]
ARMS["mix: 0.5 text + 0.5 when, >=0.64, case, not"] = lambda i: [c for c in sorted(pools[i], key=lambda c: -(c["sem_text"] + c["sem_when"]))
                                                                if eligible(c) and c["case_ok"] and (c["sem_text"] + c["sem_when"]) / 2 >= 0.64 and c["sem_not"] <= c["sem_when"]]

lines = ["# When-then extraction test (recall4)", "", "Extractor saw only memory text, scope and date. Same blind judgments as before.", ""]
for st in ("human", "long", "all"):
    sub = [i for i in pools if st == "all" or msgs[i]["stratum"] == st]
    lines += [f"## {st} (n={len(sub)})", "", "Signal AUC within message (0.5 = chance):", ""]
    for name, f in (("sem_text", lambda c: c["sem_text"]), ("sem_when", lambda c: c["sem_when"]), ("bm_when", lambda c: c["bm_when"]),
                    ("sem_when - sem_not", lambda c: c["sem_when"] - c["sem_not"]), ("sem_text + sem_when", lambda c: c["sem_text"] + c["sem_when"])):
        lines.append(f"- {name}: {auc(sub, f):.3f}")
    kinds = collections.Counter((c["kind"], c["rel"]) for i in sub for c in pools[i] if c["rel"] is not None)
    lines += ["", "Relevant share by extracted kind: " + ", ".join(f"{k} {kinds[(k, True)] / max(1, kinds[(k, True)] + kinds[(k, False)]):.2f} (n={kinds[(k, True)] + kinds[(k, False)]})" for k in ("rule", "preference", "fact", "record")),
              f"Relevant share when CASE excludes the project: {sum(1 for i in sub for c in pools[i] if c['rel'] and not c['case_ok']) / max(1, sum(1 for i in sub for c in pools[i] if c['rel'] is not None and not c['case_ok'])):.2f} (n={sum(1 for i in sub for c in pools[i] if c['rel'] is not None and not c['case_ok'])})", ""]
    res = {name: run(f, sub) for name, f in ARMS.items()}
    lines += ["| arm | rel/msg | irr/msg | unjudged/msg | rel vs current [CI] | irr vs current [CI] |", "|---|---:|---:|---:|---:|---:|"]
    cur = res["current recall"]
    for name, r in res.items():
        a = sum(x[0] for x in r.values()) / len(sub); b = sum(x[1] for x in r.values()) / len(sub); u = sum(x[2] for x in r.values()) / len(sub)
        if name == "current recall":
            lines.append(f"| {name} | {a:.2f} | {b:.2f} | {u:.2f} | - | - |"); continue
        lo, hi = boot(r, cur, sub, 0); lo2, hi2 = boot(r, cur, sub, 1)
        lines.append(f"| {name} | {a:.2f} | {b:.2f} | {u:.2f} | {a - sum(x[0] for x in cur.values()) / len(sub):+.2f} [{lo:+.2f}, {hi:+.2f}] | {b - sum(x[1] for x in cur.values()) / len(sub):+.2f} [{lo2:+.2f}, {hi2:+.2f}] |")
    lines.append("")
text = "\n".join(lines)
(R / "report-whenthen.md").write_text(text)
print(text)
