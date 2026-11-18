#!/usr/bin/env python3
"""Time jcode's post_tool_feedback hook (n7jbmx) as jcode runs it: one
`hug mem hook jcode` process per tool call, on a copy of the live store
(HUG_HOME=hookhome, tool_recall on), niced like nothing: the hook is not.
Usage: time_hook.py HUG REPLAY OUT.json [--index]"""
import json, os, subprocess, sys, time
hug, replay, out = sys.argv[1:4]
here = os.path.dirname(os.path.abspath(__file__))
base = {k: v for k, v in os.environ.items() if not k.startswith("JCODE_") or k == "JCODE_SCRATCH_DIR"}
base.update(HUG_HOME=os.path.join(here, "hookhome"))
if "--index" in sys.argv:
    t = time.time()
    r = subprocess.run([hug, "mem", "index"], env=base, capture_output=True, text=True)
    print("index", r.returncode, round(time.time() - t, 1), "s", r.stdout[-300:], r.stderr[-300:], flush=True)
rows = []
for m in json.load(open(replay)):
    env = dict(base, JCODE_HOME=m["home"], JCODE_HOOK_EVENT="post_tool_feedback", JCODE_HOOK_SESSION_ID="session_timing_" + str(m["id"]),
               JCODE_HOOK_TOOL_NAME=m["tool"], JCODE_HOOK_CWD=m["cwd"], JCODE_HOOK_TOUCHED_PATHS="\n".join(m["paths"]),
               JCODE_HOOK_TOOL_INPUT=json.dumps({"command": m["command"]} if m["tool"] == "bash" else {"file_path": (m["paths"] or [""])[0]}))
    t = time.perf_counter()
    r = subprocess.run([hug, "mem", "hook", "jcode"], env=env, capture_output=True, text=True, timeout=30)
    ms = (time.perf_counter() - t) * 1000
    rows.append({"id": m["id"], "ms": ms, "rc": r.returncode, "printed": bool(r.stdout.strip()), "stderr": r.stderr[-200:]})
lat = sorted(x["ms"] for x in rows)
q = lambda p: lat[min(len(lat) - 1, int(p * len(lat)))]
summary = {"calls": len(rows), "p50_ms": round(q(.5), 1), "p95_ms": round(q(.95), 1), "max_ms": round(lat[-1], 1),
           "printed": sum(x["printed"] for x in rows), "nonzero_exit": sum(x["rc"] != 0 for x in rows), "load": os.getloadavg()}
json.dump({"summary": summary, "rows": rows}, open(out, "w"), indent=1)
print(json.dumps(summary))
