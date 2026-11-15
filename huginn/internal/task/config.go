package task

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// DepsPolicy reads the project's dependency policy, defaulting to block.
// Unrelated TOML keys are left to their owning components.
func DepsPolicy(root string) (string, error) {
	return readPolicy(root, "deps")
}

// AcceptancePolicy defaults to block, mirroring DepsPolicy.
func AcceptancePolicy(root string) (string, error) { return readPolicy(root, "acceptance") }

// ConfigName is the task settings file at the root of the task branch.
// The old name huginn.toml is still read; hug upgrade renames it.
const ConfigName = "hug.toml"

func configPath(root string) string {
	p := filepath.Join(root, ConfigName)
	if _, e := os.Stat(p); e != nil {
		if old := filepath.Join(root, "huginn.toml"); fileExists(old) {
			return old
		}
	}
	return p
}

func fileExists(p string) bool { _, e := os.Stat(p); return e == nil }

func readPolicy(root, field string) (string, error) {
	f, e := os.Open(configPath(root))
	if os.IsNotExist(e) {
		return "block", nil
	}
	if e != nil {
		return "", e
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	section := ""
	policy := "block"
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		if section != "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != field {
			continue
		}
		value = scalar(value)
		if !contains([]string{"block", "warn"}, value) {
			e := Invalid(field, value, []string{"block", "warn"})
			e.Fix = "set " + field + " = \"block\" in " + ConfigName
			return "", e
		}
		policy = value
	}
	return policy, scan.Err()
}

// ForbidCite lists path prefixes that task text must not cite, because those
// folders get deleted (hug.toml: forbid_cite = [".jcode/scratch/"]).
func ForbidCite(root string) ([]string, error) {
	b, e := os.ReadFile(configPath(root))
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	section := ""
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if section != "" || !ok || strings.TrimSpace(key) != "forbid_cite" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
		var out []string
		for _, v := range strings.Split(value, ",") {
			if v = scalar(v); v != "" {
				out = append(out, v)
			}
		}
		return out, nil
	}
	return nil, nil
}
