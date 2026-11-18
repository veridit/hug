#!/bin/bash
# capture.sh BINARY PREFIX: real-binary tmux captures of the copied hug board.
# Writes out/PREFIX/WxH/NAME.{txt,ansi}. Keys mirror tmux_test.go.
set -u
BIN=$1; PREFIX=$2
OUT=/Users/jhf/.jcode/scratch/grid-mock/out/$PREFIX
REPO=/Users/jhf/.jcode/scratch/board-repo
SOCK=gridmock-$$
T() { tmux -N -L $SOCK -f /dev/null "$@"; }
for size in 100x30 160x45; do
  W=${size%x*}; H=${size#*x}; D=$OUT/$size; mkdir -p $D
  S=s$W
  tmux -L $SOCK -f /dev/null new-session -d -s $S -x $W -y $H -c $REPO "env ${EXTRA:-} TERM=xterm-256color COLORTERM=truecolor COLORFGBG=15\;0 CLICOLOR_FORCE=1 HUG_USER=jhf GOMAXPROCS=2 nice -n 19 $BIN tui"
  T set-option -g remain-on-exit on; T set-option -g status off; T resize-window -t $S -x $W -y $H
  tgt=$S:0.0
  send() { T send-keys -t $tgt -l "$1"; sleep 0.35; }
  key() { T send-keys -t $tgt "$1"; sleep 0.35; }
  snap() { sleep 0.6; T capture-pane -p -t $tgt > $D/$1.txt; T capture-pane -p -e -t $tgt > $D/$1.ansi; }
  sleep 2.5
  snap overview-initial
  # view line, v pressed
  send v; snap viewline-pending; sleep 0.53; T capture-pane -p -e -t $tgt > $D/viewline-pending-blink2.ansi
  key Escape
  # overview: NEXT UP section (Tab x3 from ASKED: IN PROGRESS, WAITING, NEXT UP), 3 rows down, column 3 (priority)
  key Tab; key Tab; key Tab; send jjj; send "${COLKEYS:-LLLL}"
  snap overview-cursor
  send vl; send "<"; for i in $(seq $((H/2))); do T send-keys -t $tgt -l j; sleep 0.03; done; sleep 0.5; snap list-cursor
  send vb; key Tab; send jjjjjjjjj; send kkk; snap board-cursor
  send v; snap board-viewline-pending
  send o; snap after-vo
  send q; sleep 0.5; T kill-session -t $S 2>/dev/null
done
T kill-server 2>/dev/null
