# Layout 3 catch-up

`hug upgrade` previews a board migration. `hug upgrade --apply` changes this board, not this machine's hook or memory configuration. Ordinary local migration works offline and does not fetch.

When sync has fetched a remote board with a different layout and this clone has unpushed commits, it refuses to rebase and names `hug upgrade`. The preview lists this clone's first-parent commits. Explicit catch-up fetches the remote again before replaying. A network failure leaves the board and local commits unchanged, with an actionable retry.

## Replay

Replay runs in a private worktree. Each first-parent commit contributes its complete delta against parent 1, including a merge commit's conflict-resolution edits, exactly once. Merged side-branch commits are not separately replayed. Each replay keeps its original message and author. A pure layout move with no translated delta can be omitted. Mixed migration commits retain their other content. Nothing is pushed by upgrade.

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

The shared resolver is used by catch-up here. Normal same-layout sync is a separate integration tracked by 8zwy6r.
