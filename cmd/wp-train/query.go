package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ── progress ─────────────────────────────────────────────────────────────────

func cmdProgress() {
	db := openDB()
	defer db.Close()
	bank := loadTaskBank()

	// Build layer info from task bank (no hardcoded names)
	layerTopics := make(map[string]int)
	layerFirstName := make(map[string]string) // first topic name as fallback
	for key, entry := range bank {
		layer := strings.TrimPrefix(strings.Split(key, ".")[0], "L")
		layerTopics[layer]++
		if _, ok := layerFirstName[layer]; !ok {
			layerFirstName[layer] = entry.Name
		}
	}

	// Layer display names from curriculum (best effort)
	layerNames := map[string]string{
		"1": "初识 WordPress", "2": "内容管理", "3": "建站实战", "4": "主题与定制",
		"5": "插件生态", "6": "内部原理", "7": "运维与安全", "8": "排障",
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

	// Collect all layer numbers (from bank + mastery)
	allLayers := make(map[string]bool)
	for l := range layerTopics {
		allLayers[l] = true
	}
	var layerNums []string
	for l := range allLayers {
		layerNums = append(layerNums, l)
	}
	sort.Strings(layerNums)

	var layers []map[string]any
	for _, num := range layerNums {
		name := layerNames[num]
		if name == "" {
			name = layerFirstName[num]
		}
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

// ── shared query helpers ─────────────────────────────────────────────────────

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
