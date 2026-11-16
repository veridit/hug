package cli

import "strings"

// Command help is self-contained so agents never need to hand-format Markdown.
func commandHelp(command string) string {
	usage := map[string]string{
		"install":   "install jcode [--all-instances | --jcode-home PATH] [--dry-run]",
		"setup":     "setup",
		"uninstall": "uninstall jcode [--all-instances | --jcode-home PATH] [--dry-run]",
		"init":      "init", "new": `new "title" --goal "outcome" --accept "observable check" [flags]`,
		"ls":   "ls [--status todo|doing|done|dropped] [--type TYPE] [--priority LABEL] [--label LABEL] [--group FOLDER] [--epic EPIC] [--mine] [--jsonl]",
		"show": "show ID", "edit": "edit ID", "move": "move ID (--before ID | --after ID | --top)", "mv": "mv ID GROUP",
		"take": "take ID", "done": "done ID", "check": "check ID N", "promote": "promote ID N", "note": `note ID "text"`,
		"set":  `set ID [--title "title"] [--type TYPE] [--priority LABEL] [--label LABEL] [--status STATUS] [--assignee NAME] [--parent EPIC|none]`,
		"drop": "drop ID", "dep": "dep ID [--add REF ...] [--remove REF ...]", "next": "next [--type TYPE] [--priority LABEL] [--label LABEL] [--group FOLDER] [--epic EPIC] [--mine]",
		"rebalance": "rebalance [ID ...]", "types": "types", "lint": "lint [--code CODE]", "sync": "sync", "import": "import backlog DIRECTORY [--dry-run] | import github [OWNER/REPO] [--dry-run] | import files DIR [--cite PREFIX] [--dry-run]", "board": "board [--all] [--wide] [--type TYPE] [--priority LABEL] [--label LABEL] [--group FOLDER] [--epic EPIC] [--mine]", "tui": "tui", "mcp": "mcp", "upgrade": "upgrade", "help": "help [COMMAND]",
	}
	details := map[string]string{
		"setup":     "Prints what an agent needs to switch this repository to huginn, then a checklist of the steps still open, each with the exact command or edit. Reads only; changes nothing. Re-run after each step.\n",
		"install":   "Adds huginn -> hug task mcp to Jcode mcp.json. Keeps a timestamped backup and other servers. Default: JCODE_HOME or ~/.jcode. --all-instances discovers the static zsh registry. Restart Jcode afterward.\n",
		"uninstall": "Removes only the huginn MCP entry, keeping backups and other servers. Same instance selection as install.\n",
		"new":       "Required: --goal TEXT and at least one --accept TEXT. Repeat --accept for each check.\nDescribe with --goal (where we aim), --purpose (why), --principle (what we do and do not do)\nand --details (only facts a decision depends on); lint asks for them on new tasks.\nOptional: --type feature|bug|chore|spike|docs (default feature), --priority LABEL.\nRepeat --dep REF for dependencies and --label LABEL for labels.\nUse --group FOLDER, --top, or --after REF to choose placement.\nExample: hug task new \"Form\" --goal \"Submit works\" --accept \"Saved\" --dep k7m2qx --group auth\n",
		"ls":        "Columns: id, status, type, slug, title. All statuses are included unless filtered. Filters combine.\n--jsonl prints one JSON task per line; --json one object with a tasks array.\n--mine uses HUGINN_USER, otherwise git config user.name.\n",
		"show":      "Shows goal, notes, numbered acceptance and dependency ids with current titles/statuses/paths.\n",
		"edit":      "Runs $EDITOR, commits the edited task and reports lint findings.\nPrefer check, note and set for non-interactive changes.\n",
		"move":      "Changes filename rank only, repairing neighbouring ranks automatically if needed.\nUse mv to change folder.\n",
		"mv":        "Moves the task to a folder without changing its id, rank or contents.\nExample: hug task mv k7m2qx auth\n",
		"take":      "Pulls, checks dependencies and ownership, sets doing, commits and pushes as a lock.\nIdentity: HUGINN_USER, otherwise git config user.name.\nRepeating take on your own doing task succeeds without changing it.\n",
		"done":      "Blocks on open dependencies and unchecked acceptance (exit 3 with exact fixes).\nCheck every acceptance item first. There is no --force. Repeating done succeeds.\n",
		"promote":   "Turns acceptance item N into its own task, ranked right after ID in the same group and under\nthe same epic (an epic's item becomes its child). The item then reads 'done when task:NEW is done'\nand is ticked automatically when NEW is done.\nExample: hug task promote k7m2qx 2\n",
		"check":     "Ticks acceptance item N, numbered from 1 in show output. Repeating it succeeds.\nExample: hug task check k7m2qx 1\n",
		"note":      "Appends text to Notes, preserving existing content.\nExample: hug task note k7m2qx \"Blocked on p3rt9a; waiting for API\"\n",
		"set":       "Types: feature|bug|chore|spike|docs. Statuses: todo|doing|done|dropped.\nPriority is a free label. --label and --assignee replace the field with one value.\nStatus done still enforces dependencies and acceptance. Use take to claim work.\n--parent EPIC puts the task under an epic (epic > task > acceptance item; an epic has no parent);\n--parent none clears it. An epic is done only when its children are done or dropped.\n",
		"dep":       "Add or remove dependency references, stored as immutable ids. Repeat --add/--remove.\nRepeats are no-ops. A dependency cycle blocks with exit 3 and an exact fix.\nExample: hug task dep k7m2qx --add p3rt9a\n",
		"drop":      "Marks a task dropped even when blocked. Does not delete the file. Repeating it succeeds.\n",
		"next":      "Plain next: highest-ranked unblocked, unassigned todo that is not an epic (an epic finishes through its children). --epic EPIC limits to that epic. Exit 4 when none match.\n--mine: my unblocked doing first, then todo, in rank order. Filters combine.\nTypes: feature|bug|chore|spike|docs. Priority is a free label.\n",
		"rebalance": "Rekeys the rank prefix of task filenames (e.g. 000000000003.ID.slug.md becomes 00i.ID.slug.md)\nso later moves have room. Task order, ids and contents do not change; it is one commit,\nundone with git revert in .hug/tasks. Usually unnecessary: move rekeys neighbours itself.\nWith IDs, only the span between them is rekeyed.\n",
		"lint":      "Reports file:line: [code] problem, then fix: COMMAND, for malformed tasks, cycles, dependencies\non no task (dangling-dependency), Depends on lines that are not a dependency (not-a-dependency),\nparents, long titles and task files. next: is the fix for the first finding.\n--code CODE keeps one kind (e.g. long-title, no-acceptance, parent-too-deep) so they can be fixed one by one.\nJSON uses findings (path, line, problem, code, fix), an array (empty when clean).\n",
		"sync":      "Commits local task edits, pulls with rebase and pushes only the task branch (hug/tasks).\n",
		"upgrade":   "Moves tasks from the old layout (branch huginn at .huginn/) to branch hug/tasks at .hug/tasks/:\nrenames the branch, moves the folder, renames the remote branch. History is unchanged. Safe to run again.\n",
		"import": "Imports or re-syncs backlog-md tasks idempotently, preserving old ids as aliases.\nDIRECTORY is the backlog root (e.g. .backlog: reads tasks, drafts, completed and archive in one run;\nwhen an id is in several folders the newest updated_date wins) or a single folder of task files.\nRe-running merges backlog changes; edits made in huginn win. --dry-run reports without writing.\n" +
			"import github reads every issue (open and closed) of OWNER/REPO, or of this repository, through the gh CLI:\n" +
			"alias GH-<number>, labels and assignees kept, task-list checkboxes become acceptance items,\n" +
			"closed becomes done, closed as not planned becomes dropped. Re-run to sync; huginn edits win.\n",
		"tui":   "Interactive ranked list and board. For non-interactive output use board or ls --json.\n",
		"board": "Prints todo, doing and done side by side in rank order; blocked tasks are marked.\nDone shows the last 5 unless --all. Filters match ls. --json gives the columns.\n",
		"mcp":   "Runs the standard-I/O MCP server with ten flat tools and next-step hints.\n",
	}
	return "hug task " + usage[command] + "\n" + details[command] + "Refs accept ids, unique prefixes, slugs and aliases case-insensitively.\nAdd --json for structured output.\nnext: hug task next\n"
}
func isJSONL(args []string) bool {
	for _, a := range args {
		if a == "--jsonl" {
			return true
		}
	}
	return false
}
func isJSON(args []string) bool {
	for _, a := range args {
		if a == "--json" || strings.HasPrefix(a, "--json=") {
			return true
		}
	}
	return false
}
