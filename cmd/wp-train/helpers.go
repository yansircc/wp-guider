package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const defaultPort = "10001"

var (
	homeDir      string
	mainPort     string // main training site port — DB lives here
	sitePort     string // currently active wp-cli target port (may differ for builder sites)
	locwpBase    string // set dynamically per active site port
	wpRoot       string
	wpContent    string
	trainingDir  string // always based on mainPort
	dbPath       string // always based on mainPort
	claudeDir    string // .claude/ directory (product root)
	taskBankPath string
)

func init() {
	homeDir, _ = os.UserHomeDir()

	// Find .claude/ directory first (needed for DB path to resolve port)
	claudeDir = findClaudeDir()
	taskBankPath = filepath.Join(claudeDir, "references", "task-bank.json") // legacy fallback only

	// Resolve main site port and set training paths (DB always lives on main site)
	mainPort = resolveMainPort()
	setTrainingPaths(mainPort)
	switchSite(mainPort)
}

// setTrainingPaths sets the training DB/data paths (always anchored to main site).
func setTrainingPaths(port string) {
	trainingDir = filepath.Join(homeDir, ".locwp", "sites", port, "training")
	dbPath = filepath.Join(trainingDir, "wp-guider.db")
}

// switchSite updates the active wp-cli target without touching trainingDir/dbPath.
func switchSite(port string) {
	sitePort = port
	locwpBase = filepath.Join(homeDir, ".locwp", "sites", port)
	wpRoot = filepath.Join(locwpBase, "wordpress")
	wpContent = filepath.Join(wpRoot, "wp-content")
}

// resolveMainPort finds the main training site port.
// Priority: 1) WP_TRAIN_PORT env  2) existing DB  3) default 10001
func resolveMainPort() string {
	if p := os.Getenv("WP_TRAIN_PORT"); p != "" {
		return p
	}
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

// ── site map (multi-site port registry) ──────────────────────────────────────

// SiteMap maps builder profile names to their locwp port numbers.
type SiteMap struct {
	Main      string  `json:"main"`
	Elementor *string `json:"elementor"`
	Zeroy     *string `json:"zeroy"`
}

func siteMapPath() string {
	return filepath.Join(trainingDir, "sites.json")
}

func loadSiteMap() SiteMap {
	data, err := os.ReadFile(siteMapPath())
	if err != nil {
		return SiteMap{Main: mainPort}
	}
	var sm SiteMap
	if json.Unmarshal(data, &sm) != nil {
		return SiteMap{Main: mainPort}
	}
	return sm
}

func saveSiteMap(sm SiteMap) {
	os.MkdirAll(trainingDir, 0755)
	data, _ := json.MarshalIndent(sm, "", "  ")
	os.WriteFile(siteMapPath(), data, 0644)
}

// switchToCurrentTaskSite reads the active task from DB and switches sitePort
// to the task's site profile. No-op if no active task or profile is "main".
func switchToCurrentTaskSite() {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)")
	if err != nil {
		return
	}
	defer db.Close()

	var taskJSON string
	if err := db.QueryRow("SELECT task_json FROM current_task WHERE id = 1").Scan(&taskJSON); err != nil {
		return
	}
	var task map[string]any
	if json.Unmarshal([]byte(taskJSON), &task) != nil {
		return
	}
	profile, _ := task["site_profile"].(string)
	if profile == "" || profile == "main" {
		return
	}
	if port := portForProfile(profile); port != "" {
		switchSite(port)
	}
}

// profileForPort returns the site profile name for a given port.
func profileForPort(port string) string {
	sm := loadSiteMap()
	if port == sm.Main {
		return "main"
	}
	if sm.Elementor != nil && *sm.Elementor == port {
		return "elementor"
	}
	if sm.Zeroy != nil && *sm.Zeroy == port {
		return "zeroy"
	}
	return "main"
}

// portForProfile returns the locwp port for a site profile, or "" if not yet created.
func portForProfile(profile string) string {
	sm := loadSiteMap()
	switch profile {
	case "", "main":
		return sm.Main
	case "elementor":
		if sm.Elementor != nil {
			return *sm.Elementor
		}
	case "zeroy":
		if sm.Zeroy != nil {
			return *sm.Zeroy
		}
	}
	return ""
}

// ensureBuilderSite returns the port for a builder profile, creating (forking from
// main) if it doesn't exist yet. Returns the port and the site URL.
func ensureBuilderSite(profile string) (string, error) {
	if port := portForProfile(profile); port != "" {
		return port, nil
	}

	log("Creating " + profile + " site (forking from main site " + mainPort + ")...")

	// Create new locwp site
	out, err := shell("locwp add --pass admin")
	if err != nil {
		return "", fmt.Errorf("locwp add failed: %s", out)
	}
	newPort := parsePort(out)
	if newPort == "" {
		return "", fmt.Errorf("could not determine port from locwp output: %s", out)
	}

	// Copy WordPress SQLite database from main site (inherits all settings/content)
	srcDB := filepath.Join(homeDir, ".locwp", "sites", mainPort, "wordpress", "wp-content", "database", ".ht.sqlite")
	dstDB := filepath.Join(homeDir, ".locwp", "sites", newPort, "wordpress", "wp-content", "database", ".ht.sqlite")
	if fileExists(srcDB) {
		log("Copying WordPress database...")
		src, err := os.ReadFile(srcDB)
		if err != nil {
			return "", fmt.Errorf("failed to read main DB: %v", err)
		}
		if err := os.WriteFile(dstDB, src, 0644); err != nil {
			return "", fmt.Errorf("failed to write builder DB: %v", err)
		}
	}

	// Fix all URL references (wp search-replace handles serialized data correctly)
	log("Updating URLs for new port...")
	shellMust(fmt.Sprintf(
		"locwp wp %s -- search-replace 'localhost:%s' 'localhost:%s' --all-tables --quiet",
		newPort, mainPort, newPort,
	))

	// Initialize git tracking on new wp-content
	newWpContent := filepath.Join(homeDir, ".locwp", "sites", newPort, "wordpress", "wp-content")
	log("Initializing git in new wp-content...")
	shellMust(fmt.Sprintf("cd %s && git init -q && git add -A && git commit -m baseline -q", newWpContent))

	// Persist the new port to sites.json
	sm := loadSiteMap()
	switch profile {
	case "elementor":
		sm.Elementor = &newPort
	case "zeroy":
		sm.Zeroy = &newPort
	}
	saveSiteMap(sm)

	log("Builder site ready on port " + newPort)
	return newPort, nil
}

// parsePort extracts the port number from a "localhost:XXXXX" substring.
func parsePort(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if idx := strings.Index(line, "localhost:"); idx >= 0 {
			rest := line[idx+len("localhost:"):]
			port := ""
			for _, c := range rest {
				if c >= '0' && c <= '9' {
					port += string(c)
				} else {
					break
				}
			}
			if port != "" {
				return port
			}
		}
	}
	return ""
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
