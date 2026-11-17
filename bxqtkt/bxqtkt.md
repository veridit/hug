# Contributions: one-way GitHub intake (bxqtkt design note)

Status: proposal only. Jørgen has not approved it, and there is no code. Author: kestrel,
2026-11-17 14:50, at crab's request. The ruling this applies is Jørgen's, on bxqtkt:
**one-way intake**. When an issue is taken in, our ticket is created and the issue is
closed with a link to that ticket. The board stays the single truth, and nothing flows
back to GitHub later. (h176dp check 4, which pushed status back, was struck today as
superseded by this ruling.)

## What exists today, with evidence

- `hug task import github [OWNER/REPO] [--dry-run]` (4d2b85e, 2026-11-15) **mirrors**
  issues. It imports open *and* closed issues (`--state all --limit 5000`) and turns
  closed into done and not_planned into dropped. On re-import it three-way merges
  GitHub-side changes (huginn/internal/importer/github.go:39, cli/hooks.go:88). That
  is inbound sync, which the ruling also rules out.
- Nobody relies on the mirror. There are 0 `GH-` aliases among the 635 Statbus task
  files, and 0 `origin: "github…"` lines under ~/Projects/*/.hug/tasks. The Statbus
  run (42 issues) noted on h176dp left nothing behind.
- veridit/hug is public with issues on, and it has 0 issues so far
  (`gh api repos/veridit/hug/issues -f state=all` gives 0).
- Ticket file names change, but ids don't. bxqtkt alone has had 6 file names
  (`zzzzzzzzzzzy…` through `zzzzzzzzzzyh.bxqtkt.contributions-agent-feedback.md`).
  hug/tasks has 135 rename commits since 2026-11-10.

## 1. Intake: `hug task import github`

The command changes from a mirror into a triage step over **open issues only**.

```
hug task import github [OWNER/REPO]                 # the inbox: list only, writes nothing
hug task import github [OWNER/REPO] N [N...]        # preview the intake of these issues
hug task import github [OWNER/REPO] N... --yes      # take them in
hug task import github [OWNER/REPO] --dismiss N --reason "..." [--yes]
```

**Inbox (no numbers).** It lists the repo's open issues that have no ticket yet: number,
title, author, age and labels. An issue that links a fork ticket (see 2) is marked
`fork: OWNER/FORK ID`. Issues already taken in but still open, because intake stopped
halfway, are listed with `resume: hug task import github N --yes`.

**Preview (numbers, no `--yes`).** For each issue it prints the ticket it would create
and the **exact comment** it would post. Nothing is written and nothing is posted. This
applies bxqtkt's rule "Nothing is posted without the user seeing the exact text" to
our own side too. An agent shows the preview to the person before running `--yes`.

**Take (`--yes`)**, per issue, in this order. Each step can be repeated safely:

1. **Create the ticket** in todo. Title is the issue title. The body goes into Goal plus
   a Description section, and task-list checkboxes become acceptance items. This is
   today's WriteIssues rendering, so h176dp checks 1 and 3 carry over. Fields:
   `origin: "github:OWNER/REPO#N"` (the dedupe key, checked by LintOrigins), alias
   `GH-N`, labels copied. **Credit** goes in a first note:
   `From GitHub OWNER/REPO#N by @login, 2026-11-17`. If the body has a `Model:` line (a
   feedback issue), the note copies it. No assignee and no status are taken from GitHub.
2. **Commit and push hug/tasks**, so the link in the comment works when it is posted.
   If the push fails, it stops here. The ticket exists locally and the issue is
   untouched. A rerun skips step 1 because the origin is already on the board.
3. **Comment** on the issue: `Taken in as hug ticket ID: <stable link, see 3>. Thank
   you, @login. Work and discussion continue there.` The comment ends with a hidden
   marker `<!-- hug-intake: ID -->`, so a rerun finds the comment and does not post it
   twice.
4. **Close** the issue as `completed`. (GitHub has no "moved" reason. Closing as
   completed with the comment is the convention. Question Q3.)

A rerun on a half-done intake finishes the missing steps and nothing else. It never
reopens an issue, edits a comment, or posts ticket progress later.

**Dismiss.** It closes the issue as `not_planned`, with a comment holding the reason.
The preview shows that comment first. No ticket is created.

**Not imported:** closed issues, pull requests, issue comments after intake, and any
later edits on GitHub. After intake, the issue is history.

**Auth and tests.** Auth stays as h176dp check 5: the gh CLI token
(`gh auth token`) or `GITHUB_TOKEN`, and hug stores no token. Calls go through a small
REST client over net/http (list issues, get issue, comments, post comment, patch state),
with the base URL taken from `HUG_GITHUB_API` (default `https://api.github.com`).
Tests run that client against an httptest fake GitHub that records each request.
Nothing touches the real API. Today's `gh issue list` shell-out goes, because a test
cannot point it at httptest without a fake gh binary.

## 2. Fork tickets: `hug task upstream ID`

This is for someone who forked hug and wrote a ticket on the fork's own hug/tasks
branch.

```
hug task upstream ID            # preview the issue it would open
hug task upstream ID --yes      # open it
```

1. Find the upstream repo with `gh api repos/{fork}`, field `.parent.full_name`. If
   the repo is not a fork, it refuses with
   `fix: hug task upstream ID --repo OWNER/REPO`.
2. Check that the fork's hug/tasks is pushed and contains ID. If not, it stops with
   `fix: hug task sync`.
3. Open a **short** issue on upstream. The title is the ticket title. The body is the
   ticket's Goal, one stable link to the fork's ticket (keyed on id, see 3), and a
   marker line `hug-ticket: FORK_OWNER/FORK_REPO ID @SHA`. SHA is the hug/tasks commit,
   so intake reads the exact version that was proposed. The rest of the ticket is not
   copied, because the link holds it. The preview shows this text exactly, and nothing
   is posted without `--yes`.
4. Record `upstream: "github:OWNER/REPO#N"` on the fork's ticket, so a second run
   refuses with `fix: open https://github.com/OWNER/REPO/issues/N`.

**On our side**, the inbox marks such an issue as `fork:`. Intake then reads the fork's
ticket file at SHA: it fetches the tree, finds the file whose name has `.ID.` in it, and
fetches the blob. It imports the whole ticket: Goal, Purpose, Principle, Relevant
details and Acceptance. It does not import the fork's status, stage, assignee or notes.
Credit: the note names the issue's author and the fork's `model:` line, as
`From fork OWNER/FORK ticket ID @SHA by @login, model claude-…`.

Which id does our ticket get? The fork's ID if it is free on our board, otherwise a new
one with the fork's ID kept as an alias. Ids are random 6-character strings, so a
collision is unlikely, and keeping the same id makes both links read the same.
(Question Q5.)

## 3. Links keyed on the stable id

A link to the current file (`blob/hug/tasks/zzzzzzzzzzyh.bxqtkt.….md`) breaks at the
next rank change. The first candidate in the bxqtkt notes, a GitHub code-search URL for
the id on branch hug/tasks, **does not work**: GitHub code search only indexes the
default branch (main), never hug/tasks.

**Proposal: one small stub file per linked id**, at `.id/ID.md` on hug/tasks:

```
# bxqtkt: Contributions: agent feedback and GitHub intake…
Current file: [zzzzzzzzzzyh.bxqtkt.contributions-agent-feedback.md](../zzzzzzzzzzyh.bxqtkt.contributions-agent-feedback.md)
```

- The stable link is `https://github.com/OWNER/REPO/blob/hug/tasks/.id/ID.md`, which is
  one click away from the ticket. Its URL never changes.
- Stubs exist **only for ids linked from outside**: those created by intake and those
  sent upstream. They are not created for every task, which keeps churn to a few files.
- The store rewrites the stub in the same commit as any rename of that task, so the stub
  and the file cannot drift. Lint checks that every stub points at an existing file. A
  dropped task keeps its stub, because the link must still resolve.
- One file per id, instead of one index file, means two agents renaming different
  tasks never conflict on it.
- Stubs are never read as tasks. The store's walk already skips dot-folders
  (huginn/internal/task/store.go:79). A plain `id/` would not work, because
  subfolders are task groups (design/, later/, tui/ …).

Rejected options: a single index file (every rename on the board touches it, so it
conflicts in concurrent syncs), renaming task files to be id-only (a big change to rank
and slug order for one link), and a hug-served page (bxqtkt: no server).

## 4. Feedback (bxqtkt step 1, for completeness)

`hug feedback "…"` and an MCP `feedback` tool open an issue on veridit/hug through the
user's gh login, labelled `agent-feedback`. Paths, repo names and code are stripped
first, then the exact text is shown and `--yes` (in MCP, `confirmed: true`) is required.
It arrives in the same inbox and goes through the same intake as any issue. Nothing on
our side is special for feedback.

## 5. What is out

Two-way sync of any kind: no reopen, no pushing status, checks or notes, and no
re-import merges. Also out: importing closed issues, handling pull requests
(CONTRIBUTING covers those, see 7), a server or home page, and webhooks or scheduled
intake. Intake runs only when someone runs it.

## 6. Build tickets, once approved (bxqtkt check 2)

1. GitHub REST client plus httptest fake. It covers list open issues, comment (with
   the marker lookup) and close/dismiss, and supports `HUG_GITHUB_API`.
2. `import github` becomes intake: inbox, preview, `--yes` steps 1-4 that can each be
   repeated, and `--dismiss`. The mirror mode and its closed-issue import are removed.
   CHANGELOG notes the change.
3. Id stubs: `.id/ID.md` written on intake and rewritten on rename in the same commit,
   plus a lint check.
4. `hug task upstream ID` on forks, plus the fork-ticket read during intake.
5. `hug feedback` plus the MCP feedback tool (strip, preview, confirm).

They depend on each other in order 1 → 2 → 3, with 4 and 5 after 2.

## 7. CONTRIBUTING.md draft (bxqtkt check 3), for Jørgen's review

> ## How contributions land
>
> Issues are the easiest way in. When we take one, it becomes a ticket on our board and
> the issue is closed with a link to that ticket. Follow the work there.
>
> Pull requests are welcome as proposals. We read them, and usually rebuild the change
> ourselves so it fits the code and the tests, crediting you in the commit and the
> changelog. Sometimes we merge a pull request as it is. Either way, the contributor
> agreement above applies.
>
> Agents can send feedback with `hug feedback "…"`. You see the exact text before
> anything is posted.

## Questions for Jørgen

- **Q1. Replace the mirror.** `import github` stops importing closed issues and stops
  re-merging, and becomes the intake above. Nobody uses the mirror (0 GH- aliases
  anywhere). OK to remove it rather than keep both?
- **Q2. Preview, then `--yes`, also for our own intake comment?** Or may the
  maintainer side post without a preview?
- **Q3. Close reason.** Close a taken-in issue as `completed` with the link comment, as
  proposed. The other choice is `not_planned`, which some read as "rejected".
- **Q4. Id stubs** at `.id/ID.md` on hug/tasks, only for ids linked from outside. Is
  that the link you want, and is the folder name OK?
- **Q5. A fork ticket's id.** Keep the fork's id when it is free, or always mint ours?
- **Q6. CONTRIBUTING wording** in section 7: approve or edit.
