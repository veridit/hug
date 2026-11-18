#!/bin/bash
# 911j70 compare sheets from current main (kikazaru): real binary in tmux,
# a read-only snapshot of hug's own board. VARIANT: before | after-A | after-B.
# Keys mirror puppy's mock/capture.sh (overview, list, board cursors).
set -u
VARIANT=$1
OUT=/Users/jhf/.jcode/scratch/911j70-sheets-out/out/$VARIANT
BIN=/Users/jhf/.jcode/scratch/911j70-sheets-out/bin/hug
REPO=/Users/jhf/.jcode/scratch/911j70-board
HOMEDIR=/Users/jhf/.jcode/scratch/911j70-sheets-out/home
mkdir -p "$HOMEDIR"
case $VARIANT in
  before) MOCK="" ;;
  after-A) MOCK="MOCKGRID=1 MOCKGRID_ORDER=A" ;;
  after-B) MOCK="MOCKGRID=1 MOCKGRID_ORDER=B" ;;
esac
SOCK=kz911j70-$$
tmux -D -L "$SOCK" -f /dev/null &
SERVER=$!
sleep 1
T() { tmux -N -L "$SOCK" -f /dev/null "$@"; }
T set-option -s exit-empty off
for size in 100x30 160x45; do
  W=${size%x*}; H=${size#*x}; D=$OUT/$size; mkdir -p "$D"
  S=s$W
  T new-session -d -s $S -x $W -y $H -c $REPO "env HOME=$HOMEDIR XDG_CONFIG_HOME=$HOMEDIR/.config $MOCK TERM=xterm-256color COLORTERM=truecolor COLORFGBG='15;0' CLICOLOR_FORCE=1 HUG_USER=jhf GOMAXPROCS=2 nice -n 19 $BIN tui; sleep 30"
  T set-option -g status off; T resize-window -t $S -x $W -y $H
  tgt=$S:0.0
  send() { T send-keys -t $tgt -l "$1"; sleep 0.4; }
  key() { T send-keys -t $tgt "$1"; sleep 0.4; }
  snap() { sleep 0.8; T capture-pane -p -t $tgt > "$D/$1.txt"; T capture-pane -p -e -t $tgt > "$D/$1.ansi"; }
  sleep 4
  snap overview-initial
  # overview: NEXT UP (Tab x3 from ASKED), 3 rows down, then L to PRIORITY
  key Tab; key Tab; key Tab; send jjj
  # L until the footer names priority: the shown columns differ per
  # variant and section (a column every row shares is not drawn, 1bbe96).
  for i in 1 2 3 4 5 6 7 8 9; do
    send L
    T capture-pane -p -t $tgt | grep -q "order by priority" && break
  done
  snap overview-cursor
  send vl; send "<"; for i in $(seq $((H/2))); do T send-keys -t $tgt -l j; sleep 0.03; done; sleep 0.6; snap list-cursor
  send vb; key Tab; send jjjjjjjjj; send kkk; snap board-cursor
  send q; sleep 0.5; T kill-session -t $S 2>/dev/null
done
T kill-server 2>/dev/null
wait $SERVER 2>/dev/null
