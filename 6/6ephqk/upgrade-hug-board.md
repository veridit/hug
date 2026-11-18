# 6ephqk check 4: hug upgrade on hug's own board (v0.7.45, 2026-11-17 18:49Z)

## lint before (hug task lint --summary)
```
167 findings (136 broken) in 5 codes:
  126  broken  old-layout        the file says status: where state: goes (a board from before the state: rename; hug upgrade moves it)  hug task lint --code old-layout
    9  broken  shipped-not-done  question: a release contains the task, but it is not done                                              hug task lint --code shipped-not-done
    1  broken  missing-file      a file: link names no file                                                                             hug task lint --code missing-file
   30  advice  missing-section   an open task lacks Purpose or Principle                                                                hug task lint --code missing-section
    1  advice  long-title        the title is longer than 80 characters                                                                 hug task lint --code long-title
next: hug task lint --code old-layout
```

## hug upgrade (preview, verbatim, 126+3 diff lines)
```
layout: nothing to do
tasks: would: 126 tasks moved to state: (was status:)
tasks: would: 3 waits on a person moved to waiting_for
variables: nothing to do
memory: nothing to do
6ephqk: status: doing -> state: doing
v3k6cp: status: done -> state: done
102bc0: status: done -> state: done
tzc9hb: status: done -> state: done
tb0dra: status: doing -> state: doing
mcvfbz: status: done -> state: done
9xfnwr: status: done -> state: done
9taa9d: status: todo -> state: todo
zrjgyr: status: done -> state: done
mqzpyg: status: done -> state: done
4emme8: status: done -> state: done
8bsx4y: status: done -> state: done
kx2ndm: status: todo -> state: todo
16dxsr: status: done -> state: done
rzbgqe: status: done -> state: done
rr378d: status: done -> state: done
4k35r1: status: done -> state: done
sfa8w4: status: done -> state: done
a1dx7s: status: todo -> state: todo
p277bp: status: todo -> state: todo
pbh9xq: status: done -> state: done
34r1ft: status: todo -> state: todo
7js5hw: status: done -> state: done
63tyrz: status: dropped -> state: dropped
66ex4y: status: todo -> state: todo
ta306h: status: todo -> state: todo
gav9nw: status: todo -> state: todo
hpqt99: status: todo -> state: todo
vk9td0: status: todo -> state: todo
2easb0: status: done -> state: done
h0t41j: status: todo -> state: todo
j0xa38: status: todo -> state: todo
zmy9rw: status: done -> state: done
3z7zrs: status: done -> state: done
n3ybbc: status: dropped -> state: dropped
6tj2b8: status: done -> state: done
5z0rdq: status: done -> state: done
s6ff5k: status: doing -> state: doing
h176dp: status: done -> state: done
c3hszg: status: todo -> state: todo
6assqk: status: todo -> state: todo
jrj1vq: status: todo -> state: todo
5g86n6: status: done -> state: done
npda3r: status: done -> state: done
wdkq16: status: done -> state: done
48hx44: status: done -> state: done
n7jbmx: status: todo -> state: todo
d634kr: status: todo -> state: todo
heype5: status: done -> state: done
x1t1t1: status: done -> state: done
065wwn: status: done -> state: done
gem59b: status: todo -> state: todo
qawmnb: status: done -> state: done
0sw0dd: status: todo -> state: todo
mkzcqr: status: done -> state: done
4tycwd: status: todo -> state: todo
mv4vye: status: done -> state: done
05r3pg: status: todo -> state: todo
jmx10s: status: todo -> state: todo
xxvm6t: status: done -> state: done
rx74bp: status: done -> state: done
nfv18p: status: done -> state: done
44w1ev: status: doing -> state: doing
ksv323: status: done -> state: done
6neax1: status: done -> state: done
et85h1: status: done -> state: done
m1c28r: status: done -> state: done
j2vwk0: status: done -> state: done
t709he: status: done -> state: done
zaec08: status: done -> state: done
me51r2: status: done -> state: done
ez50rb: status: done -> state: done
e4w9ya: status: todo -> state: todo
3xcesq: status: done -> state: done
zgr01t: status: done -> state: done
h9zghj: status: done -> state: done
z6kz20: status: done -> state: done
m8shdj: status: done -> state: done
jnaspy: status: done -> state: done
0hkaa5: status: done -> state: done
d8yx7y: status: done -> state: done
x325pp: status: done -> state: done
tpnd3n: status: done -> state: done
bxqtkt: status: doing -> state: doing
5dxzxe: status: todo -> state: todo
805p6y: status: dropped -> state: dropped
typq1e: status: todo -> state: todo
txjkrm: status: todo -> state: todo
0sbvnq: status: doing -> state: doing
fkpyg3: status: done -> state: done
scrxak: status: doing -> state: doing
bejd89: status: doing -> state: doing
3jtfw9: status: doing -> state: doing
47sxn2: status: doing -> state: doing
g9dg32: status: done -> state: done
at3qep: status: done -> state: done
n2hq8p: status: done -> state: done
zhvkmj: status: doing -> state: doing
vzghj8: status: doing -> state: doing
49n9kw: status: done -> state: done
kbc99j: status: done -> state: done
4vcwxk: status: done -> state: done
zd3x24: status: done -> state: done
1rca5g: status: done -> state: done
ggydj2: status: doing -> state: doing
4vxqp4: status: todo -> state: todo
x1ra0t: status: done -> state: done
5z4zt4: status: doing -> state: doing
89y0b7: status: doing -> state: doing
np2kk2: status: doing -> state: doing
vnyz8z: status: doing -> state: doing
cvbpdy: status: doing -> state: doing
jr55s8: status: doing -> state: doing
fcf4fg: status: todo -> state: todo
95ahtk: status: done -> state: done
r5r8c1: status: doing -> state: doing
p29rbz: status: done -> state: done
t3fqaw: status: doing -> state: doing
asr4t2: status: done -> state: done
gjmqhn: status: done -> state: done
456vyt: status: done -> state: done
6w0gve: status: done -> state: done
46ek1f: status: doing -> state: doing
ce3dmq: status: doing -> state: doing
4bwcgp: status: todo -> state: todo
3mavpt: status: todo -> state: todo
ggydj2: waiting_on "jhf: check 4: approve the Seid cleanup diff (attached: ggydj2-seid-dry-run.md). Seid's pre-commit and tools/evidence.py read .backlog, so they must move to hug first" -> waiting_for "jhf", waiting_on "check 4: approve the Seid cleanup diff (attached: ggydj2-seid-dry-run.md). Seid's pre-commit and tools/evidence.py read .backlog, so they must move to hug first"
np2kk2: waiting_on "jhf: check 4: open hug tui on Seid and say if it feels instant (copy median first frame 0.174 s, quit 0.211 s); checks 1-3 tick on green main CI 44126454244 (e582ab4)" -> waiting_for "jhf", waiting_on "check 4: open hug tui on Seid and say if it feels instant (copy median first frame 0.174 s, quit 0.211 s); checks 1-3 tick on green main CI 44126454244 (e582ab4)"
vnyz8z: waiting_on "jhf: check 4: open and quit hug tui on Seid (median first frame 0.26 s, quit 0.29 s measured on a copy) and say if you agree; checks 1-3 tick on green main CI 44124879024" -> waiting_for "jhf", waiting_on "check 4: open and quit hug tui on Seid (median first frame 0.26 s, quit 0.29 s measured on a copy) and say if you agree; checks 1-3 tick on green main CI 44124879024"
next: hug upgrade --apply
fix: hug upgrade --apply
exit 5
```

## hug upgrade --apply
```
layout: nothing to do
tasks: 126 tasks moved to state: (was status:)
tasks: 3 waits on a person moved to waiting_for
variables: nothing to do
memory: nothing to do
...
next: hug setup
exit 0
```
Commits on hug/tasks: abcf36b 'hug upgrade: status: -> state: in 126 tasks' (126 files, every -/+ pair status:X -> state:X, sorted values byte-identical, 0 other lines), 37aec93 'hug upgrade: waiting_for in 3 tasks' (ggydj2, np2kk2, vnyz8z: 3 files +6 -3).

## lint after
```
41 findings (10 broken) in 4 codes:
   9  broken  shipped-not-done  question: a release contains the task, but it is not done  hug task lint --code shipped-not-done
   1  broken  missing-file      a file: link names no file                                 hug task lint --code missing-file
  30  advice  missing-section   an open task lacks Purpose or Principle                    hug task lint --code missing-section
   1  advice  long-title        the title is longer than 80 characters                     hug task lint --code long-title
next: hug task lint --code shipped-not-done
```
old-layout 126 -> 0; every other code unchanged (shipped-not-done 9, missing-file 1, missing-section 30, long-title 1).

## hug upgrade again
```
layout: nothing to do
tasks: nothing to do
variables: nothing to do
memory: nothing to do
next: hug setup
exit 0
```
