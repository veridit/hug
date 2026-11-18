#!/usr/bin/env python3
"""ansi2png.py IN.ansi OUT.png COLS ROWS: render a tmux capture-pane -e dump.

SGR subset: reset, bold, dim, italic, underline, reverse, 30-37/90-97/40-47/100-107,
38;5;n / 48;5;n, 38;2;r;g;b / 48;2;r;g;b, 39/49 defaults. Dark background #0b1620.
"""
import re, sys
from PIL import Image, ImageDraw, ImageFont

FG0, BG0 = (220, 226, 234), (11, 22, 32)
BASIC = [(0,0,0),(205,49,49),(13,188,121),(229,229,16),(36,114,200),(188,63,188),(17,168,205),(229,229,229),
         (102,102,102),(241,76,76),(35,209,139),(245,245,67),(59,142,234),(214,112,214),(41,184,219),(255,255,255)]

def x256(n):
    if n < 16: return BASIC[n]
    if n < 232:
        n -= 16; c = [0,95,135,175,215,255]
        return (c[n//36], c[(n//6)%6], c[n%6])
    v = 8 + (n-232)*10; return (v,v,v)

def parse(path, cols, rows):
    text = open(path, encoding="utf-8", errors="replace").read()
    grid = []
    for line in text.split("\n")[:rows]:
        cells = []; st = dict(fg=None, bg=None, b=False, u=False, d=False, r=False)
        for tok in re.split(r"(\x1b\[[0-9;:]*m)", line):
            if tok.startswith("\x1b["):
                ps = [p for p in re.split(r"[;:]", tok[2:-1])] or ["0"]
                ps = [int(p) if p else 0 for p in ps]
                i = 0
                while i < len(ps):
                    p = ps[i]
                    if p == 0: st = dict(fg=None, bg=None, b=False, u=False, d=False, r=False)
                    elif p == 1: st["b"] = True
                    elif p == 2: st["d"] = True
                    elif p == 4: st["u"] = True
                    elif p == 7: st["r"] = True
                    elif p == 22: st["b"] = st["d"] = False
                    elif p == 24: st["u"] = False
                    elif p == 27: st["r"] = False
                    elif 30 <= p <= 37: st["fg"] = BASIC[p-30]
                    elif 90 <= p <= 97: st["fg"] = BASIC[p-90+8]
                    elif 40 <= p <= 47: st["bg"] = BASIC[p-40]
                    elif 100 <= p <= 107: st["bg"] = BASIC[p-100+8]
                    elif p == 39: st["fg"] = None
                    elif p == 49: st["bg"] = None
                    elif p in (38, 48) and i+1 < len(ps):
                        key = "fg" if p == 38 else "bg"
                        if ps[i+1] == 5 and i+2 < len(ps): st[key] = x256(ps[i+2]); i += 2
                        elif ps[i+1] == 2 and i+4 < len(ps): st[key] = tuple(ps[i+2:i+5]); i += 4
                    i += 1
            else:
                for ch in tok:
                    cells.append((ch, dict(st)))
        grid.append(cells[:cols])
    return grid

def main():
    src, dst, cols, rows = sys.argv[1], sys.argv[2], int(sys.argv[3]), int(sys.argv[4])
    size = 15
    reg = ImageFont.truetype("/System/Library/Fonts/Menlo.ttc", size, index=0)
    bold = ImageFont.truetype("/System/Library/Fonts/Menlo.ttc", size, index=1)
    fallback = ImageFont.truetype("/System/Library/Fonts/Apple Symbols.ttf", size + 2)
    notdef = bytes(reg.getmask("\U0010fffd"))
    cache = {}
    def missing(f, c):
        if c not in cache:
            cache[c] = ord(c) > 0x2000 and bytes(reg.getmask(c)) == notdef
        return cache[c]
    cw = int(round(reg.getlength("M"))); ch = int(size * 1.35)
    pad = 12
    img = Image.new("RGB", (cols*cw + 2*pad, rows*ch + 2*pad), BG0)
    d = ImageDraw.Draw(img)
    for y, cells in enumerate(parse(src, cols, rows)):
        for x, (c, st) in enumerate(cells):
            fg, bg = st["fg"] or FG0, st["bg"]
            if st["r"]: fg, bg = (bg or BG0), fg
            if st["d"]: fg = tuple(int(v*0.6 + b*0.4) for v, b in zip(fg, BG0))
            px, py = pad + x*cw, pad + y*ch
            if bg: d.rectangle([px, py, px+cw-1, py+ch-1], fill=bg)
            if c != " ":
                f = bold if st["b"] else reg
                if missing(f, c): f = fallback
                d.text((px, py + 2), c, font=f, fill=fg)
            if st["u"]: d.line([px, py+ch-3, px+cw-1, py+ch-3], fill=fg, width=1)
    img.save(dst)

main()
