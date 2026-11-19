package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/veridit/hug/internal/hugconf"
)

func equalJSON(a, b json.RawMessage) bool {
	var x, y any
	return json.Unmarshal(a, &x) == nil && json.Unmarshal(b, &y) == nil && fmt.Sprint(x) == fmt.Sprint(y)
}

// editServers puts hug's MCP server in a jcode mcp.json (ynh84t): one entry
// named hug running `hug mcp --jcode-home HOME`, the same entry hug mem
// install writes, in place of the huginn or muninn entries hug wrote
// before. Uninstall takes off every entry hug wrote. Other servers stay;
// a "hug" entry hug did not write is an error naming the fix.
func editServers(root map[string]json.RawMessage, bin, home string, install bool) (bool, error) {
	var want json.RawMessage
	if install {
		if !filepath.IsAbs(bin) {
			return false, fmt.Errorf("hug command must be an absolute executable path")
		}
		want = hugconf.MCPEntry(hugconf.MCPCommand(bin), home)
	}
	changed, err := hugconf.EditServers(root, want, install)
	if errors.Is(err, hugconf.ErrForeign) {
		return false, fmt.Errorf("%w: rename it or remove it, then run hug task install jcode again", err)
	}
	return changed, err
}
