package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ── checkpoint ───────────────────────────────────────────────────────────────

func cmdCheckpoint(args []string) {
	if len(args) < 1 {
		jprintln(map[string]any{"status": "error", "message": "usage: wp-train checkpoint <save|restore|restore-last|list> [name]"})
		return
	}

	// Always operate on the current task's site
	switchToCurrentTaskSite()

	action := args[0]
	name := "default"
	if len(args) >= 2 {
		name = args[1]
	}

	switch action {
	case "save":
		doCheckpointSave(name)
		jprintln(map[string]any{"status": "ok", "action": "save", "name": name})
	case "restore":
		doCheckpointRestore(name)
		jprintln(map[string]any{"status": "ok", "action": "restore", "name": name})
	case "restore-last":
		doCheckpointRestoreLast()
	case "list":
		doCheckpointList()
	default:
		jprintln(map[string]any{"status": "error", "message": "unknown action: " + action + " (save|restore|restore-last|list)"})
	}
}

func checkpointDir() string {
	profile := profileForPort(sitePort)
	d := filepath.Join(trainingDir, "checkpoints", profile)
	os.MkdirAll(d, 0755)
	return d
}

func doCheckpointSave(name string) {
	dir := filepath.Join(checkpointDir(), name)
	os.MkdirAll(dir, 0755)

	// 1. DB: copy SQLite file (or fallback to wp db export for MySQL)
	log("Saving database to checkpoint '" + name + "'...")
	sqliteDB := filepath.Join(wpContent, "database", ".ht.sqlite")
	if fileExists(sqliteDB) {
		// SQLite: direct file copy
		dstDB := filepath.Join(dir, "db.sqlite")
		src, _ := os.ReadFile(sqliteDB)
		os.WriteFile(dstDB, src, 0644)
	} else {
		// MySQL fallback
		dumpPath := filepath.Join(dir, "db.sql")
		shellMust(fmt.Sprintf("locwp wp %s -- db export %s --quiet", sitePort, dumpPath))
	}

	// 2. Git tag in wp-content
	log("Tagging wp-content state...")
	tag := "checkpoint/" + name
	// Delete old tag if exists
	shell(fmt.Sprintf("cd %s && git tag -d %s 2>/dev/null || true", wpContent, tag))
	// Commit any uncommitted changes first
	gitStat, _ := shell(fmt.Sprintf("cd %s && git status --porcelain 2>/dev/null", wpContent))
	if strings.TrimSpace(gitStat) != "" {
		shell(fmt.Sprintf("cd %s && git add -A && git commit -m 'checkpoint: %s' -q 2>/dev/null || true", wpContent, name))
	}
	shellMust(fmt.Sprintf("cd %s && git tag %s", wpContent, tag))

	// 3. Save wp-config.php (contains DB creds, debug settings, etc.)
	configSrc := filepath.Join(wpRoot, "wp-config.php")
	configDst := filepath.Join(dir, "wp-config.php")
	data, err := os.ReadFile(configSrc)
	if err == nil {
		os.WriteFile(configDst, data, 0644)
	}

	log("Checkpoint '" + name + "' saved")
}

func doCheckpointRestore(name string) {
	dir := filepath.Join(checkpointDir(), name)
	if !fileExists(dir) {
		fatal("checkpoint not found: " + name)
	}

	// 1. Restore wp-config.php FIRST (faults may modify config)
	configSrc := filepath.Join(dir, "wp-config.php")
	configDst := filepath.Join(wpRoot, "wp-config.php")
	if fileExists(configSrc) {
		log("Restoring wp-config.php...")
		data, _ := os.ReadFile(configSrc)
		os.WriteFile(configDst, data, 0644)
	}

	// 2. Restore database
	sqliteDst := filepath.Join(wpContent, "database", ".ht.sqlite")
	sqliteSrc := filepath.Join(dir, "db.sqlite")
	dumpPath := filepath.Join(dir, "db.sql")
	if fileExists(sqliteSrc) {
		// SQLite: direct file copy
		log("Restoring SQLite database from checkpoint '" + name + "'...")
		data, _ := os.ReadFile(sqliteSrc)
		os.WriteFile(sqliteDst, data, 0644)
	} else if fileExists(dumpPath) {
		// MySQL fallback
		log("Restoring MySQL database from checkpoint '" + name + "'...")
		shellMust(fmt.Sprintf("locwp wp %s -- db import %s --quiet", sitePort, dumpPath))
	}

	// 3. Restore wp-content from git tag
	tag := "checkpoint/" + name
	tagExists, _ := shell(fmt.Sprintf("cd %s && git tag -l %s", wpContent, tag))
	if strings.TrimSpace(tagExists) != "" {
		log("Restoring wp-content from git tag...")
		shellMust(fmt.Sprintf("cd %s && git checkout %s -- . 2>/dev/null && git checkout main 2>/dev/null || git checkout master 2>/dev/null || true", wpContent, tag))
		// Clean untracked files that weren't in the checkpoint
		shell(fmt.Sprintf("cd %s && git clean -fd 2>/dev/null || true", wpContent))
	}

	log("Checkpoint '" + name + "' restored")
}

func doCheckpointRestoreLast() {
	dir := filepath.Join(checkpointDir(), "latest")
	if !fileExists(dir) {
		jprintln(map[string]any{"status": "error", "message": "no checkpoint found for profile: " + profileForPort(sitePort) + " — complete at least one task first"})
		return
	}
	doCheckpointRestore("latest")
	jprintln(map[string]any{"status": "ok", "action": "restore-last", "profile": profileForPort(sitePort)})
}

func doCheckpointList() {
	baseDir := filepath.Join(trainingDir, "checkpoints")
	profiles := []string{"main", "elementor", "zeroy"}

	allCheckpoints := map[string][]map[string]any{}
	for _, profile := range profiles {
		dir := filepath.Join(baseDir, profile)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		var cps []map[string]any
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			info, _ := e.Info()
			cp := map[string]any{"name": e.Name()}
			if info != nil {
				cp["created_at"] = info.ModTime().Format("2006-01-02 15:04:05")
			}
			cpDir := filepath.Join(dir, e.Name())
			cp["has_db"] = fileExists(filepath.Join(cpDir, "db.sqlite")) || fileExists(filepath.Join(cpDir, "db.sql"))
			cp["has_config"] = fileExists(filepath.Join(cpDir, "wp-config.php"))
			cps = append(cps, cp)
		}
		if len(cps) > 0 {
			allCheckpoints[profile] = cps
		}
	}

	jprintln(map[string]any{"status": "ok", "checkpoints": allCheckpoints})
}
