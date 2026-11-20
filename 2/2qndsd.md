---
id: 2qndsd
title: "PNG screenshots open inside the file frame"
state: "done"
type: bug
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzov"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T20:54"
---

## Goal
A .png or .jpg attachment opens inside the same frame as other files: crumb and note on top, rule, the image, rule, keys at the bottom, so it is clear where the ticket ends and the screenshot begins.

## Purpose
Jørgen 2026-11-18 on 5z4zt4: when viewing something everything else goes away, with lines at top and bottom so a screenshot is seen inside the ticket. z82qsy framed text and .ans files, but images still open full screen through imageView.

## Principle
One viewer frame for every file kind. The terminal draws the image (iTerm2/kitty) inside the frame's rows, sized to fit; elsewhere the one-line fallback sits in the frame.

## Relevant details
z82qsy ef9c060f added the frame (glance.go, render.go); imageView draws PNGs full screen outside it. Jørgen's terminal: iTerm2 (OSC 1337).

## Depends on

## Acceptance
- [x] A PNG opens with the crumb, the note, a top rule, the image within the frame rows, a bottom rule and the keys (golden for the fallback; tmux capture in iTerm2 attached to this task).
- [x] n/p moves between images and text files with the same frame (test).

## Notes

macaque (claude-opus-5-5) is working this stacked on z82qsy at crab's direction (13:16Z), branch 2qndsd from origin/ci/z82qsy-rebased 161381f3. hug task take refuses ('2qndsd is blocked by z82qsy') until z82qsy is done, so the dependency stays.

crab 13:17Z: removed the dependency on z82qsy so macaque can take it. The real ordering is in git: branch 2qndsd stacks on z82qsy's commits, and crab lands them in order.


file:tmux-iterm-image-frame.txt (826 bytes, sha256:e2165d8a) tmux capture-pane, real hug tui from 499f9220 at 100x24, TERM_PROGRAM=iTerm.app, Space on board.png: crumb, note, rule, 17 blank rows the image is drawn over, rule, keys
file:tmux-iterm-raw-overlay.txt (430 bytes, sha256:275af325) pipe-pane bytes: OSC 1337 drawn at row 4, 100x17 cells, inside tmux passthrough
file:tmux-n-to-text.txt (818 bytes, sha256:c63d4e09) n from the image to notes.txt: same frame, 2 of 2
macaque 499f9220 (branch 2qndsd, ci/2qndsd on 161381f3). Check 1: TestImageOpensInTheFrame golden testdata/detail/image-frame.txt (fallback at 80 and 140: crumb 'HUG TASKS · list › 000002 › compiler/000002/board.png · 67 B · 1 of 3', note 'screenshot of the board', rule, 'image, 67 B: this terminal draws no images (iTerm2 and kitty do): Enter opens it', rule, keys); TestImageDrawnInTheFrameRows: iTerm2/kitty/tmux overlay '\x1b7\x1b[4;1H...\x1b8' after the frame, OSC 1337 width=100;height=17, kitty a=T,f=100,C=1,r=17; frameView holds no image bytes. Real binary in tmux at 100x24, TERM_PROGRAM=iTerm.app: attached tmux-iterm-image-frame.txt and tmux-iterm-raw-overlay.txt (stream '^[7^[[4;1H^[Ptmux;^[^[]1337;File=name=Ym9hcmQucG5n;size=742;inline=1;width=100;height=17;preserveAspectRatio=1'). The picture on a real iTerm2 screen is for Jørgen to see.
Check 2: TestStepImagesAndTextInOneFrame (n/p through board.png, rows.ans, shot.png in kitty: same crumb+frame, 'n of 3', image drawn only on .png, kitty delete '\x1b_Ga=d,d=A\x1b\\' on line 0 under a text file); TestLeavingAnImageRepaints (n or u off an image: tea.ClearScreen once, same image again: none). tmux: n to notes.txt gives 'HUG TASKS · overview › 2g4zbs › 2/2g4zbs/notes.txt · 18 B · 2 of 2' (tmux-n-to-text.txt), stream holds one \x1b[2J. Fail first on 161381f3: 'TestImageOpensInTheFrame: at 80 Space on an image did not open it in the frame', 'TestImageDrawnInTheFrameRows: ... the image is not drawn at the frame's first row (4)', 'TestStepImagesAndTextInOneFrame: crumb ... list › 000002 ...'; repaint test without the eraseImage call: 'the image shown is not recorded as drawn'. Full suite: all 34 packages ok, e.g. 'ok  github.com/veridit/hug/huginn/internal/tui 140.106s'.
CI 44316906824 on 499f9220: attempt 1 failed only test (ubuntu-latest), huginn/internal/cli TestBackgroundSyncParallelNotesAllLand 'sync_test.go:92: background pass still running after 30 s' (known flake d2n0h7, package untouched by this diff); attempt 2 (rerun failed): completed success, all 6 jobs success. Report: /Users/jhf/.jcode/scratch/2qndsd/REPORT.md
crab review 15:20 local: accepted. 499f9220 on 161381f3 (now on main). Diff: picture.go/render.go/glance.go/tui.go + 2qndsd_test.go + golden image-frame.txt; failing-first 4 tests quoted in REPORT. CI 44316906824 6/6 on attempt 2 (attempt 1 ubuntu: d2n0h7 sync flake, package untouched). Queued after the current landings. Open: scale small images up to the frame, or natural size up to it; Jørgen's call at check after release.
Landed on main via 56e2cc6f (stack rjk44w c2df86c4, jj7jpz 215df154, 2qndsd 56e2cc6f; CI 44332698559 exact SHA 6/6). Ships in the next release.
landed 56e2cc6 on main 2026-11-18: tui: images open inside the file frame (2qndsd)
shipped in v0.7.65 (56e2cc6)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T14:18 todo -> doing/active (take)
- 2026-11-18T15:11 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T14:18; waiting_asked_by: macaque; waiting_on: crab review; waiting_since: 2026-11-18T15:11
