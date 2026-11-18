# Layout 3 catch-up

`hug upgrade` previews a board migration. `hug upgrade --apply` changes this board, not this machine's hook or memory configuration. Ordinary local task-file migration works offline and does not fetch. Branch-name migration is separate, as described below.

When sync has fetched a remote board with a different layout and this clone has unpushed commits, it refuses to rebase and names `hug upgrade`. The preview lists this clone's first-parent commits. Explicit catch-up fetches the remote again before replaying. A network failure leaves the board and local commits unchanged, with an actionable retry.

## Replay

Replay runs in a private worktree. Each first-parent commit contributes its complete delta against parent 1, including a merge commit's conflict-resolution edits, exactly once. Merged side-branch commits are not separately replayed. Each replay keeps its original message and author. A pure layout move with no translated delta can be omitted. Mixed migration commits retain their other content. Catch-up replay itself pushes nothing. The separate branch-name migration below can rename the corresponding remote refs.

Every changed file is translated to layout 3 before its three-way merge. Rank changes become `rank:` edits. A successful complete replay is installed only if the live board's HEAD and clean status are still unchanged. A failed replay leaves the original HEAD and all local commits intact.

The reverse case is supported too: when this clone moved first and the fetched board still uses layout 2, the same translator prepares the entire remote base in the private worktree before replay. Task paths, attachment paths and `hug.toml` become coherent together in the first retained local commit. No uncommitted transform is installed. Remote-only content, modes, aliases and `updated:` values survive this preparation. Ambiguous paths or unreadable tasks stop before any live change.

## Task values and append-only content

Notes, History, requests, answers and independent checks retain both sides. Explicit timestamps compare real instants, including timezone offsets, with stable board-first ties. Untimed prose has no recoverable chronology. It uses deterministic board-before-local order, preserves order within each side, and retains each distinct append once. No timestamp is invented and this fallback is not a claim of recovered time order.

For a single-valued field changed differently on both sides, the board's value remains canonical. A visible note retains this clone's value, author, timestamp and source commit, and upgrade prints a summary line.

For competing resolutions of the same request, the board's resolution remains canonical. This clone's resolution is retained verbatim in a labelled append-only Notes entry, including its resolver and timestamp. The note links to a stable identity derived from the original request's time, author and text, not its potentially renumbered display number. It is not a second open request. Repeating the same merge does not append the retained resolution again.

## Attachments and configuration

For a true conflict in a non-task file, the board's bytes stay at the original path. This clone's exact original bytes are retained at `ORIGINAL.local-FULLCOMMIT`, with the local executable mode. Upgrade prints both paths and the full source commit. A recovery config is not an active config and a recovered attachment is not a task file. Identical changes need no duplicate. An already present identical recovery with identical mode is reused. A differing recovery collision stops safely rather than overwriting either copy.

## Explicit limitations and safe stops

A file hug cannot parse stops catch-up and names its original source file and commit. A task deleted remotely but edited locally also stops: no resurrection or automatic drop is attempted. The error identifies the task, original commit and original file so its exact content remains recoverable. A tested explicit deletion-preservation policy is required before that case can proceed automatically. Recovery-file collisions and unsafe tree entries also stop without installing partial replay.

## Ordinary sync after migration

Once both clones use layout 3, divergent ordinary `hug task sync` uses this same per-file resolver and private replay, without invoking upgrade. The detached background pass uses it too. Fast-forward and already-ahead paths retain their normal Git behavior. All divergent task-board changes use semantic merging, even if Git could text-merge different lines of one scalar section into a hybrid value.

If both clones still use the old layout, ordinary sync retains its existing same-layout Git behavior and never translates their files implicitly. Migration still requires explicit `hug upgrade --apply`.

Independent same-task appends converge on both clones exactly once after A-sync, B-sync, A-sync. Competing scalar values keep the board's value and preserve this clone's value in a visible note. Foreground sync prints one `sync:` summary line per retained competing value to stderr. Background sync writes those lines to its sync log. Summaries are emitted only after the entire replay is installed, never for a private partial replay that later fails. Unreadable files and the safe-stop cases above preserve every original local commit and live HEAD here too.

## Project memory branch and worktree

The board-local preview includes:

```
layout: would: .muninn-notes/ moves to .hug/memory/, branch muninn-notes becomes hug/memory
```

`hug upgrade --apply` moves this repository's memory branch and worktree using the same shared Git branch-name migration as tasks. It preserves the memory branch HEAD, complete committed contents and modes. Linked code worktrees use the main worktree's one `.hug/memory/` checkout. Memory JSONL keeps its own existing merger, never the task YAML resolver. This step does not authorize changes to native memories, hooks or MCP configs. Those still require `--machine`. Ordinary memory use never creates an empty canonical branch beside detected old data or implicitly performs this migration.

Both old and new local branches, an occupied destination, dirty source or foreign checkout are refused without discarding either side. Known old memory data only in a remote-tracking ref is not implicitly bootstrapped: upgrade refuses, names the actually observed ref, and offers a quoted `git worktree add -b muninn-notes ...` command followed by `hug upgrade`. The command is offered only when both local branch names and the intended checkout paths are safe and the old branch is not registered elsewhere. Its final `hug upgrade` is a preview, not permission to apply. After attachment, inspect the preview and run `hug upgrade --apply` explicitly.

## Branch-name migration network failures

When a repository branch-name move needs the remote, shared migration probes the relevant refs before changing local refs or paths. An unreachable remote stops at that point. Remote creation of the new name and deletion of the old name use one atomic push with leases on both observed names. A failed local move or rejected remote push restores the original local branch, checkout path and config. It never independently creates the new remote ref and then tries a second destructive deletion.

A lost push acknowledgement is not proof that the remote stayed unchanged. A bounded follow-up `git ls-remote` observes both names. If the old name is absent and the new name equals the expected local commit, the migration is verified as successful and the local move is retained. If the push acknowledgement and the verification both fail, the error explicitly says remote outcome **UNKNOWN**, names both full refs and expected OIDs, and distinguishes verified local restoration from a local rollback that could not be verified. All original commits remain recoverable. It never claims an unobserved remote outcome or blindly compensates against another writer.

The fix first runs an exact, nonmutating `git ls-remote --heads ...` for both names, then `hug upgrade` to retry. Retry reconciles current refs before any mutation: the expected new OID with no old name completes the local move without another remote write; the expected old OID with no new name permits normal migration; both names, an unexpectedly advanced new OID, or an unreachable remote stop safely. An already-upgraded remote ancestor can still accept a second clone's retained local commits without rewriting or rewinding that remote. Repeated successful apply is a no-op. No marker or index file records this reconciliation state.
