package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ── init ─────────────────────────────────────────────────────────────────────

func cmdInit() {
	// Delete existing training site if found
	if sitePort != defaultPort || fileExists(filepath.Join(locwpBase, "wordpress")) {
		log("Deleting existing site on port " + sitePort + "...")
		shell("locwp delete " + sitePort)
	}

	log("Creating new site...")
	out, err := shell("locwp add --pass admin")
	if err != nil {
		fatal("locwp add failed: " + out)
	}
	// Parse port from locwp add output (e.g., "Site created at http://localhost:10001")
	port := ""
	for _, line := range strings.Split(out, "\n") {
		if idx := strings.Index(line, "localhost:"); idx >= 0 {
			rest := line[idx+len("localhost:"):]
			for _, c := range rest {
				if c >= '0' && c <= '9' {
					port += string(c)
				} else {
					break
				}
			}
			break
		}
	}
	if port == "" {
		// Fallback: find the newest site directory
		entries, _ := os.ReadDir(filepath.Join(homeDir, ".locwp", "sites"))
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].IsDir() && entries[i].Name() >= "10001" {
				port = entries[i].Name()
				break
			}
		}
	}
	if port == "" {
		fatal("could not determine site port from locwp add output")
	}

	sitePort = port
	setSitePaths(port)

	log("Installing Classic Editor...")
	wp("plugin install classic-editor --activate")
	wp("option update classic-editor-replace classic")
	wp("option update classic-editor-allow-users disallow")

	log("Configuring local environment...")
	muPluginDir := filepath.Join(wpRoot, "wp-content/mu-plugins")
	os.MkdirAll(muPluginDir, 0755)
	os.WriteFile(filepath.Join(muPluginDir, "skip-email-confirm.php"), []byte(
		"<?php\nadd_filter('send_email_change_email','__return_false');\nadd_filter('send_password_change_email','__return_false');\n"), 0644)
	wp("option update default_comment_status closed")

	log("Initializing git in wp-content...")
	shellMust(fmt.Sprintf("cd %s && git init -q && git add -A && git commit -m baseline -q", wpContent))

	log("Initializing database...")
	db := openDB()
	db.Exec("INSERT INTO sessions (started_at) VALUES (?)", nowISO())
	db.Close()

	// Clean old progress.json
	os.Remove(filepath.Join(trainingDir, "progress.json"))

	url := wp("option get siteurl")
	jprintln(map[string]any{
		"status":      "ok",
		"site":        sitePort,
		"url":         url,
		"admin_url":   url + "/wp-admin/",
		"credentials": map[string]string{"user": "admin", "pass": "admin"},
		"wp_root":     wpRoot,
		"db_path":     dbPath,
	})
}

// ── status ───────────────────────────────────────────────────────────────────

func cmdStatus() {
	db := openDB()
	defer db.Close()

	// Current task
	var taskJSON sql.NullString
	var issuedAt sql.NullString
	db.QueryRow("SELECT task_json, issued_at FROM current_task WHERE id = 1").Scan(&taskJSON, &issuedAt)

	var current any
	if taskJSON.Valid {
		json.Unmarshal([]byte(taskJSON.String), &current)
	}

	out := map[string]any{
		"current_task": current,
		"issued_at":    nullStr(issuedAt),
		"stats": map[string]any{
			"total_attempts":  dbGetInt(db, "SELECT COUNT(*) FROM attempts"),
			"total_passes":    dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE passed = 1"),
			"topics_mastered": dbGetInt(db, "SELECT COUNT(*) FROM topic_mastery WHERE mastered = 1"),
			"topics_started":  dbGetInt(db, "SELECT COUNT(DISTINCT topic) FROM attempts"),
		},
	}
	jprintln(out)
}

// ── next ─────────────────────────────────────────────────────────────────────

func cmdNext(args []string) {
	force := false
	topic := ""
	for i, a := range args {
		if a == "--force" {
			force = true
		} else if strings.HasPrefix(a, "--topic=") {
			topic = strings.TrimPrefix(a, "--topic=")
		} else if a == "--topic" && i+1 < len(args) {
			topic = args[i+1]
		}
	}

	db := openDB()
	defer db.Close()

	// Check existing task
	var existing sql.NullString
	db.QueryRow("SELECT task_json FROM current_task WHERE id = 1").Scan(&existing)
	if existing.Valid && !force {
		var task any
		json.Unmarshal([]byte(existing.String), &task)
		jprintln(map[string]any{"status": "existing", "task": task, "message": "Active task exists. Use --force to skip."})
		return
	}
	if force {
		db.Exec("DELETE FROM current_task WHERE id = 1")
	}

	bank := loadTaskBank()
	if bank == nil {
		jprintln(map[string]any{"status": "error", "message": "Task bank not found at " + taskBankPath})
		return
	}

	var selectedTopic string
	var selectedTask *Task

	if topic != "" {
		entry, ok := bank[topic]
		if !ok {
			jprintln(map[string]any{"status": "error", "message": "Topic not found: " + topic})
			return
		}
		selectedTopic = topic
		selectedTask = pickLeastAttempted(db, topic, entry.Tasks)
	} else {
		selectedTopic, selectedTask = selectNextTask(db, bank)
	}

	if selectedTask == nil {
		jprintln(map[string]any{"status": "complete", "message": "All topics mastered!"})
		return
	}

	record := map[string]any{
		"topic":        selectedTopic,
		"topic_name":   bank[selectedTopic].Name,
		"task_id":      selectedTask.ID,
		"difficulty":   selectedTask.Difficulty,
		"description":  selectedTask.Description,
		"hints":        selectedTask.Hints,
		"on_pass_note": selectedTask.OnPassNote,
		"verify":       selectedTask.Verify,
	}
	if selectedTask.Chain != "" {
		record["chain"] = selectedTask.Chain
		record["chain_order"] = selectedTask.ChainOrder
	}
	recordJSON, _ := json.Marshal(record)
	db.Exec("INSERT OR REPLACE INTO current_task (id, task_json, issued_at) VALUES (1, ?, ?)", string(recordJSON), nowISO())

	// Output without verify, with context
	mastered := listMastered(db)
	weakTopics := listWeak(db)
	currentCategory := topicCategory(selectedTopic)

	output := map[string]any{
		"status":       "ok",
		"topic":        selectedTopic,
		"topic_name":   bank[selectedTopic].Name,
		"task_id":      selectedTask.ID,
		"difficulty":   selectedTask.Difficulty,
		"description":  selectedTask.Description,
		"hints":        selectedTask.Hints,
		"on_pass_note": selectedTask.OnPassNote,
		"context": map[string]any{
			"current_category": currentCategory,
			"topics_mastered":  mastered,
			"weak_topics":      weakTopics,
			"task_attempts":   dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE task_id = ?", selectedTask.ID),
		},
	}
	if selectedTask.Chain != "" {
		output["chain"] = selectedTask.Chain
		output["chain_step"] = selectedTask.ChainOrder + 1
		// Count total steps in this chain
		total := 0
		for _, t := range bank[selectedTopic].Tasks {
			if t.Chain == selectedTask.Chain {
				total++
			}
		}
		output["chain_total"] = total
	}
	jprintln(output)

	// auto-sync progress after next
	if bank != nil {
		syncProgressMD(db, bank)
	}
	}

// builderPreference returns the builder topic the user has started, or "" if none/both.
func builderPreference(db *sql.DB) string {
	zeroyCount := dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE topic = 'zeroy'")
	elementorCount := dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE topic = 'elementor'")
	if zeroyCount > 0 && elementorCount == 0 {
		return "zeroy"
	}
	if elementorCount > 0 && zeroyCount == 0 {
		return "elementor"
	}
	return ""
}

// builderExclusions defines mutually exclusive builder topic pairs.
var builderExclusions = map[string]string{
	"zeroy":    "elementor",
	"elementor": "zeroy",
}

func selectNextTask(db *sql.DB, bank TaskBank) (string, *Task) {
	mastered := make(map[string]bool)
	rows, _ := db.Query("SELECT topic, mastered FROM topic_mastery")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var m int
			rows.Scan(&t, &m)
			if m == 1 {
				mastered[t] = true
			}
		}
	}

	pref := builderPreference(db)

	for _, key := range sortedKeys(bank) {
		if mastered[key] {
			continue
		}
		// Builder mutual exclusion: skip the other builder if user started one
		if pref != "" {
			if exclude, ok := builderExclusions[key]; ok && exclude != pref {
				continue
			}
		}
		entry := bank[key]
		task := pickLeastAttempted(db, key, entry.Tasks)
		if task != nil {
			return key, task
		}
	}
	return "", nil
}

func pickLeastAttempted(db *sql.DB, topic string, tasks []Task) *Task {
	counts := make(map[string]int)
	passed := make(map[string]bool)
	rows, _ := db.Query("SELECT task_id, COUNT(*) FROM attempts WHERE topic = ? GROUP BY task_id", topic)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			var c int
			rows.Scan(&id, &c)
			counts[id] = c
		}
	}
	// Check which tasks have been passed at least once
	rows2, _ := db.Query("SELECT DISTINCT task_id FROM attempts WHERE topic = ? AND passed = 1", topic)
	if rows2 != nil {
		defer rows2.Close()
		for rows2.Next() {
			var id string
			rows2.Scan(&id)
			passed[id] = true
		}
	}

	// Group by chain
	chains := make(map[string][]Task)
	var unchained []Task
	for _, t := range tasks {
		if t.Chain != "" {
			chains[t.Chain] = append(chains[t.Chain], t)
		} else {
			unchained = append(unchained, t)
		}
	}

	// For chained tasks: find the first not-yet-passed task in chain order
	allChainsPassed := true
	for _, chainTasks := range chains {
		sort.Slice(chainTasks, func(i, j int) bool {
			return chainTasks[i].ChainOrder < chainTasks[j].ChainOrder
		})
		for i := range chainTasks {
			if !passed[chainTasks[i].ID] {
				allChainsPassed = false
				t := chainTasks[i]
				return &t
			}
		}
	}

	// For unchained tasks: pick least attempted
	sorted := make([]Task, len(unchained))
	copy(sorted, unchained)
	sort.Slice(sorted, func(i, j int) bool {
		return counts[sorted[i].ID] < counts[sorted[j].ID]
	})
	if len(sorted) > 0 {
		return &sorted[0]
	}

	// All chains completed and no unchained tasks: pick least attempted
	// from all tasks for mastery review.
	if allChainsPassed && len(chains) > 0 {
		all := make([]Task, len(tasks))
		copy(all, tasks)
		sort.Slice(all, func(i, j int) bool {
			return counts[all[i].ID] < counts[all[j].ID]
		})
		return &all[0]
	}

	return nil
}

// ── verify ───────────────────────────────────────────────────────────────────

func cmdVerify() {
	db := openDB()
	defer db.Close()

	var taskJSON sql.NullString
	var issuedAt sql.NullString
	db.QueryRow("SELECT task_json, issued_at FROM current_task WHERE id = 1").Scan(&taskJSON, &issuedAt)
	if !taskJSON.Valid {
		jprintln(map[string]any{"status": "error", "message": "No active task. Use 'next' first."})
		return
	}

	var task map[string]any
	json.Unmarshal([]byte(taskJSON.String), &task)

	// Git auto-commit and capture diff
	var gitDiff string
	gitStat, _ := shell(fmt.Sprintf("cd %s && git status --porcelain 2>/dev/null", wpContent))
	if strings.TrimSpace(gitStat) != "" {
		gitDiff, _ = shell(fmt.Sprintf("cd %s && git diff 2>/dev/null", wpContent))
		shell(fmt.Sprintf("cd %s && git add -A && git commit -m 'checkpoint' -q 2>/dev/null || true", wpContent))
	}

	// Run checks
	checksRaw, _ := task["verify"].([]any)
	var checks []map[string]any
	for _, c := range checksRaw {
		if m, ok := c.(map[string]any); ok {
			checks = append(checks, m)
		}
	}
	results := runVerify(checks)

	allPassed := true
	for _, r := range results {
		if !r.Passed {
			allPassed = false
			break
		}
	}

	// Duration
	var durationSec *int
	if issuedAt.Valid {
		if t, err := time.Parse(time.RFC3339, issuedAt.String); err == nil {
			d := int(time.Since(t).Seconds())
			durationSec = &d
		}
	}

	topicStr := str(task["topic"])
	taskID := str(task["task_id"])
	difficulty := task["difficulty"]

	// Record attempt
	var errDetail *string
	if !allPassed {
		failedJSON, _ := json.Marshal(filterFailed(results))
		s := string(failedJSON)
		errDetail = &s
	}
	db.Exec("INSERT INTO attempts (topic, task_id, difficulty, passed, error_detail, duration_sec) VALUES (?, ?, ?, ?, ?, ?)",
		topicStr, taskID, difficulty, boolInt(allPassed), errDetail, durationSec)

	// Update mastery
	var cp int
	var wasMastered int
	var masteredAt sql.NullString
	err := db.QueryRow("SELECT consecutive_passes, mastered, mastered_at FROM topic_mastery WHERE topic = ?", topicStr).Scan(&cp, &wasMastered, &masteredAt)
	if err == sql.ErrNoRows {
		newCP := 0
		if allPassed {
			newCP = 1
		}
		db.Exec("INSERT INTO topic_mastery (topic, consecutive_passes, mastered, updated_at) VALUES (?, ?, 0, ?)", topicStr, newCP, nowISO())
		cp = newCP
	} else {
		if allPassed {
			cp++
			isMastered := 0
			ma := masteredAt
			if cp >= 2 {
				isMastered = 1
				if !masteredAt.Valid {
					ma = sql.NullString{String: nowISO(), Valid: true}
				}
			}
			db.Exec("UPDATE topic_mastery SET consecutive_passes = ?, mastered = ?, mastered_at = ?, updated_at = ? WHERE topic = ?",
				cp, isMastered, nullStr(ma), nowISO(), topicStr)
		} else {
			cp = 0
			db.Exec("UPDATE topic_mastery SET consecutive_passes = 0, updated_at = ? WHERE topic = ?", nowISO(), topicStr)
		}
	}

	if allPassed {
		db.Exec("DELETE FROM current_task WHERE id = 1")
	}

	// Get final mastery state
	var finalMastered int
	db.QueryRow("SELECT mastered FROM topic_mastery WHERE topic = ?", topicStr).Scan(&finalMastered)

	hints, _ := task["hints"].([]any)
	onPassNote, _ := task["on_pass_note"].(string)

	// Get attempt counts for this task and topic
	taskAttempts := dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE task_id = ?", taskID)
	topicAttempts := dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE topic = ?", topicStr)

	out := map[string]any{
		"status":         ternary(allPassed, "passed", "failed"),
		"task_id":        taskID,
		"topic":          topicStr,
		"topic_name":     str(task["topic_name"]),
		"results":        results,
		"task_attempts":  taskAttempts,
		"topic_attempts": topicAttempts,
		"mastery": map[string]any{
			"consecutive_passes": cp,
			"mastered":           finalMastered == 1,
		},
		"duration_sec": durationSec,
	}
	if gitDiff != "" {
		if len(gitDiff) > 2000 {
			gitDiff = gitDiff[:2000] + "\n... (truncated)"
		}
		out["git_diff"] = gitDiff
	}
	if allPassed {
		out["on_pass_note"] = onPassNote
		out["hints"] = []string{}
	} else {
		out["on_pass_note"] = ""
		out["hints"] = hints
	}
	jprintln(out)

		// auto-sync progress after verify
		bank := loadTaskBank()
		if bank != nil {
			syncProgressMD(db, bank)
		}
	}

	// ── small helpers ────────────────────────────────────────────────────────────

func nullStr(ns sql.NullString) any {
	if ns.Valid {
		return ns.String
	}
	return nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

func filterFailed(results []CheckResult) []CheckResult {
	var out []CheckResult
	for _, r := range results {
		if !r.Passed {
			out = append(out, r)
		}
	}
	return out
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ── progress MD sync ──────────────────────────────────────────────────────────

func syncProgressMD(db *sql.DB, bank TaskBank) {
	var sb strings.Builder
	sb.WriteString("# WP Guider 训练进度\n")

	now := time.Now().Format("2006-01-02 15:04")
	fmt.Fprintf(&sb, "> 最后更新: %s\n\n", now)

	// Current task
	var taskJSON sql.NullString
	db.QueryRow("SELECT task_json FROM current_task WHERE id = 1").Scan(&taskJSON)
	if taskJSON.Valid {
		var t map[string]any
		json.Unmarshal([]byte(taskJSON.String), &t)
		fmt.Fprintf(&sb, "## 当前任务\n")
		fmt.Fprintf(&sb, "- **%s** (%s) — %s\n\n",
			str(t["task_id"]), str(t["topic_name"]), str(t["description"]))
	} else {
		sb.WriteString("## 当前任务\n- 无活跃任务，运行 `wp-train next` 获取新题\n\n")
	}

	// Builder preference
	pref := builderPreference(db)
	if pref != "" {
		fmt.Fprintf(&sb, "- Builder 偏好: **%s**\n\n", pref)
	}

	// Stats
	totalTopics := len(bank)
	masteredTopics := 0
	rows, _ := db.Query("SELECT topic FROM topic_mastery WHERE mastered = 1")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			rows.Scan(&t)
			masteredTopics++
		}
	}
	totalAttempts := dbGetInt(db, "SELECT COUNT(*) FROM attempts")
	totalPasses := dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE passed = 1")
	passRate := 0.0
	if totalAttempts > 0 {
		passRate = float64(totalPasses) / float64(totalAttempts) * 100
	}

	sb.WriteString("## 总览\n")
	fmt.Fprintf(&sb, "- 已掌握: %d/%d 主题\n", masteredTopics, totalTopics)
	fmt.Fprintf(&sb, "- 总尝试: %d 次 | 通过: %d 次 | 通过率: %.0f%%\n\n", totalAttempts, totalPasses, passRate)

	// Category breakdown
	catOrder := []string{"基础设施", "站点设置", "内容管理", "外观定制", "插件与扩展", "运维与安全"}
	sb.WriteString("## 各分类进度\n")
	for _, cat := range catOrder {
		var topics []string
		for _, key := range sortedKeys(bank) {
			if topicCategory(key) == cat {
				topics = append(topics, key)
			}
		}
		if len(topics) == 0 {
			continue
		}
		catMastered := 0
		for _, t := range topics {
			if dbGetInt(db, "SELECT mastered FROM topic_mastery WHERE topic = ?", t) == 1 {
				catMastered++
			}
		}
		fmt.Fprintf(&sb, "### %s (%d/%d)\n", cat, catMastered, len(topics))
		for _, t := range topics {
			mastered := dbGetInt(db, "SELECT mastered FROM topic_mastery WHERE topic = ?", t) == 1
			entry := bank[t]
			if mastered {
				fmt.Fprintf(&sb, "- ✅ %s (%s)\n", entry.Name, t)
			} else {
				cp := dbGetInt(db, "SELECT consecutive_passes FROM topic_mastery WHERE topic = ?", t)
				attempts := dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE topic = ?", t)
				if attempts > 0 {
					fmt.Fprintf(&sb, "- 🔵 %s (%s) — 连续通过 %d/2 | 尝试 %d 次\n", entry.Name, t, cp, attempts)
				} else {
					fmt.Fprintf(&sb, "- ⬜ %s (%s)\n", entry.Name, t)
				}
			}
		}
		sb.WriteString("\n")
	}

	// Weak topics
	weak := listWeak(db)
	if len(weak) > 0 {
		sb.WriteString("## 薄弱主题\n")
		for _, w := range weak {
			topic, _ := w["topic"].(string)
			cp, _ := w["consecutive_passes"].(int)
			attempts, _ := w["attempts"].(int)
			fmt.Fprintf(&sb, "- ⚠️ %s — 连续通过 %d/2 | 尝试 %d 次\n", topic, cp, attempts)
		}
		sb.WriteString("\n")
	}

	// Recent attempts
	sb.WriteString("## 最近记录\n")
	recent := recentAttempts(db, 5)
	for _, r := range recent {
		passed, _ := r["passed"].(bool)
		topic, _ := r["topic"].(string)
		taskID, _ := r["task_id"].(string)
		mark := "✅"
		if !passed {
			mark = "❌"
		}
		fmt.Fprintf(&sb, "- %s %s/%s\n", mark, topic, taskID)
	}

	// Write file
	mdPath := filepath.Join(claudeDir, "PROGRESS.md")
	os.WriteFile(mdPath, []byte(sb.String()), 0644)
}
