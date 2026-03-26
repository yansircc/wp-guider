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
	// Check if site exists
	out, _ := shell(fmt.Sprintf(`locwp list 2>/dev/null | grep "^%s "`, siteName))
	if out != "" {
		log("Deleting existing site " + siteName + "...")
		shellMust("locwp delete " + siteName)
	}

	log("Creating site " + siteName + "...")
	shellMust("locwp add " + siteName + " --pass admin")

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
		"site":        siteName,
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
	currentLayer := "1"
	for _, key := range sortedKeys(bank) {
		layer := strings.TrimPrefix(strings.Split(key, ".")[0], "L")
		if !contains(mastered, key) {
			currentLayer = layer
			break
		}
	}

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
			"current_layer":   currentLayer,
			"topics_mastered": mastered,
			"weak_topics":     weakTopics,
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

	for _, key := range sortedKeys(bank) {
		if mastered[key] {
			continue
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
	for _, chainTasks := range chains {
		sort.Slice(chainTasks, func(i, j int) bool {
			return chainTasks[i].ChainOrder < chainTasks[j].ChainOrder
		})
		for i := range chainTasks {
			if !passed[chainTasks[i].ID] {
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
}

// ── progress ─────────────────────────────────────────────────────────────────

func cmdProgress() {
	db := openDB()
	defer db.Close()
	bank := loadTaskBank()

	layerNames := map[string]string{
		"1": "初识 WordPress", "2": "内容管理", "3": "文件系统与引导", "4": "数据层",
		"5": "主题引擎", "6": "插件与扩展", "7": "HTTP 与服务器层", "8": "排障",
	}

	layerTopics := make(map[string]int)
	for key := range bank {
		layer := strings.TrimPrefix(strings.Split(key, ".")[0], "L")
		layerTopics[layer]++
	}

	layerMastered := make(map[string]int)
	rows, _ := db.Query("SELECT topic, mastered FROM topic_mastery")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var m int
			rows.Scan(&t, &m)
			if m == 1 {
				layer := strings.TrimPrefix(strings.Split(t, ".")[0], "L")
				layerMastered[layer]++
			}
		}
	}

	totalAttempts := dbGetInt(db, "SELECT COUNT(*) FROM attempts")
	totalPasses := dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE passed = 1")
	totalT := 0
	totalM := 0
	for _, v := range layerTopics {
		totalT += v
	}
	for _, v := range layerMastered {
		totalM += v
	}

	var layers []map[string]any
	for _, num := range []string{"1", "2", "3", "4", "5", "6", "7", "8"} {
		layers = append(layers, map[string]any{
			"layer":    num,
			"name":     layerNames[num],
			"mastered": layerMastered[num],
			"total":    layerTopics[num],
		})
	}

	passRate := 0.0
	if totalAttempts > 0 {
		passRate = float64(totalPasses) / float64(totalAttempts)
	}

	jprintln(map[string]any{
		"layers": layers,
		"stats": map[string]any{
			"total_topics":   totalT,
			"total_mastered": totalM,
			"total_attempts": totalAttempts,
			"total_passes":   totalPasses,
			"pass_rate":      passRate,
		},
		"mastered_topics": listMastered(db),
		"weak_topics":     listWeak(db),
		"recent_attempts": recentAttempts(db, 5),
	})
}

// ── snapshot ─────────────────────────────────────────────────────────────────

func cmdSnapshot() {
	data := make(map[string]any)

	safeWPJSON := func(args string) []map[string]any {
		defer func() { recover() }()
		return wpJSON(args)
	}

	data["pages"] = safeWPJSON("post list --post_type=page --fields=ID,post_title,post_status,post_name")
	data["posts"] = safeWPJSON("post list --post_type=post --fields=ID,post_title,post_status,post_date")
	data["themes"] = safeWPJSON("theme list --fields=name,status,version")
	data["plugins"] = safeWPJSON("plugin list --fields=name,status,version")
	data["menus"] = safeWPJSON("menu list")
	data["categories"] = safeWPJSON("term list category --fields=term_id,name,slug,count")
	data["tags"] = safeWPJSON("term list post_tag --fields=term_id,name,slug,count")
	data["users"] = safeWPJSON("user list --fields=ID,user_login,roles")

	opts := make(map[string]any)
	for _, key := range []string{"blogname", "blogdescription", "siteurl", "home", "show_on_front", "page_on_front", "page_for_posts", "permalink_structure", "template", "stylesheet"} {
		v, err := shell(fmt.Sprintf("locwp wp %s -- option get %s", siteName, key))
		if err != nil {
			opts[key] = nil
		} else {
			opts[key] = v
		}
	}
	data["options"] = opts

	gitStat, _ := shell(fmt.Sprintf("cd %s && git diff --stat 2>/dev/null", wpContent))
	if strings.TrimSpace(gitStat) == "" {
		data["git_changes"] = "(clean)"
	} else {
		data["git_changes"] = gitStat
	}

	jprintln(data)
}

// ── explain ──────────────────────────────────────────────────────────────────

func cmdExplain(topic string) {
	curriculumPath := filepath.Join(claudeDir, "skills", "wp-train", "references", "curriculum.md")
	data, err := os.ReadFile(curriculumPath)
	if err != nil {
		jprintln(map[string]any{"status": "error", "message": "Curriculum not found"})
		return
	}

	content := string(data)
	parts := strings.Split(strings.TrimPrefix(topic, "L"), ".")
	if len(parts) >= 2 {
		section := fmt.Sprintf("### %s.%s", parts[0], parts[1])
		idx := strings.Index(content, section)
		if idx >= 0 {
			rest := content[idx:]
			// Find next section
			end := len(rest)
			for _, prefix := range []string{"\n### ", "\n## "} {
				if i := strings.Index(rest[1:], prefix); i >= 0 && i+1 < end {
					end = i + 1
				}
			}
			jprintln(map[string]any{"status": "ok", "topic": topic, "content": strings.TrimSpace(rest[:end])})
			return
		}
	}
	jprintln(map[string]any{"status": "error", "message": "Topic not found: " + topic})
}

// ── helpers ──────────────────────────────────────────────────────────────────

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

func listMastered(db *sql.DB) []string {
	var topics []string
	rows, _ := db.Query("SELECT topic FROM topic_mastery WHERE mastered = 1 ORDER BY topic")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			rows.Scan(&t)
			topics = append(topics, t)
		}
	}
	return topics
}

func listWeak(db *sql.DB) []map[string]any {
	var weak []map[string]any
	rows, _ := db.Query("SELECT topic, consecutive_passes FROM topic_mastery WHERE mastered = 0")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var cp int
			rows.Scan(&t, &cp)
			attempts := dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE topic = ?", t)
			if attempts >= 2 {
				weak = append(weak, map[string]any{
					"topic":              t,
					"consecutive_passes": cp,
					"attempts":           attempts,
				})
			}
		}
	}
	return weak
}

func recentAttempts(db *sql.DB, n int) []map[string]any {
	var attempts []map[string]any
	rows, _ := db.Query("SELECT topic, task_id, passed, duration_sec, created_at FROM attempts ORDER BY id DESC LIMIT ?", n)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var topic, taskID, createdAt string
			var passed int
			var durSec sql.NullInt64
			rows.Scan(&topic, &taskID, &passed, &durSec, &createdAt)
			a := map[string]any{
				"topic":      topic,
				"task_id":    taskID,
				"passed":     passed == 1,
				"created_at": createdAt,
			}
			if durSec.Valid {
				a["duration_sec"] = durSec.Int64
			}
			attempts = append(attempts, a)
		}
	}
	return attempts
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ── history ──────────────────────────────────────────────────────────────────

func cmdHistory(args []string) {
	n := 10
	for i, a := range args {
		if a == "-n" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &n)
		}
	}

	db := openDB()
	defer db.Close()

	jprintln(map[string]any{
		"recent_attempts": recentAttempts(db, n),
		"mastered_topics": listMastered(db),
		"weak_topics":     listWeak(db),
	})
}

