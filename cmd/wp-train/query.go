package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// ── progress ─────────────────────────────────────────────────────────────────

func cmdProgress() {
	db := openDB()
	defer db.Close()
	bank := loadTaskBank()

	// Build category info from task bank
	catTopics := make(map[string]int)    // category → topic count
	catMastered := make(map[string]int)  // category → mastered count
	for key := range bank {
		cat := topicCategory(key)
		catTopics[cat]++
	}

	rows, _ := db.Query("SELECT topic, mastered FROM topic_mastery")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t string
			var m int
			rows.Scan(&t, &m)
			if m == 1 {
				cat := topicCategory(t)
				catMastered[cat]++
			}
		}
	}

	totalAttempts := dbGetInt(db, "SELECT COUNT(*) FROM attempts")
	totalPasses := dbGetInt(db, "SELECT COUNT(*) FROM attempts WHERE passed = 1")
	totalT := 0
	totalM := 0
	for _, v := range catTopics {
		totalT += v
	}
	for _, v := range catMastered {
		totalM += v
	}

	// Ordered category list
	catOrder := []string{"基础设施", "站点设置", "内容管理", "外观定制", "插件与扩展", "运维与安全"}
	var categories []map[string]any
	for _, cat := range catOrder {
		if catTopics[cat] == 0 && catMastered[cat] == 0 {
			continue
		}
		categories = append(categories, map[string]any{
			"category": cat,
			"mastered": catMastered[cat],
			"total":    catTopics[cat],
		})
	}

	passRate := 0.0
	if totalAttempts > 0 {
		passRate = float64(totalPasses) / float64(totalAttempts)
	}

	jprintln(map[string]any{
		"categories": categories,
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
		v, err := shell(fmt.Sprintf("locwp wp %s -- option get %s", sitePort, key))
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
	bank := loadTaskBank()
	entry, ok := bank[topic]
	if !ok {
		jprintln(map[string]any{"status": "error", "message": "Topic not found: " + topic})
		return
	}
	cat := topicCategory(topic)
	taskIDs := make([]string, len(entry.Tasks))
	for i, t := range entry.Tasks {
		taskIDs[i] = t.ID
	}
	jprintln(map[string]any{
		"status":   "ok",
		"topic":    topic,
		"name":     entry.Name,
		"category": cat,
		"tasks":    len(entry.Tasks),
		"task_ids": taskIDs,
	})
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
