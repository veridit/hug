# Scratch-only recall experiments

`recalleval` is an opt-in evaluation command, not a production recall hook. It requires an explicit database under `$JCODE_SCRATCH_DIR/recall3`. Never use a live database or commit replay messages, transcripts, memories, judgments, or benchmark data.

Build with `go build -o "$JCODE_SCRATCH_DIR/recall3/recalleval" ./cmd/recalleval`.

## Chronological replay

Supply an independent SQLite backup for each variant and split. Clear injection history only in that backup. Input is a JSON array with `id`, `home`, `session`, `message`, `cwd`, `split`, and RFC3339 `timestamp`. Binding uses the original home and working directory. Output is JSONL with the response, effective scopes, error, and warm elapsed milliseconds. Original self-output suppression remains active after heuristic rewriting.

```sh
"$JCODE_SCRATCH_DIR/recall3/recalleval" \
  -db "$JCODE_SCRATCH_DIR/recall3/baseline-development.db" \
  -input "$JCODE_SCRATCH_DIR/recall3/replay.json" \
  -split development -variant baseline
```

Variants: `baseline`, `fts`, `semantic`, `no-community`, `no-reserve`, `no-gates`, `no-semantic-gate`, `no-relative-gate`, `heuristic`, `recency`, `normalised-time`, `union`, and `tool-search`. `llm-rewrite` consumes already rewritten input through the unchanged baseline path. The external transport, blind judge, passage implementation, and private statistical analysis are separate scratch artifacts.

`no-gates` removes semantic and relative injection gates, not the original minimum score. Pure lexical and semantic variants remove the other score contributions but retain some unused concept-lookup overhead. `union` requests 20 candidates without injection gates or session recording, for a subsequent fixed-union reranker. `tool-search` requests 30 without those gates. Normalized temporal ranking preserves original score eligibility and scope restrictions before changing score scale. It uses update time, not access time, and has no importance term.

## External round retrieval

`-benchmark` treats `-db` as an output directory. Input contains question `id`, `question`, `date`, evidence `sessions`, and `rounds`. Each round has `key`, `session`, `date`, `body`, and an `evidence` boolean. Dates use `2006/01/02 (Mon) 15:04`. Evidence labels are never indexed or used by search.

Each question receives an independent native-memory index and `MUNINN_HOME`. `baseline.db` preserves the unaugmented index before keyphrase titles are added. Embedding caching only avoids repeated indexing, not query calls. `retrieval` is an ungated minimum-score-zero reference. `keyphrases` adds the twelve most frequent non-stopword terms longer than three characters, with lexical tie-breaking. It is not LLM fact extraction.

The command emits ranked round IDs and timings, not benchmark scores. Analysis must distinguish fractional gold recall from LongMemEval's official all-gold recall, exclude abstention cases from positive recall, and state its DCG convention. Do not compare this harness's gated nomic/FTS results directly with paper BM25 or Contriever results without describing dataset, split, unit, model, gate, and metric differences.

No experiment changes default production options. A successful build or targeted test is not a shipping gate.
