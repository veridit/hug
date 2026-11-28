# hug MCP: memory_remember from a syncthing session landed in the lodgebook scope

Reported by the jcode agent in session `session_pawprint_1791205347390_cc7faf5c1c2b5149`
(working dir `/Users/jhf/Projects/syncthing`), 2026-11-28. hug v0.7.118.

## Verdict

`hug mcp` resolves "here" from **its own process cwd**, not from the calling
session. One `hug mcp` process is spawned per jcode *server*, and that server
hosts sessions from many directories. This server was started from lodgebook,
so every session it hosts writes "here" memories into lodgebook's scope.
The hooks (`hug mem hook jcode`) are correct, because they run per session in
the session's cwd: recall in this session showed syncthing memories all day.

## Process evidence (ps/lsof, 13:12 UTC)

    my shell 25246  <- jcode server 95335 "/Users/jhf/.jcode/builds/shared-server/jcode --provider auto serve"
                       parent 95275 "jcode --resume session_octopus_1794415386779_f5105e74343093e6"
                       parent 95242 "/bin/zsh -lc cd '/Users/jhf/Projects/Hermon/lodgebook' && jcode --resume session_octopus_…"
    server 95335 children: 92915 "/opt/homebrew/bin/hug mcp"  (started 2026-11-26 23:34:42)
    hug mcp 92915 cwd: /Users/jhf/Projects/Hermon/lodgebook

MCP config (`~/.jcode/mcp.json`): `"command": "/opt/homebrew/bin/hug", "args": ["mcp", "--jcode-home", "/Users/jhf/.jcode"]`.
Nothing in the call carries the session's working dir. Six `hug mcp` processes run
on this machine, one per jcode server, each with its server's cwd
(`/Users/jhf/Projects`, `Brovs/DevOps`, `truth_tribe_and_treasure`, `ssb/dbeaver`,
`GenderedGames`, `Hermon/lodgebook`).

Correct identity for this session: `cd ~/Projects/syncthing && hug mem project` gives
`scope: project:rc-82f3e8642441ebef`. Older syncthing memories use that scope.
In lodgebook, `hug mem project` fails: "project memories need an explicit layout
upgrade: .muninn-notes or branch muninn-notes still exists". That may be why
"here" came out as `folder:/Users/jhf/Projects/Hermon/lodgebook` rather than a
`project:` scope.

## The calls: 4 memory_remember, 2 memory_forget (via jcode `mcp` tool, server `hug`)

1. 12:18:39. Args: `content` (tmp/ has its own go.mod …), `type: "fact"`,
   `when: "running the full syncthing Go test suite …"`,
   `case: {scope: ["syncthing"], paths: ["tmp/"]}`. No `place`.
   Reply: `{"id":"fmsn14","place":"here","places":[{"name":"here","scope":"folder:/Users/jhf/Projects/Hermon/lodgebook",…},{"name":"global","scope":"jcode@global",…}]}`.
   Stored: `scope folder:/Users/jhf/Projects/Hermon/lodgebook`, category `fact`.
   **Agent error:** the reply named lodgebook and I did not read it.
2. 12:30:23. Same shape (`type: "rule"`, `case: {scope:["syncthing"], tools:["git"]}`), no `place`.
   Reply again said here = lodgebook. Stored: lodgebook, `type rule`, `category fact`. I noticed this time.
3. 12:31:02 / 12:31:03. `memory_forget` fmsn14 and fks94p: `"forgot fmsn14"`, `"forgot fks94p"` (now `active:false`).
4. 12:31:06. `memory_remember` with `scope: "folder:/Users/jhf/Projects/syncthing"`, `type: "rule"`, `when: …`, no `case`.
   Reply: **`Error: no visible memory`**. But the memory WAS stored: `aypq2r`,
   `scope folder:/Users/jhf/Projects/syncthing`, `active:true`. Believing the error,
   I stored it again:
5. 12:31:19. `place: "global"`, `place_reason: …`, `case: {scope:["syncthing"], paths:["~/Projects/syncthing"]}`.
   Stored `v1jd95`, `scope jcode@global`.

Records in `~/Library/Application Support/muninn/memories.json` (ids mem_1309dbd355d109c5,
mem_add914a147fad2d7, mem_dd040a3b5c44674e, mem_3fb661c75d01d01e) confirm all of the above.

## What the agent was told

The tool description says: "Use here (default, also when unsure) for this project".
`scope` is described as "literal binding scope, overrides place". Nothing says
"here" means the MCP server's cwd, and the reply's `places` list was the only
signal that it was lodgebook.

## Defects, in order of harm

1. **"here" is the server's cwd, not the caller's project.** Any shared-server
   harness misfiles. Possible fixes: take the caller's working dir per call (a
   required `cwd`/`project_root` argument, or MCP roots if jcode sends them), or
   refuse "here" when it cannot know the caller's project.
2. **A successful write returns `Error: no visible memory`** when `scope` names a
   scope outside the server's visible set. The agent then writes a duplicate.
   Either reject the write or report success. The write must not succeed silently
   behind an error.
3. **An explicit folder scope bypasses project identity:**
   `folder:/Users/jhf/Projects/syncthing` sits beside the repo's
   `project:rc-82f3e8642441ebef`, so it is a third home for the same project.
4. Minor: `type: "rule"` was stored with `category: "fact"`.

## Proposed fix (discussed with Jørgen, 2026-11-28)

Refuse "here" when the server cannot know where the caller is, and make
the caller either give its working directory or say global on purpose.

Limits, from this incident:

- **Detecting a "global" directory would not have caught this case.** The
  server's cwd was `Hermon/lodgebook`, a real project. Only one of the six
  `hug mcp` processes (pid 1402, cwd `/Users/jhf/Projects`) sits in a
  non-project directory.
- **The trigger has to be "this server may be shared", not "this directory
  looks global".** A server spawned by `jcode serve` hosts sessions from many
  directories, so its own cwd is never evidence of the caller's project.
  Simplest rule: the MCP path always needs a caller directory or an explicit
  global choice. Only the per-session hook can trust its cwd.
- **The caller's directory is easy for an agent to supply.** This session's
  shell cwd is `/Users/jhf/Projects/syncthing`, and hug resolves that to
  `project:rc-82f3e8642441ebef`. That also closes defect 3: a resolved
  directory goes through project identity, while a hand-written `folder:`
  scope does not.
- **Echo the resolved project in plain words**, e.g. `stored in syncthing
  (project:rc-82f3e8642441ebef)`. In this incident the scope was in the
  `places` list of the reply and the agent missed it.
- **Defect 2 (stored, yet the reply is an error) is separate** and needs its
  own fix.

## State after cleanup (2026-11-28 13:29, nothing left to clean)

- `fmsn14`, `fks94p` (lodgebook scope): retired.
- `aypq2r`: moved with `hug mem rescope aypq2r --to project:rc-82f3e8642441ebef`, active. This is the one kept.
- `v1jd95` (global duplicate): retired.
- `t0hymt` (syncthing, stale: the repro tests left tmp/): retired.

Retired records stay in `memories.json` with `active:false` if you want to replay the incident.
