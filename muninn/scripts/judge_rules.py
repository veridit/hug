"""Judge the unjudged picks of the two leading local rules, then re-score."""
import json, sys, os, importlib.util, random
sys.argv = ["judge_real.py", "--report"]
spec = importlib.util.spec_from_file_location("jr", "judge_real.py"); jr = importlib.util.module_from_spec(spec); spec.loader.exec_module(jr)
src = open("local_pick.py").read().split("w=fit(dev)")[0]
ns = {}; exec(src, ns)
pools, msgs = ns["pools"], ns["msgs"]
todo = {}
for ms in (0.60, 0.64):
    for i, cs in pools.items():
        picks = [c for c in sorted(cs, key=lambda c: -c["sem"]) if c["sem"] >= ms and len(c["text"]) <= 2000 and c["own"]][:5]
        for c in picks:
            k = jr.key(i, c["text"])
            if k not in jr.cache:
                todo[k] = {"id": k, "message": msgs[i]["message"], "memory": c["text"]}
items = list(todo.values()); random.Random(7521).shuffle(items)
print("unjudged rule picks:", len(items), flush=True)
batches = [items[j:j + 6] for j in range(0, len(items), 6)]
import concurrent.futures
with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
    for n, ratings in enumerate(pool.map(lambda b: jr.job(b) or [], batches)):
        for r in ratings:
            jr.cache[r["id"]] = r
        json.dump(jr.cache, open(jr.cachefile, "w"), ensure_ascii=False, indent=1)
        print(f"JCODE_PROGRESS {json.dumps({'current': n + 1, 'total': len(batches), 'unit': 'batches'})}", flush=True)
