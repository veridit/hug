# Private capture replay adapter

This opt-in evaluation command is not installed as a capture hook. It reuses
production transcript parsing, secret detection, candidate validation and scoped
known-memory retrieval without applying capture actions.

Build its executable outside the repository, for example under
`$JCODE_SCRATCH_DIR/capture2/`. Keep **all real session/memory inputs, exported
text, model outputs, judgments, databases and reports there**, not in this public
repository. Read live source sessions only. Prefer historical memory snapshots
that exclude facts created or updated after the evaluated session began.

Modes:

* `transcript -home PATH -session ID`: production snapshot/journal reader with
  whole-message credential withholding, stable IDs and the persisted cwd.
* `secrets`: JSON array of strings on stdin, JSON array of blocked flags out.
* `validate`: production JSONL candidate parser, including rejection counts.
* `known`: JSONL `{ "Home": "...", "Cwd": "...", "Query": "..." }` queries,
  scoped top-12 memory results. Requires an explicit private `MUNINN_HOME` and
  indexes configured sources into that home. It does not write source memories.
* `embed`: JSON array of strings to the already-installed local Ollama
  `nomic-embed-text` model. No downloads, cloud calls or fallback model selection.

Remote extraction/judging is deliberately outside this adapter. Its caller must
filter inputs before egress, forbid tool access for model subprocesses, verify
the actual model, distinguish observation evidence from user intent, and retain
token/latency records. Absence of a credential-pattern match is not proof that
arbitrary text contains no sensitive information.

Replay comparisons are not end-to-end live-hook acceptance. Window cadence,
48-KiB message chunks, final partial windows, baseline retrieval settings,
carried summaries, and session-end passes must be reported separately. Never
ship a capture policy merely because it agrees with another extractor.
