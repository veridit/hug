#!/usr/bin/env python3
"""Generate N synthetic hug task files in the current directory (a .hug/tasks worktree).

Used on 2026-11-16 for the 10,000-task measurement in task 44w1ev.
Usage: cd <repo>/.hug/tasks && python3 gen-scale-board.py [N]   (default 10000)
Then: git add -A && git commit -m "synthetic tasks"
Deterministic (seed 7). Ids use hug's id alphabet, ranks its rank alphabet.
"""
import os
import random
import sys

N = int(sys.argv[1]) if len(sys.argv) > 1 else 10000
random.seed(7)
A = "0123456789abcdefghjkmnpqrstvwxyz"
R = "0123456789abcdefghijklmnopqrstuvwxyz"
words = ("retry parser cache login export import schema index worker queue report billing "
         "search upload token session layout router render config metric alert backup migrate sync").split()
types = ["feature", "bug", "chore", "spike", "docs"]
pr = ["critical", "high", "medium", "low", ""]
groups = ["", "", "", "api", "web", "compiler", "later", "ops"]


def rank(n):
    s = ""
    for _ in range(6):
        s = R[n % 36] + s
        n //= 36
    return s


ids = set()
while len(ids) < N:
    ids.add("".join(random.choice(A) for _ in range(6)))
ids = sorted(ids)
random.shuffle(ids)
epics = ids[:150]
step = max(1, (36 ** 6 - 1) // (N + 1))
for n, i in enumerate(ids):
    t = " ".join(random.choice(words) for _ in range(random.randint(3, 9))).capitalize()
    slug = "-".join(t.lower().split()[:4])[:32]
    st = random.choices(["todo", "doing", "done", "dropped"], [45, 3, 48, 4])[0]
    g = random.choice(groups)
    checks = [" ".join(random.choice(words) for _ in range(random.randint(4, 14)))
              for _ in range(random.randint(1, 5))]
    box = "x" if st == "done" else " "
    deps = ""
    if n > 200 and random.random() < 0.08:
        deps = "- task:%s\n" % ids[random.randrange(150, n)]
    parent = ""
    if n >= 150 and random.random() < 0.3:
        parent = "parent: %s\n" % random.choice(epics)
    notes = "\n\n".join(" ".join(random.choice(words) for _ in range(random.randint(20, 120)))
                        for _ in range(random.randint(0, 4)))
    body = f"""---
id: {i}
title: "{t}"
status: {st}
type: {random.choice(types)}
priority: "{random.choice(pr)}"
assignee: {'["tester"]' if st == "doing" else "[]"}
labels: []
{parent}created: 2026-10-{random.randint(1, 28):02d}
updated: 2026-11-{random.randint(1, 16):02d}
---

## Goal
{" ".join(random.choice(words) for _ in range(18))}

## Purpose
{" ".join(random.choice(words) for _ in range(14))}

## Principle
{" ".join(random.choice(words) for _ in range(14))}

## Relevant details
{" ".join(random.choice(words) for _ in range(40))}

## Depends on
{deps}
## Acceptance
{"".join(f"- [{box}] {c}" + chr(10) for c in checks)}
## Notes
{notes}
"""
    name = f"{rank((n + 1) * step)}.{i}.{slug}.md"
    p = os.path.join(g, name) if g else name
    if g:
        os.makedirs(g, exist_ok=True)
    with open(p, "w") as f:
        f.write(body)
print("written", len(ids))
