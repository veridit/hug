#!/usr/bin/env bash
# Rewrite the TUI goldens a failed CI run reported as drifted, from that
# run's own screens. For when a local HUG_UPDATE_GOLDEN=1 run cannot finish
# (a loaded machine) or the drift shows on one platform only. Review the
# diff before committing, as with any golden update.
#
# Usage: scripts/ci-goldens.sh RUNID [JOB]
#        scripts/ci-goldens.sh --log FILE [JOB]   (a saved gh run view --log-failed)
#   RUNID  a GitHub Actions run of this repo, e.g. 44403710185
#   JOB    only that job's log, e.g. "test (ubuntu-latest)"; without it every
#          failed job must report the same screen for a path
# Run from the repository root; it writes huginn/internal/tui/<path>.
#
# Reads each block the TUI golden tests print,
#   golden drift <path>\nGOT:\n<screen>\nWANT:\n<golden>
# (tmux_test.go). go test indents every line after the first, blank lines
# included, and drops the message's final newline; gh prefixes job, step
# and a timestamp. Undone, the lines between GOT: and WANT: are the screen
# with its own final newline; WANT gets its newline back. Before writing
# anything, every WANT block, undone the same way, must equal the committed
# golden byte for byte: proof the decoding holds for this log. Otherwise
# nothing is written. Needs bash 3.2+ and any POSIX awk (mawk, BSD, gawk).
set -euo pipefail
export LC_ALL=C

usage='usage: scripts/ci-goldens.sh RUNID [JOB] | --log FILE [JOB]'
if [ "${1:-}" = --log ]; then
	src=${2:?$usage}
	job=${3:-}
	label=$src
else
	run=${1:?$usage}
	job=${2:-}
	label="run $run"
fi
pkg=huginn/internal/tui
if [ ! -d "$pkg" ]; then
	echo "ci-goldens: no $pkg here: run it from the repository root" >&2
	echo "fix: cd \"\$(git rev-parse --show-toplevel)\" && scripts/ci-goldens.sh $*" >&2
	exit 2
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
if [ -n "${run:-}" ]; then
	src=$work/log
	if ! gh run view "$run" --log-failed > "$src"; then
		echo "ci-goldens: could not read the failed log of run $run" >&2
		echo "fix: gh run view $run --log-failed | head" >&2
		exit 2
	fi
fi

# Split into $work/N.path, N.got, N.want, one N per drift block.
awk -v job="$job" -v out="$work" '
	function flush(name,   i, f) {
		f = out "/" name
		printf "" > f
		for (i = 1; i <= cnt; i++) printf "%s%s", buf[i], (i < cnt ? "\n" : "") > f
		close(f)
	}
	function indented(s) { return substr(s, 1, indent) == pad }
	BEGIN { FS = "\t"; n = 0; mode = "" }
	{
		if (job != "" && $1 != job) next
		line = $0
		sub(/^[^\t]*\t[^\t]*\t/, "", line)                                           # job, step
		sub(/^[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9:.]*Z /, "", line)  # timestamp
		if (mode == "got" || mode == "want") {
			if (indented(line)) {
				rest = substr(line, indent + 1)
				if (mode == "got" && rest == "WANT:") { flush(n ".got"); mode = "want"; cnt = 0; next }
				buf[++cnt] = rest
				next
			}
			if (mode == "got") {
				printf "ci-goldens: drift %d (%s) ends before its WANT: line\n", n, p > "/dev/stderr"
				bad = 1; exit
			}
			flush(n ".want"); mode = ""
		}
		if (mode == "head" && match(line, /^ *GOT:$/)) {
			indent = RLENGTH - 4; pad = substr(line, 1, indent); mode = "got"; cnt = 0; next
		}
		if (match(line, /golden drift [^ ]+$/)) {
			n++; mode = "head"
			p = substr(line, RSTART + 13)
			print p > (out "/" n ".path"); close(out "/" n ".path")
		}
	}
	END {
		if (bad) exit 3
		if (mode == "want") flush(n ".want")
		else if (mode != "") { printf "ci-goldens: drift %d (%s) is cut off\n", n, p > "/dev/stderr"; exit 3 }
		print n > (out "/count")
	}
' "$src" || { echo "fix: gh run view ${run:-RUNID} --log-failed | grep -n -A3 'golden drift'" >&2; exit 3; }

n=$(cat "$work/count")
if [ "$n" -eq 0 ]; then
	echo "ci-goldens: $label has no 'golden drift' block${job:+ in job $job}" >&2
	echo "fix: gh run view ${run:-RUNID} --log-failed | grep -n 'golden drift'" >&2
	exit 1
fi

bad=0
: > "$work/seen" # path<TAB>first drift index (bash 3.2 has no associative arrays)
for i in $(seq 1 "$n"); do
	p=$(cat "$work/$i.path")
	dst=$pkg/$p
	if [ ! -f "$dst" ]; then
		echo "ci-goldens: $dst is not in this tree" >&2
		bad=1
		continue
	fi
	printf '\n' >> "$work/$i.want"
	if ! cmp -s "$work/$i.want" "$dst"; then
		echo "ci-goldens: the WANT block of $p differs from $dst: this tree is not the run's commit, or the log format changed" >&2
		bad=1
	fi
	first=$(awk -F'\t' -v p="$p" '$1 == p { print $2; exit }' "$work/seen")
	if [ -z "$first" ]; then
		printf '%s\t%s\n' "$p" "$i" >> "$work/seen"
	elif ! cmp -s "$work/$i.got" "$work/$first.got"; then
		echo "ci-goldens: two jobs report different screens for $p" >&2
		bad=1
	fi
done
if [ "$bad" != 0 ]; then
	echo "ci-goldens: wrote nothing" >&2
	echo "fix: git checkout <the run's commit>, or name one job: scripts/ci-goldens.sh ${run:-RUNID} \"test (ubuntu-latest)\"" >&2
	exit 4
fi

while IFS=$'\t' read -r p i; do
	cp "$work/$i.got" "$pkg/$p"
	echo "wrote $pkg/$p"
done < "$work/seen"
