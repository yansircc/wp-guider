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
		jprintln(map[string]any{"status": "error", "message": "usage: wp-train checkpoint <save|restore|list> [name]"})
		return
	}

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
	case "list":
		doCheckpointList()
	default:
		jprintln(map[string]any{"status": "error", "message": "unknown action: " + action + " (save|restore|list)"})
	}
}

func checkpointDir() string {
	d := filepath.Join(trainingDir, "checkpoints")
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

func doCheckpointList() {
	dir := checkpointDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		jprintln(map[string]any{"status": "ok", "checkpoints": []string{}})
		return
	}

	var checkpoints []map[string]any
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, _ := e.Info()
		cp := map[string]any{
			"name": e.Name(),
		}
		if info != nil {
			cp["created_at"] = info.ModTime().Format("2006-01-02 15:04:05")
		}
		// Check what's in the checkpoint
		cpDir := filepath.Join(dir, e.Name())
		cp["has_db"] = fileExists(filepath.Join(cpDir, "db.sql"))
		cp["has_config"] = fileExists(filepath.Join(cpDir, "wp-config.php"))
		checkpoints = append(checkpoints, cp)
	}

	jprintln(map[string]any{"status": "ok", "checkpoints": checkpoints})
}
