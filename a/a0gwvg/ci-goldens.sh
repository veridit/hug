#!/usr/bin/env bash
# ci-goldens.sh RUNID [JOB]: rewrite the goldens a CI run reported as drifted.
#
# Reads `gh run view RUNID --log-failed`, takes each
#   golden drift <path>
#   GOT:
#   ...
#   WANT:
#   ...
# block from the TUI golden tests (tmux_test.go prints
# "golden drift %s\nGOT:\n%s\nWANT:\n%s"), and writes GOT to
# huginn/internal/tui/<path> in the current tree.
#
# Byte rule: the lines between GOT: and WANT: (log prefix and the GOT: line's
# own indent stripped) joined by "\n" are exactly the captured text: the
# "\n" before WANT: stands in for the capture's own trailing newline.
#
# Self-check: the WANT block of each drift is the committed golden; the script
# rebuilds it the same way and refuses to write anything unless every WANT is
# byte-equal to the file in the tree. JOB (e.g. "test (ubuntu-latest)") limits
# the log to one job; without it, all failed jobs must agree byte for byte.
set -euo pipefail
export LC_ALL=C

run=${1:?usage: ci-goldens.sh RUNID [JOB]
fix: ci-goldens.sh 44403710185 "test (ubuntu-latest)"}
job=${2:-}
pkg=huginn/internal/tui
[ -d "$pkg" ] || { echo "ci-goldens: no $pkg here (run in the repo root)" >&2; echo "fix: cd \$(git rev-parse --show-toplevel)" >&2; exit 2; }

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
gh run view "$run" --log-failed > "$work/log"

# Split into $work/N.path, $work/N.got, $work/N.want (one N per drift, per job).
awk -v job="$job" -v out="$work" '
	BEGIN { FS = "\t"; n = 0; mode = "" }
	{
		if (job != "" && $1 != job) next
		line = $0
		sub(/^[^\t]*\t[^\t]*\t/, "", line)                       # job, step
		sub(/^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9:.]+Z /, "", line)  # timestamp
		if (match(line, /golden drift [^ ]+$/)) {
			n++; mode = "head"
			p = substr(line, RSTART + 13)
			print p > (out "/" n ".path"); close(out "/" n ".path")
			next
		}
		if (mode == "head" && match(line, /^ *GOT:$/)) {
			indent = RLENGTH - 4; mode = "got"; cnt = 0; next
		}
		if (mode == "got" && line ~ ("^ {" indent "}WANT:$")) {
			flush(n ".got"); mode = "want"; cnt = 0; next
		}
		if (mode == "want" && (line ~ /^ *--- (FAIL|PASS)/ || line ~ /golden drift / || line ~ /^ *tmux_test\.go:/ || line ~ /^(FAIL|ok|PASS)/)) {
			flush(n ".want"); mode = ""
			if (match(line, /golden drift [^ ]+$/)) { n++; mode = "head"; p = substr(line, RSTART + 13); print p > (out "/" n ".path"); close(out "/" n ".path") }
			next
		}
		if (mode == "got" || mode == "want") {
			if (substr(line, 1, indent) ~ ("^ {" indent "}$")) line = substr(line, indent + 1)
			else if (line ~ /^ *$/) line = ""
			else { printf "ci-goldens: line without the %d-space indent in drift %d\n", indent, n > "/dev/stderr"; exit 3 }
			buf[++cnt] = line
		}
	}
	END { if (mode == "want") flush(n ".want"); print n > (out "/count") }
	function flush(name,   i, f) {
		f = out "/" name
		printf "" > f
		for (i = 1; i <= cnt; i++) printf "%s%s", buf[i], (i < cnt ? "\n" : "") > f
		close(f)
	}
' "$work/log"

n=$(cat "$work/count")
[ "$n" -gt 0 ] || { echo "ci-goldens: run $run has no 'golden drift' block${job:+ in job $job}" >&2; echo "fix: gh run view $run --log-failed | grep -n 'golden drift'" >&2; exit 1; }

# testing.decorate drops the message's final "\n" and indents every other line,
# blank ones included, so WANT's lines joined by "\n" plus one "\n" must equal
# the committed golden byte for byte. That proves the stripping on this run's
# own encoding before GOT, encoded the same way, is written anywhere.
bad=0
: > "$work/seen" # path<TAB>index of its first drift (bash 3.2: no assoc arrays)
for i in $(seq 1 "$n"); do
	p=$(cat "$work/$i.path")
	dst="$pkg/$p"
	[ -f "$dst" ] || { echo "ci-goldens: $dst does not exist in this tree" >&2; bad=1; continue; }
	printf '\n' >> "$work/$i.want"
	if ! cmp -s "$work/$i.want" "$dst"; then
		echo "ci-goldens: WANT of $p differs from $dst: this tree is not the run's tree, or the parser is wrong" >&2
		bad=1
	fi
	first=$(awk -F'\t' -v p="$p" '$1 == p { print $2; exit }' "$work/seen")
	if [ -z "$first" ]; then
		printf '%s\t%s\n' "$p" "$i" >> "$work/seen"
	elif ! cmp -s "$work/$i.got" "$work/$first.got"; then
		echo "ci-goldens: two jobs disagree on $p" >&2
		echo "fix: ci-goldens.sh $run \"test (ubuntu-latest)\"" >&2
		bad=1
	fi
done
[ "$bad" = 0 ] || exit 4

while IFS=$'\t' read -r p i; do
	cp "$work/$i.got" "$pkg/$p"
	echo "wrote $pkg/$p"
done < "$work/seen"
