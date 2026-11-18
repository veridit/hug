#!/usr/bin/env python3
"""sheets.py: side-by-side before/after sheets and the view-line strip."""
from PIL import Image, ImageDraw, ImageFont
O = "/Users/jhf/.jcode/scratch/grid-mock/out"
lab = ImageFont.truetype("/System/Library/Fonts/Menlo.ttc", 18, index=1)

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
              [f"BEFORE (main) {view} {size}", f"AFTER order A {view} {size}", f"AFTER order B {view} {size}"],
              f"{O}/compare-{view}-{size}.png", horizontal=(size == "100x30"))

# view line: crop line 2 (y = pad 12 + 1 row of 20px) from each state
rows = []
def line(p, n=1):
    im = Image.open(p); ch = int(15 * 1.35); return im.crop((0, 12 + n * ch - 2, im.width, 12 + (n + 1) * ch + 2))
states = [("before: idle", "before", "overview-initial"), ("before: after v (whole line lights up)", "before", "viewline-pending"),
          ("before: after v o", "before", "after-vo"),
          ("after: idle (v cyan, o l b ! violet)", "after-A", "overview-initial"), ("after: v pressed, blink phase 1", "after-A", "viewline-pending"),
          ("after: v pressed, phase 2 (+530 ms)", "after-A", "viewline-pending-blink2"), ("after: v o chosen, cursor gone", "after-A", "after-vo"),
          ("alt 'letters': v pressed, o l b ! underlined", "after-A-letters", "viewline-pending")]
ims = [(l, line(f"{O}/{v}/100x30/{f}.png")) for l, v, f in states]
W = max(i.width for _, i in ims) + 470; H = sum(i.height + 6 for _, i in ims)
out = Image.new("RGB", (W, H), (60, 60, 60)); d = ImageDraw.Draw(out); y = 0
small = ImageFont.truetype("/System/Library/Fonts/Menlo.ttc", 14)
for l, im in ims:
    d.text((8, y + 3), l, font=small, fill=(255, 255, 255)); out.paste(im, (470, y)); y += im.height + 6
out.save(f"{O}/compare-viewline.png")
print("ok")
