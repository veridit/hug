#!/usr/bin/env python3
"""Real-binary tmux acceptance and golden screenshots for muninn's terminal UI.

Usage: scripts/tui-pty.py bin/muninn [--update] [--golden-dir DIR]
Runs only on synthetic data in a throwaway HOME. Every tmux session is killed.
The historical filename is retained, but tmux now supplies the real PTY and
screen emulator, avoiding false positives from concatenated ANSI output.
"""
import argparse
from contextlib import contextmanager
import difflib
import json
import os
from pathlib import Path
import re
import shlex
import shutil
import signal
import sqlite3
import subprocess
import tempfile
import time
import uuid

REPO = Path(__file__).resolve().parent.parent
TMUX_SOCKET = None


def command(*args, **kwargs):
    # Never share the owner's or another worker's server/configuration.
    if args[0] == "tmux" and TMUX_SOCKET:
        args = (args[0], "-L", TMUX_SOCKET, "-f", "/dev/null", *args[1:])
    timeout = kwargs.pop("timeout", 30)
    return subprocess.run(args, check=True, text=True, stdout=subprocess.PIPE,
                          stderr=subprocess.PIPE, timeout=timeout, **kwargs).stdout


@contextmanager
def private_server():
    """Keep a known foreground PID, even when a server command blocks in the OS."""
    global TMUX_SOCKET
    TMUX_SOCKET = "muninn-tui-" + uuid.uuid4().hex[:12]
    socket = Path(os.environ.get("TMUX_TMPDIR", "/tmp")) / f"tmux-{os.getuid()}" / TMUX_SOCKET
    server = subprocess.Popen(["tmux", "-D", "-L", TMUX_SOCKET, "-f", "/dev/null"],
                              stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        deadline = time.monotonic() + 30
        while not socket.exists():
            if server.poll() is not None or time.monotonic() > deadline:
                raise RuntimeError("private tmux server did not create its socket")
            time.sleep(.05)
        yield
    finally:
        try:
            command("tmux", "kill-server", timeout=3)
        except (subprocess.SubprocessError, OSError):
            pass
        if server.poll() is None:
            server.terminate()
            try:
                server.wait(timeout=3)
            except subprocess.TimeoutExpired:
                server.kill()
        server.wait(timeout=3)
        TMUX_SOCKET = None


def fixture(root, binary):
    home = root / "home"
    muninn = root / "muninn"
    project = home / "project"
    for path in (home / ".jcode", muninn / "data" / "tidy", muninn / "config", project / ".muninn"):
        path.mkdir(parents=True, exist_ok=True)
    (project / ".muninn" / "project").write_text("terminal-fixture\n")
    env = dict(os.environ, HOME=str(home), MUNINN_HOME=str(muninn),
               JCODE_HOME=str(home / ".jcode"), MUNINN_NO_EMBED="1",
               TERM="xterm-256color", COLORTERM="truecolor", COLORFGBG="15;0",
               GOMAXPROCS="2", GOFLAGS="-p=2", TZ="UTC", LANG="en_US.UTF-8")
    env.pop("NO_COLOR", None)
    config = {"embedder": "none", "jcode_discover": False, "notes_roots": [], "jcode_homes": []}
    (muninn / "config" / "config.json").write_text(json.dumps(config))
    rows = [
        ("fixture_banana", "Bake banana cake at 175 degrees. Use a skewer to check the centre.", "fact", "global"),
        ("fixture_deploy", "Deploys go through staging first.\n\n**Always** review the checklist. Use `release verify`.\n\n- [ ] Review changes\n- [x] Run tests\n\n<!-- SECTION:TEXT:END -->", "decision", "project:terminal-fixture"),
        ("fixture_keys", "Prefer Vim navigation and a visible overview before searching.", "preference", "global"),
        ("fixture_dupe_a", "Use ripgrep for repository searches, not recursive grep.", "fact", "global"),
        ("fixture_dupe_b", "Use ripgrep for repository searches, not recursive grep.", "fact", "global"),
        ("fixture_personal", "Tea is best after lunch.", "preference", "personal"),
    ]
    memories = []
    for i, (ref, body, category, scope) in enumerate(rows):
        date = f"2026-11-09T{10+i:02}:30:00Z"
        memories.append(dict(id=ref, content=body, category=category, scope=scope,
                             active=True, strength=1, created=date, updated=date))
    (muninn / "data" / "memories.json").write_text(json.dumps({"version": 1, "memories": memories}))
    command("nice", "-n", "19", binary, "index", env=env, cwd=project)
    db = sqlite3.connect(muninn / "data" / "muninn.db")
    unit, content_hash = db.execute("SELECT id,content_hash FROM units WHERE key='fixture_banana'").fetchone()
    db.execute("INSERT INTO injections VALUES(?,?,?,?,?)", ("crab", unit, content_hash, "jcode", 1794227400000000000))
    ids = dict(db.execute("SELECT key,sid FROM units"))
    db.commit()
    db.close()
    command("nice", "-n", "19", binary, "tidy", "dupes", "--out", str(muninn / "data" / "tidy" / "dupes-fixture.json"), env=env, cwd=project)
    return env, project, ids


def normalized(screen, root, project):
    screen = screen.replace(str(project), "<fixture>")
    screen = screen.replace(str(root / "home"), "<home>")
    screen = screen.replace(str(root / "muninn"), "<muninn>")
    screen = re.sub(r"results in [0-9.]+(?:ns|µs|ms|s)", "results in <time>", screen)
    return "\n".join(line.rstrip() for line in screen.splitlines()) + "\n"


def exercise(binary, root, width, height, no_color):
    env, project, ids = fixture(root, binary)
    if no_color:
        env["NO_COLOR"] = "1"
    name = "muninn-tui-" + uuid.uuid4().hex[:12]
    exports = [f"{k}={v}" for k, v in env.items() if k in {
        "HOME", "MUNINN_HOME", "JCODE_HOME", "MUNINN_NO_EMBED", "TERM", "COLORTERM",
        "COLORFGBG", "GOMAXPROCS", "GOFLAGS", "TZ", "LANG", "NO_COLOR"}]
    # env -u prevents an inherited NO_COLOR from contaminating the colour lane.
    shell = shlex.join(["env", "-u", "NO_COLOR", *exports, "nice", "-n", "19", binary, "tui"])
    shots = {}
    try:
        # macOS DirectoryServices may briefly stall tmux's getpwuid during
        # session creation. The foreground server remains owned and cancellable.
        command("tmux", "new-session", "-d", "-x", str(width), "-y", str(height), "-s", name, "-c", str(project), shell, timeout=120)
        command("tmux", "set-option", "-t", name, "status", "off")
        command("tmux", "set-option", "-t", name, "remain-on-exit", "on")

        def raw():
            return command("tmux", "capture-pane", "-p", "-t", name)

        def wait_for(needle):
            deadline = time.monotonic() + 20
            while time.monotonic() < deadline:
                current = raw()
                if needle in current:
                    return current
                time.sleep(.08)
            raise AssertionError(f"did not render {needle!r}:\n{raw()}")

        def send(text=None, key=None):
            if text is not None:
                command("tmux", "send-keys", "-t", name, "-l", text)
            if key is not None:
                command("tmux", "send-keys", "-t", name, key)
            time.sleep(.12)

        def shot(label, needle):
            current = wait_for(needle)
            # Capture only a settled frame, not a transient search result.
            time.sleep(.12)
            current = raw()
            assert len(current.splitlines()) == height, (label, "height", len(current.splitlines()))
            shots[label] = normalized(current, root, project)

        shot("overview", "6 visible memories")
        styled = command("tmux", "capture-pane", "-p", "-e", "-t", name)
        # Any colour SGR counts (truecolor/256, basic, bright). Bold/dim alone does not.
        sgr = re.findall(r"\x1b\[([0-9;:]*)m", styled)
        colours = [s for s in sgr if any(p in ("38", "48") or (p.isdigit() and (30 <= int(p) <= 37 or 40 <= int(p) <= 47 or 90 <= int(p) <= 97 or 100 <= int(p) <= 107)) for p in re.split(r"[;:]", s))]
        assert bool(colours) != no_color, f"NO_COLOR or terminal colour detection regressed (no_color={no_color}, TERM={env.get('TERM')}, sgr sample={sorted(set(sgr))[:12]})"
        send(text="/banana")
        wait_for("results in")
        send(key="Enter")
        shot("search", "filter kept")
        send(key="Enter")
        shot("detail", "Origin/session:")
        send(key="Escape")
        send(key="Escape")
        send(key="Escape")
        send(text=":category pref")
        shot("palette", "category preference")
        send(key="Escape")
        send(text="?")
        shot("help", "Help · Vim keys")
        send(key="Escape")
        send(text=":dupes")
        send(key="Enter")
        shot("needs-a-look", "KEEP ")
        send(key="Escape")
        send(text="fc")
        shot("filter-picker", "Quick filter · category")
        send(key="Enter")
        shot("filter-chips", "[category:fact]")
        send(text="F1")
        shot("quick-place", "[place:here]")
        send(text="F#" + ids["fixture_banana"][:3])
        shot("id-jump", "Jump to memory")
        send(key="Enter")
        wait_for("Origin/session:")
        send(text="q")
        # remain-on-exit lets us assert a clean process exit rather than losing
        # the PTY before tmux can report the exit code.
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            dead = command("tmux", "display-message", "-p", "-t", name, "#{pane_dead}:#{pane_dead_status}").strip()
            if dead == "1:0":
                break
            time.sleep(.05)
        else:
            raise AssertionError("q did not exit cleanly: " + dead)
        return shots
    finally:
        try:
            command("tmux", "kill-session", "-t", name, timeout=3)
        except (subprocess.SubprocessError, OSError):
            pass


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary")
    parser.add_argument("--golden-dir", type=Path, default=REPO / "internal" / "tui" / "testdata" / "tmux")
    parser.add_argument("--update", action="store_true")
    args = parser.parse_args()
    if not shutil.which("tmux"):
        parser.error("tmux is required for real-terminal golden tests")
    binary = str(Path(args.binary).resolve())
    scratch = Path(os.environ.get("JCODE_SCRATCH_DIR", str(REPO.parent / "tui" / "muninn")))
    scratch.mkdir(parents=True, exist_ok=True)
    failures = 0
    with tempfile.TemporaryDirectory(prefix="terminal-fixture-", dir=scratch) as tmp, private_server():
        for width, height in ((80, 24), (140, 40)):
            for no_color in (False, True):
                label = f"{width}x{height}-" + ("no-color" if no_color else "color")
                shots = exercise(binary, Path(tmp) / label, width, height, no_color)
                for name, actual in shots.items():
                    path = args.golden_dir / f"{label}-{name}.txt"
                    if args.update:
                        path.parent.mkdir(parents=True, exist_ok=True)
                        path.write_text(actual)
                    elif not path.exists() or path.read_text() != actual:
                        failures += 1
                        expected = path.read_text() if path.exists() else ""
                        print("".join(difflib.unified_diff(expected.splitlines(True), actual.splitlines(True), fromfile=str(path), tofile="actual")))
                print(f"{'PASS' if not failures else 'FAIL'} tui tmux {label}: {len(shots)} golden screens", flush=True)
    return bool(failures)


def interrupted(*_):
    raise KeyboardInterrupt


if __name__ == "__main__":
    # A coordinator cancelling this harness still reaches the tmux cleanup.
    signal.signal(signal.SIGTERM, interrupted)
    raise SystemExit(main())
