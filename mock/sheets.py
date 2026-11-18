#!/usr/bin/env python3
"""sheets.py: 911j70 compare sheets from current main (kikazaru), in puppy's
format: BEFORE (main) | order A | order B, side by side at 100x30, stacked
at 160x45. Renders each .ansi with puppy's mock/ansi2png.py first."""
import subprocess, sys
from PIL import Image, ImageDraw, ImageFont

O = "/Users/jhf/.jcode/scratch/911j70-sheets-out/out"
A2P = "/Users/jhf/.jcode/scratch/grid-mock/mock/ansi2png.py"
lab = ImageFont.truetype("/System/Library/Fonts/Menlo.ttc", 18, index=1)

for v in ["before", "after-A", "after-B"]:
    for size in ["100x30", "160x45"]:
        w, h = size.split("x")
        for view in ["overview-cursor", "list-cursor", "board-cursor"]:
            subprocess.run([sys.executable, A2P, f"{O}/{v}/{size}/{view}.ansi", f"{O}/{v}/{size}/{view}.png", w, h], check=True)

def stack(paths, labels, dst, horizontal=True):
    ims = [Image.open(p) for p in paths]
    gap, head = 16, 30
    if horizontal:
        W = sum(i.width for i in ims) + gap * (len(ims) - 1); H = max(i.height for i in ims) + head
    else:
        W = max(i.width for i in ims); H = sum(i.height + head for i in ims) + gap * (len(ims) - 1)
    out = Image.new("RGB", (W, H), (60, 60, 60)); d = ImageDraw.Draw(out); x = y = 0
    for im, l in zip(ims, labels):
        d.text((x + 8, y + 5), l, font=lab, fill=(255, 255, 255))
        out.paste(im, (x, y + head))
        if horizontal: x += im.width + gap
        else: y += im.height + head + gap
    out.save(dst)

for size in ["100x30", "160x45"]:
    for view in ["overview-cursor", "list-cursor", "board-cursor"]:
        stack([f"{O}/{v}/{size}/{view}.png" for v in ["before", "after-A", "after-B"]],
              [f"BEFORE (main v0.7.70+tags) {view} {size}", f"order A {view} {size}", f"order B {view} {size}"],
              f"{O}/compare-{view}-{size}-main.png", horizontal=(size == "100x30"))
print("ok")
