package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const defaultPort = "10001"

var (
	homeDir      string
	locwpBase    string // set dynamically per site port
	wpRoot       string
	wpContent    string
	trainingDir  string
	dbPath       string
	claudeDir    string // .claude/ directory (product root)
	taskBankPath string
	sitePort     string // locwp port, resolved at runtime
)

func init() {
	homeDir, _ = os.UserHomeDir()

	// Find .claude/ directory first (needed for DB path to resolve port)
	claudeDir = findClaudeDir()
	taskBankPath = filepath.Join(claudeDir, "references", "task-bank.json") // legacy fallback only

	// Resolve site port: try saved port from DB, else default
	sitePort = resolveSitePort()
	setSitePaths(sitePort)
}

// setSitePaths sets all path variables based on port.
func setSitePaths(port string) {
	locwpBase = filepath.Join(homeDir, ".locwp", "sites", port)
	wpRoot = filepath.Join(locwpBase, "wordpress")
	wpContent = filepath.Join(wpRoot, "wp-content")
	trainingDir = filepath.Join(locwpBase, "training")
	dbPath = filepath.Join(trainingDir, "wp-guider.db")
}

// resolveSitePort finds the training site port.
// Priority: 1) WP_TRAIN_PORT env  2) saved in any existing DB  3) default 10001
func resolveSitePort() string {
	if p := os.Getenv("WP_TRAIN_PORT"); p != "" {
		return p
	}
	// Scan ~/.locwp/sites/*/training/wp-guider.db to find existing training site
	sitesDir := filepath.Join(homeDir, ".locwp", "sites")
	if entries, err := os.ReadDir(sitesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			db := filepath.Join(sitesDir, e.Name(), "training", "wp-guider.db")
			if fileExists(db) {
				return e.Name()
			}
		}
	}
	return defaultPort
}

func findClaudeDir() string {
	// Try WP_GUIDER_DIR env first (points to .claude/)
	if d := os.Getenv("WP_GUIDER_DIR"); d != "" {
		return d
	}

	// Binary is at .claude/scripts/wp-train → parent of scripts/ is .claude/
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if filepath.Base(dir) == "scripts" {
			parent := filepath.Dir(dir)
			if isTaskBankDir(parent) {
				return parent
			}
		}
		// Walk up looking for references/tasks/ directory
		for d := dir; d != "/"; d = filepath.Dir(d) {
			if isTaskBankDir(d) {
				return d
			}
		}
	}

	// Fallback: cwd/.claude or cwd/out/.claude (dev mode)
	cwd, _ := os.Getwd()
	if d := filepath.Join(cwd, ".claude"); isTaskBankDir(d) {
		return d
	}
	if d := filepath.Join(cwd, "out", ".claude"); isTaskBankDir(d) {
		return d
	}
	return cwd
}

// isTaskBankDir checks whether dir looks like a valid .claude/ product root.
// Primary anchor: references/tasks/ directory with ≥1 JSON file.
// Legacy fallback: references/task-bank.json (single-file format).
func isTaskBankDir(dir string) bool {
	tasksDir := filepath.Join(dir, "references", "tasks")
	if entries, err := os.ReadDir(tasksDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				return true
			}
		}
	}
	return fileExists(filepath.Join(dir, "references", "task-bank.json"))
}

// shell runs a command and returns stdout.
func shell(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

// shellMust runs a command and panics on failure.
func shellMust(command string) string {
	out, err := shell(command)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fatal(fmt.Sprintf("command failed: %s\n%s", command, string(exitErr.Stderr)))
		}
		fatal(fmt.Sprintf("command failed: %s: %v", command, err))
	}
	return out
}

// wp runs a wp-cli command via locwp.
func wp(args string) string {
	return shellMust(fmt.Sprintf("locwp wp %s -- %s", sitePort, args))
}

// wpJSON runs a wp-cli command and parses JSON output.
func wpJSON(args string) []map[string]any {
	out := wp(args + " --format=json")
	if out == "" {
		return nil
	}
	var result []map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil
	}
	return result
}

// jprintln prints a value as JSON to stdout.
func jprintln(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// topicCategory returns the category name for a topic key.
func topicCategory(topic string) string {
	categories := map[string]string{
		"domain": "基础设施", "hosting": "基础设施", "wp-install": "基础设施",
		"site-settings": "站点设置", "user-management": "站点设置",
		"pages": "内容管理", "posts-taxonomy": "内容管理", "media": "内容管理", "menus-nav": "内容管理",
		"theme": "外观定制", "elementor": "外观定制", "zeroy": "外观定制",
		"plugins-basic": "插件与扩展", "acf": "插件与扩展", "seo": "插件与扩展", "google-analytics": "插件与扩展",
		"wp-config": "运维与安全", "security": "运维与安全", "backup-maintenance": "运维与安全", "troubleshooting": "运维与安全",
	}
	if cat, ok := categories[topic]; ok {
		return cat
	}
	return "其他"
}
