#!/usr/bin/env python3
"""recall4: which retrieval ideas (from Vestige and IR basics) help muninn?

Per LongMemEval_S question: index the rounds in an in-memory SQLite FTS5
table, rank with each arm, score. Dense arms reuse recall3's cached nomic
document vectors (muninn.db in each longmem-homes/<id>) and cache query
vectors in qvec/. Evidence labels are only read by the scorer.

Usage: recall4.py dev            100 recall3 questions, all arms
       recall4.py heldout        the other 400 questions, lexical arms only
"""
import json, math, os, random, re, sqlite3, struct, sys, urllib.request, hashlib, collections, time
from pathlib import Path

S = Path(os.environ["JCODE_SCRATCH_DIR"])
R3 = S / "recall3"
OUT = S / "recall4"
OUT.mkdir(exist_ok=True)
(OUT / "qvec").mkdir(exist_ok=True)

STOP = set("""a about above after again against all am an and any are as at be because been before being below between both but by
can could did do does doing down during each few for from further had has have having he her here hers herself him himself his how i if in
into is it its itself just me more most my myself no nor not now of off on once only or other our ours ourselves out over own same she should
so some such than that the their theirs them themselves then there these they this those through to too under until up very was we were what
when where which while who whom why will with would you your yours yourself yourselves i'm i've i'd i'll you're it's don't can't didn't
also get got tell know remember mentioned mention said say did any much many ago last""".split())

def words(text):
    return [w for w in re.findall(r"[a-z0-9]+(?:'[a-z]+)?", text.lower()) if len(w) > 1 and w not in STOP]

def fts_quote(w):
    return '"' + w.replace('"', '""') + '"'

def parse_date(s):
    return time.mktime(time.strptime(s[:10], "%Y/%m/%d"))

# ---------- data ----------

def rounds_of(q):
    out = []
    for sid, date, turns in zip(q["haystack_session_ids"], q["haystack_dates"], q["haystack_sessions"]):
        group, n = [], 0
        def emit():
            nonlocal n
            if group:
                out.append({"key": f"{sid}:{n}", "session": sid, "date": date, "turns": list(group),
                            "evidence": any(t.get("has_answer", False) for t in group)})
                n += 1
        for t in turns:
            if t["role"] == "user" and group:
                emit(); group = []
            group.append(t)
        emit()
    return out

def load(split):
    data = json.loads((R3 / "longmemeval_s_cleaned.json").read_text())
    dev_ids = {q["id"] for q in json.loads((R3 / "longmem-subset.json").read_text())}
    qs = [q for q in data if (q["question_id"] in dev_ids) == (split == "dev")]
    return qs

# ---------- dense ----------

def qvec(text):
    h = hashlib.sha256(("nomic-embed-text\0" + text).encode()).hexdigest()
    p = OUT / "qvec" / h
    if p.exists():
        b = p.read_bytes()
    else:
        req = urllib.request.Request("http://localhost:11434/api/embed", data=json.dumps({"model": "nomic-embed-text", "input": [text]}).encode())
        v = json.load(urllib.request.urlopen(req, timeout=120))["embeddings"][0]
        n = math.sqrt(sum(x * x for x in v)) or 1
        b = struct.pack(f"<{len(v)}f", *[x / n for x in v])
        p.write_bytes(b)
    return struct.unpack(f"<{len(b)//4}f", b)

def doc_vecs(qid):
    db = R3 / "longmem-homes" / qid / "muninn.db"
    if not db.exists():
        return None
    c = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
    rows = c.execute("select u.key, e.vec from units u join embeddings e on e.content_hash=u.content_hash").fetchall()
    return {k: struct.unpack(f"<{len(v)//4}f", v) for k, v in rows}

def dot(a, b):
    return sum(x * y for x, y in zip(a, b))

# ---------- lexical index ----------

class Index:
    def __init__(self, rounds, tokenizer):
        self.c = sqlite3.connect(":memory:")
        tk = f"tokenize='{tokenizer}'"
        self.c.execute(f"create virtual table r using fts5(body, user, asst, {tk})")
        self.c.execute(f"create virtual table p using fts5(body, rid UNINDEXED, {tk})")
        for i, r in enumerate(rounds):
            user = "\n".join(t["content"] for t in r["turns"] if t["role"] == "user")
            asst = "\n".join(t["content"] for t in r["turns"] if t["role"] != "user")
            self.c.execute("insert into r(rowid, body, user, asst) values (?,?,?,?)", (i, user + "\n" + asst, user, asst))
            for chunk in passages(r):
                self.c.execute("insert into p(body, rid) values (?,?)", (chunk, i))

    def bm25(self, qwords, cols="body", weights=None):
        """BM25 over columns; returns {round: score>0}. FTS5 bm25() is negative."""
        if not qwords:
            return {}
        expr = " OR ".join(fts_quote(w) for w in qwords)
        if cols != "body":
            expr = "{" + cols + "}: (" + expr + ")"
        w = ",".join(str(x) for x in (weights or [1.0, 0.0, 0.0]))
        rows = self.c.execute(f"select rowid, -bm25(r, {w}) from r where r match ?", (expr,)).fetchall()
        return {i: s for i, s in rows if s > 0}

    def best_passage(self, qwords):
        if not qwords:
            return {}
        expr = " OR ".join(fts_quote(w) for w in qwords)
        best = {}
        for rid, s in self.c.execute("select rid, -bm25(p) from p where p match ?", (expr,)):
            if s > best.get(rid, 0):
                best[rid] = s
        return best

    def term_sets(self, qwords):
        """round ids containing each query term (after the tokenizer's stemming)."""
        out = {}
        for w in qwords:
            out[w] = {i for (i,) in self.c.execute("select rowid from r where r match ?", ("{body}: " + fts_quote(w),))}
        return out

def passages(r, size=600):
    """Each turn, long turns cut into ~size-char pieces on paragraph breaks."""
    for t in r["turns"]:
        text, cur = t["content"], ""
        for para in re.split(r"\n\s*\n", text):
            if cur and len(cur) + len(para) > size:
                yield cur
                cur = ""
            cur = (cur + "\n\n" + para) if cur else para
        if cur:
            yield cur

# ---------- ranking helpers ----------

def norm(scores):
    if not scores:
        return {}
    m = max(scores.values())
    return {k: v / m for k, v in scores.items()} if m > 0 else dict(scores)

def order(scores):
    return [k for k, _ in sorted(scores.items(), key=lambda kv: (-kv[1], kv[0]))]

def rrf(*rankings, k=60):
    out = collections.defaultdict(float)
    for rk in rankings:
        for pos, i in enumerate(rk):
            out[i] += 1.0 / (k + pos + 1)
    return dict(out)

def cover_rerank(scores, terms, rounds, lam=0.5, sess_pen=0.0, depth=30, k=10):
    """Greedy: base score + lam * share of not-yet-covered query terms,
    minus sess_pen for a session already picked."""
    cand = order(scores)[:depth]
    base = norm({i: scores[i] for i in cand})
    nt = max(1, len(terms))
    has = {i: {w for w, s in terms.items() if i in s} for i in cand}
    covered, picked, sessions = set(), [], collections.Counter()
    while cand and len(picked) < k:
        def gain(i):
            return base[i] + lam * len(has[i] - covered) / nt - sess_pen * sessions[rounds[i]["session"]]
        best = max(cand, key=lambda i: (gain(i), -cand.index(i)))
        picked.append(best); cand.remove(best)
        covered |= has[best]; sessions[rounds[best]["session"]] += 1
    rest = [i for i in order(scores) if i not in picked]
    return picked + rest

# ---------- arms ----------

def arms(q, rounds, plain, porter, dvec):
    qw = words(q["question"])
    out = {}
    b_plain = plain.bm25(qw)
    b = porter.bm25(qw)
    out["bm25"] = order(b_plain)
    out["bm25-stem"] = order(b)
    # BM25F: user words weigh more than assistant words (Vestige's role fields).
    for wu in (2.0, 3.0):
        out[f"bm25f-user{wu:g}"] = order(porter.bm25(qw, cols="user asst", weights=[0.0, wu, 1.0]))
    # best passage mixed with whole-round score (Vestige window_mix 0.65).
    bp = porter.best_passage(qw)
    for mix in (0.35, 0.65):
        nb, np_ = norm(b), norm(bp)
        out[f"passage{mix:g}"] = order({i: mix * np_.get(i, 0) + (1 - mix) * nb.get(i, 0) for i in set(nb) | set(np_)})
    # Combined: role-weighted BM25F mixed with best passage.
    bf = porter.bm25(qw, cols="user asst", weights=[0.0, 3.0, 1.0])
    for mix in (0.35, 0.65):
        nb, np_ = norm(bf), norm(bp)
        out[f"user3+passage{mix:g}"] = order({i: mix * np_.get(i, 0) + (1 - mix) * nb.get(i, 0) for i in set(nb) | set(np_)})
    for wu in (5.0, 10.0, 1000.0):
        out[f"bm25f-user{wu:g}"] = order(porter.bm25(qw, cols="user asst", weights=[0.0, wu, 1.0]))
    terms = porter.term_sets(qw)
    for lam in (0.3, 0.6):
        out[f"cover{lam:g}"] = cover_rerank(b, terms, rounds, lam=lam)
    out["session-spread"] = cover_rerank(b, terms, rounds, lam=0.0, sess_pen=0.3)
    # recency: newer sessions win close calls (knowledge updates).
    dates = [parse_date(r["date"]) for r in rounds]
    lo, hi = min(dates), max(dates)
    span = (hi - lo) or 1
    for beta in (0.1, 0.3):
        out[f"recency{beta:g}"] = order({i: s * (1 + beta * (dates[i] - lo) / span) for i, s in b.items()})
    if dvec is not None:
        qv = qvec(q["question"])
        dense = {i: dot(qv, dvec[r["key"]]) for i, r in enumerate(rounds) if r["key"] in dvec}
        out["dense"] = order(dense)
        out["rrf(bm25-stem,dense)"] = order(rrf(out["bm25-stem"], out["dense"]))
        out["rrf(bm25,dense)"] = order(rrf(out["bm25"], out["dense"]))
        # The combination the single arms suggest, scored on the same footing.
        hyb = rrf(out["passage0.35"], out["dense"])
        out["rrf(passage,dense)"] = order(hyb)
        out["rrf(passage,dense)+cover"] = cover_rerank(hyb, terms, rounds, lam=0.3)
    return out

# ---------- metrics ----------

def ndcg10(ranked, gold):
    rel = [1.0 if i in gold else 0.0 for i in ranked[:10]]
    dcg = (rel[0] if rel else 0) + sum(r / math.log2(p + 1) for p, r in enumerate(rel[1:], start=2))
    ideal = [1.0] * min(len(gold), 10)
    idcg = (ideal[0] if ideal else 0) + sum(r / math.log2(p + 1) for p, r in enumerate(ideal[1:], start=2))
    return dcg / idcg if idcg else 0.0

def metrics(ranked, rounds):
    gold = {i for i, r in enumerate(rounds) if r["evidence"]}
    gold_s = {rounds[i]["session"] for i in gold}
    sess = []
    for i in ranked:
        s = rounds[i]["session"]
        if s not in sess:
            sess.append(s)
    return {
        "all@5": float(gold <= set(ranked[:5])),
        "all@10": float(gold <= set(ranked[:10])),
        "ndcg@10": ndcg10(ranked, gold),
        "sess_any@5": float(bool(gold_s & set(sess[:5]))),
        "sess_all@5": float(gold_s <= set(sess[:5])),
    }

def stored(qid, variant, rounds):
    """muninn's own ranking from recall3 (reference arms)."""
    p = R3 / "longmem-homes" / qid / "results.jsonl"
    if not p.exists():
        return None
    pos = {r["key"]: i for i, r in enumerate(rounds)}
    for line in p.open():
        r = json.loads(line)
        if r["variant"] == variant:
            return [pos[x["Unit"]["Key"]] for x in (r["results"] or []) if x["Unit"]["Key"] in pos]
    return None

def main():
    split = sys.argv[1] if len(sys.argv) > 1 else "dev"
    qs = load(split)
    per = collections.defaultdict(dict)  # arm -> qid -> metrics
    types = {}
    t0 = time.time()
    for n, q in enumerate(qs, 1):
        if q["question_id"].endswith("_abs"):
            continue
        rounds = rounds_of(q)
        types[q["question_id"]] = q["question_type"]
        plain, porter = Index(rounds, "unicode61"), Index(rounds, "porter unicode61")
        dvec = doc_vecs(q["question_id"]) if split == "dev" else None
        rk = arms(q, rounds, plain, porter, dvec)
        if split == "dev":
            for v in ("fts-retrieval", "retrieval", "baseline"):
                s = stored(q["question_id"], v, rounds)
                if s is not None:
                    rk["muninn:" + v] = s
        for arm, ranked in rk.items():
            per[arm][q["question_id"]] = metrics(ranked, rounds)
        if n % 20 == 0:
            print(f"JCODE_PROGRESS {json.dumps({'current': n, 'total': len(qs), 'unit': 'questions'})}", flush=True)
    (OUT / f"per-question-{split}.json").write_text(json.dumps({"per": per, "types": types}))
    report(per, types, split)
    print(f"done in {time.time()-t0:.0f}s")

def boot_delta(per, arm, ref, key, reps=2000, seed=7):
    ids = sorted(set(per[arm]) & set(per[ref]))
    d = [per[arm][i][key] - per[ref][i][key] for i in ids]
    rnd = random.Random(seed)
    means = sorted(sum(d[rnd.randrange(len(d))] for _ in d) / len(d) for _ in range(reps))
    return sum(d) / len(d), means[int(0.025 * reps)], means[int(0.975 * reps)]

def report(per, types, split):
    ref = "bm25"
    keys = ["all@5", "all@10", "ndcg@10", "sess_any@5", "sess_all@5"]
    lines = [f"# recall4 {split}: {len(per[ref])} non-abstention questions", "",
             "Paired bootstrap 95% CI of the all@5 delta against plain bm25 (2000 reps).", "",
             "| arm | " + " | ".join(keys) + " | all@5 delta vs bm25 [CI] |", "|---|" + "---:|" * (len(keys) + 1)]
    for arm in sorted(per, key=lambda a: -sum(m["all@5"] for m in per[a].values()) / len(per[a])):
        avg = {k: sum(m[k] for m in per[arm].values()) / len(per[arm]) for k in keys}
        d, lo, hi = boot_delta(per, arm, ref, "all@5")
        lines.append(f"| {arm} | " + " | ".join(f"{avg[k]:.3f}" for k in keys) + f" | {d:+.3f} [{lo:+.3f}, {hi:+.3f}] |")
    lines += ["", "## all@5 by question type", ""]
    tnames = sorted(set(types.values()))
    lines += ["| arm | " + " | ".join(f"{t} (n={sum(1 for v in types.values() if v == t)})" for t in tnames) + " |",
              "|---|" + "---:|" * len(tnames)]
    for arm in sorted(per):
        cells = []
        for t in tnames:
            ids = [i for i, v in types.items() if v == t and i in per[arm]]
            cells.append(f"{sum(per[arm][i]['all@5'] for i in ids) / max(1, len(ids)):.2f}")
        lines.append(f"| {arm} | " + " | ".join(cells) + " |")
    text = "\n".join(lines) + "\n"
    (OUT / f"report-{split}.md").write_text(text)
    print(text)

if __name__ == "__main__":
    main()
