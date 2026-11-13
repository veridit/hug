# Bridge experiment, 2026-11-10

A file mailbox between the Claude desktop app agent and a CLI agent, plus a
Claude Code channel server that pushes new messages into a session. Built by
the app agent as a prototype; kept as design input for `hug msg` (messages).

What it showed: a mailbox works, waking is the hard part. Without a supervisor
that wakes agents, messages sat unread overnight.
