#!/usr/bin/env python3
"""Blind-judge a tool-call replay set (n7jbmx) with judge_tool.py's rubric,
unchanged (the TASK text is read from judge_tool.py itself), the same
DeepSeek Flash transport (judge_real.job), and report against the bar
written on the ticket before any run.

Usage: judge_set.py SETDIR ARM...   (SETDIR has replay.json and ARM.jsonl)
Labels are cached in SETDIR/judgments.json. Writes SETDIR/report.md and
SETDIR/noise-examples.md (5 noise injections per arm, verbatim)."""
import collections, concurrent.futures, json, os, pathlib, random, re, sys

S = pathlib.Path(os.environ["JCODE_SCRATCH_DIR"])
setdir, arms_names = pathlib.Path(sys.argv[1]), sys.argv[2:]
src = open(S / "recall4" / "judge_real.py").read().split('if __name__ == "__main__":')[0]
jr = {"__name__": "jr"}
saved, sys.argv = sys.argv, ["judge_real.py", "--report"]
exec(compile(src, "judge_real.py", "exec"), jr)
sys.argv = saved
tool_src = open(S / "recall4" / "judge_tool.py").read()
jr["TASK"] = re.search(r"jr\.TASK = '''(.*?)'''", tool_src, re.S).group(1)
cachefile = setdir / "judgments.json"
cache = json.loads(cachefile.read_text()) if cachefile.exists() else {}
msgs = {m["id"]: m for m in json.loads((setdir / "replay.json").read_text())}
key = jr["key"]

def load(v):
    out, lat = {}, []
    for l in open(setdir / f"{v}.jsonl"):
        r = json.loads(l)
        out[r["id"]] = [h["text"] for h in r["response"].get("hits") or []]
        lat.append(r["latency_ms"])
    return out, lat

arms = {v: load(v) for v in arms_names}
todo = {}
for rows, _ in arms.values():
    for i, hs in rows.items():
        for t in hs:
            k = key(i, t)
            if k not in cache:
                todo[k] = {"id": k, "message": msgs[i]["message"], "memory": t}
items = list(todo.values()); random.Random(7521).shuffle(items)
batches = [items[j:j + 6] for j in range(0, len(items), 6)]
print(f"{len(items)} pairs to judge, {len(batches)} batches", flush=True)
def safe(b):
    try:
        return jr["job"](b) or []
    except Exception as e:
        print("batch skipped:", e, flush=True)
        return []
with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
    for n, ratings in enumerate(pool.map(safe, batches)):
        for r in ratings:
            cache[r["id"]] = r
        cachefile.write_text(json.dumps(cache, ensure_ascii=False, indent=1))
        print(f"JCODE_PROGRESS {json.dumps({'current': n + 1, 'total': len(batches), 'unit': 'batches'})}", flush=True)

def per(rows):
    return {i: (sum(cache.get(key(i, t), {}).get("relevant") is True for t in hs),
                sum(cache.get(key(i, t), {}).get("relevant") is False for t in hs)) for i, hs in rows.items()}

sess = collections.defaultdict(list)
for i, m in msgs.items():
    sess[m["session"]].append(i)
cl = sorted(sess)
def boot(stat, reps=2000):
    rng = random.Random(813); d = []
    for _ in range(reps):
        smp = [i for s in rng.choices(cl, k=len(cl)) for i in sess[s]]
        d.append(stat(smp))
    d.sort(); return d[int(reps * .025)], d[int(reps * .975) - 1]

n = len(msgs)
P = {v: per(rows) for v, (rows, _) in arms.items()}
rate = lambda p, j: (lambda smp: 100 * sum(p[i][j] for i in smp) / len(smp))
lines = [f"# Tool-call recall, {setdir.name}: {n} calls, {len(cl)} sessions, strict blind judge", "",
         "Bar (fresh set, fixed before any run): noise <=10/100 (CI upper <=15), useful >=10/100 (CI lower >0), precision >=50%,",
         "noise below default with the CI excluding 0, hook p95 <200 ms on the live store.", "",
         "| arm | useful /100 [CI] | noise /100 [CI] | precision | calls with any | noise vs default [CI] | p95 ms |", "|---|---:|---:|---:|---:|---:|---:|"]
unjudged = 0
for v, (rows, lat) in arms.items():
    p = P[v]
    u, z = rate(p, 0)(list(msgs)), rate(p, 1)(list(msgs))
    ul, uh = boot(rate(p, 0)); zl, zh = boot(rate(p, 1))
    tot = sum(a + b for a, b in p.values())
    unjudged += sum(len(hs) for hs in rows.values()) - tot
    prec = sum(a for a, _ in p.values()) / tot if tot else float("nan")
    anyinj = sum(1 for hs in rows.values() if hs) / n
    vs = "-"
    if "default" in P and v != "default":
        b = P["default"]
        dl, dh = boot(lambda smp: 100 * sum(p[i][1] - b[i][1] for i in smp) / len(smp))
        vs = f"{z - rate(b, 1)(list(msgs)):+.1f} [{dl:+.1f}, {dh:+.1f}]"
    lat = sorted(lat)
    p95 = lat[int(.95 * (len(lat) - 1))] if lat else float("nan")
    lines.append(f"| {v} | {u:.1f} [{ul:.1f}, {uh:.1f}] | {z:.1f} [{zl:.1f}, {zh:.1f}] | {prec:.0%} | {anyinj:.0%} | {vs} | {p95:.0f} |")
lines += ["", f"Unjudged pairs (judge skipped): {unjudged}. p95 here is replay latency on a scratch copy, not the hook on the live store."]
text = "\n".join(lines) + "\n"
(setdir / "report.md").write_text(text)
print(text)
ex = [f"# Noise examples, {setdir.name}: 5 per arm, verbatim, seeded", ""]
for v, (rows, _) in arms.items():
    noise = [(i, t) for i, hs in sorted(rows.items()) for t in hs if cache.get(key(i, t), {}).get("relevant") is False]
    ex.append(f"## {v} ({len(noise)} noise injections)")
    for i, t in random.Random(5).sample(noise, min(5, len(noise))):
        ex += ["", f"- call {i}: `{msgs[i]['message'][:300]}`", f"  memory: {t[:600]}", f"  judge: {cache[key(i, t)].get('reason', '')}"]
    ex.append("")
(setdir / "noise-examples.md").write_text("\n".join(ex))
