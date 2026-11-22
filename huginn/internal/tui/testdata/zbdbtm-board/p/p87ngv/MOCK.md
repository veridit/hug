# Mock: inline marks and history.sdd (p87ngv check 5)

Task in the mock: **w7tq3k** "Lint warns when an acceptance check reads like a question". It is made up, and no product code was touched.
Files: `before.md` (2 open marks) → `after.md` (no marks) + `history.sdd`; `cli.txt`; `tui-100x30.txt` (7 frames, 30 rows × ≤100 cols each).

## Syntax rules the files follow
1. A mark is a thing: space-dot-Capital (`.Request`, `.Question`, `.Comment`; a reply is `.Answer`) followed by `.lowercase:value` properties: `.by .to .at .msg .prev .on .in`.
2. **Space-dot-Capital starts a new thing in the run**: ` .Question ... .Answer ...` is a question with its answer.
3. **`:.` + Capital is a constructor value**: `.prev:.Sentence`. Such a value stands alone, so in `.prev:.Sentence .msg:"..."` the `.msg` belongs to the mark. If the value needs properties, use parentheses: `.prev:(.Sentence .n:2)`. Never write `.prev:.Sentence .n:2`.
4. **Ending:** everything that continues a run starts with a dot (`.x:`, `:.`, or space-dot-Capital). The first token that starts any other way ends the run, and prose resumes.
5. Anchor: `.prev:.Sentence | .Word | .Paragraph | .Item` is the unit just before the mark. With no `.prev`, the mark means `.prev:.Sentence`. `.on:"words"` narrows to a span inside that unit (used once: mark 2). `.in:"NAME#line"` points into an attachment (cli.txt §9).
6. Strings: `"..."`. Text that is hard to quote goes in `"""..."""`, which may span lines. The closer is the first `"""` not followed by another `"`, so a quote may touch it (`nor "Bør""""`). **This is hug's rule, pending Seid**: Seid's SYNTAX.md has no triple quotes yet. Shown in before.md (the answer) and history.sdd (the answer, plus a two-line resolve account).
7. **One form for names and times, in the ticket and history.sdd alike: bare.** `.by:jhf .at:2026-11-19T09:40Z`, as the agreed syntax in p87ngv's details writes them. Prose is always quoted (`.msg .on .was .now .field`). Open question 1 covers whether Seid allows the bare forms.
8. history.sdd: one `.Resolved .at .by .field [.check] .was .now .msg .marks: seq <the run, verbatim> end` per resolve, appended. `.msg` is the resolver's account of what changed.

## What each frame shows (tui-100x30.txt)
1. **Calm open marks.** Underline marks the text a mark is about. A dim `↳ N kind · who time · msg` line goes under it. The kind is a word, never a colour. There are no background fills and only one cue per meaning (qh9ep1 check 9). The raw `.Request ...` text is not drawn.
2. **Jump key `'`** (`"` goes back). The cursor is `>`, and the dim lifts on that one mark. Nothing else changes.
3. **Mark mode `m`.** The selection is bold, and bold appears only here. Tab steps the unit (sentence → word → paragraph → item). H/L make a free span (`.on`). The footer lists `c ? r`.
4. **`r` request.** The input opens where the ↳ line will appear. A dim `writes:` line previews the exact text.
5. **`?` question on a span of check 2.** Records `.prev:.Item .on:"Should, Can or Do"`.
6. **Answer input under the mark.** You open the task from ASKED. The cursor is on the question, and `a` answers under it. The check's own text is not changed.
7. **After both resolves.** Clean prose, and FILES lists history.sdd. This frame matches after.md.

## Checks run (scratch tools in `tools/`, not attached)
- Markdown, `pandoc 3.9.0.2`, niced, `-f commonmark|gfm|markdown -t html` on before.md and after.md: all 6 exit 0. Inside the mark runs: **0 HTML elements** in all 3 flavours. Marks stay plain text in the paragraph or list item they follow. With commonmark and gfm, the plain-text output holds both runs byte-identical. Pandoc's default `markdown` (smart quotes) curls the quotes in its *output* (`.msg:“Make it...”`, `“““English only...`). `markdown-smart` keeps them verbatim (1/1).
- Mark grammar (`tools/opencheck.py`): before.md has 2 runs, both ok. after.md has **0**. cli.txt has 10 runs and tui-100x30.txt has 3, all ok. The rule 3 probes give `.prev:(.Sentence .n:2)` ok, while `.prev:.Sentence .n:2` → `.Request: unknown properties ['n']`. history.sdd → `2 record(s), Open-tier structure ok`.

## Open questions (not decided here)
1. **Can Seid read this file at all?** Today `seid check` rejects the Open-tier example from seid-lang's own Seid-Data-Files.md (`.Dog .sleeping:True` → `1:1: unexpected token: .Dog`). It also rejects history.sdd (`2:1: unexpected token: .Resolved`). In Seid code, `.by:jhf` is `Unknown variable: jhf`, and `.at:2026-11-19T09:40Z` is `unexpected token: :`. The quoted forms typecheck. Keep bare names and times (rule 7) and ask Seid for words and a time literal, or quote them in both places?
2. **Suffix and tier.** In Seid-Data-Files.md, `.sdd` is the *Closed* tier (declared types), and Open is `.sdo` (suffix TBD). Jørgen chose history.sdd. Keep `.sdd` for Open data, rename it, or add a schema line (`: .Resolved ...`) to make it Closed?
3. **Mark numbers.** A mark has no id in the text. After mark 1 resolves, is the remaining mark still 2 or now 1? When several are asked at once, are they numbered in text order or in asking order (cli.txt §5, §7)?
4. **`--on` means two things.** On the CLI, `--on WORDS` *finds* the unit, while `.on:"..."` in the mark is a *span*. The mock adds `--span` to keep the words. Rename one of them?
5. **Keys.** `m` mark mode, `'`/`"` jump, `c ? r` inside mark mode, help on F1 there (since `?` is help elsewhere in the detail), and `R` resolve. All are proposals, and `m` is grab in the list view.
6. **history.sdd fields beyond the agreed `.at .was .now .marks`.** The mock adds `.by .field .check .msg`. Keep them?
7. **Should a resolve require `--now TEXT` or `--keep`** (the `no-change:` error in cli.txt)?
8. **Where an `.in:` mark on an attachment sits in the ticket.** The mock puts it at the end of ## Notes.
9. **Rendered Markdown inside `.msg`.** `*x*`, `` `x` ``, `<b>` and links render in a viewer (probe: `use *bold*` → `<em>bold</em>`). Smart-quote renderers curl the quotes. Accept this, or should hug escape when it writes?
