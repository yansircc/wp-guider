package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Task Bank ────────────────────────────────────────────────────────────────

func TestLoadTaskBank(t *testing.T) {
	// Create a temp task bank
	dir := t.TempDir()
	bank := map[string]any{
		"site-settings": map[string]any{
			"name": "Test Topic",
			"tasks": []any{
				map[string]any{
					"id": "SS-1", "difficulty": 1,
					"description": "Do something",
					"verify":      []any{map[string]any{"type": "option_equals", "key": "blogname", "expected": "test"}},
					"hints":       []any{"hint1"},
					"on_pass_note": "good job",
				},
			},
		},
	}
	data, _ := json.MarshalIndent(bank, "", "  ")
	path := filepath.Join(dir, "task-bank.json")
	os.WriteFile(path, data, 0644)

	// Override global
	origPath := taskBankPath
	taskBankPath = path
	defer func() { taskBankPath = origPath }()

	loaded := loadTaskBank()
	if loaded == nil {
		t.Fatal("loadTaskBank returned nil")
	}
	entry, ok := loaded["site-settings"]
	if !ok {
		t.Fatal("site-settings not found")
	}
	if entry.Name != "Test Topic" {
		t.Errorf("expected 'Test Topic', got %q", entry.Name)
	}
	if len(entry.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(entry.Tasks))
	}
	if entry.Tasks[0].ID != "SS-1" {
		t.Errorf("expected task id SS-1, got %q", entry.Tasks[0].ID)
	}
}

func TestSortedKeys(t *testing.T) {
	bank := TaskBank{
		"pages":         {Name: "b"},
		"site-settings": {Name: "a"},
		"user-management": {Name: "c"},
		"media":         {Name: "d"},
	}
	keys := sortedKeys(bank)
	expected := []string{"media", "pages", "site-settings", "user-management"}
	for i, k := range keys {
		if k != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], k)
		}
	}
}

// ── Chain ordering ───────────────────────────────────────────────────────────

func TestChainOrdering(t *testing.T) {
	tasks := []Task{
		{ID: "a", Chain: "project", ChainOrder: 2},
		{ID: "b", Chain: "project", ChainOrder: 0},
		{ID: "c", Chain: "project", ChainOrder: 1},
		{ID: "d"},  // unchained
	}

	// Simulate: none passed yet → should pick chain_order=0 first
	dir := t.TempDir()
	origDB := dbPath
	dbPath = filepath.Join(dir, "test.db")
	defer func() { dbPath = origDB }()

	origTraining := trainingDir
	trainingDir = dir
	defer func() { trainingDir = origTraining }()

	db := openDB()
	defer db.Close()

	result := pickLeastAttempted(db, "L1.1", tasks)
	if result == nil {
		t.Fatal("pickLeastAttempted returned nil")
	}
	if result.ID != "b" {
		t.Errorf("expected chain_order=0 task 'b', got %q", result.ID)
	}

	// Mark 'b' as passed → should pick 'c' (chain_order=1)
	db.Exec("INSERT INTO attempts (topic, task_id, difficulty, passed) VALUES ('L1.1', 'b', 1, 1)")
	result = pickLeastAttempted(db, "L1.1", tasks)
	if result == nil {
		t.Fatal("pickLeastAttempted returned nil after passing b")
	}
	if result.ID != "c" {
		t.Errorf("expected chain_order=1 task 'c', got %q", result.ID)
	}

	// Mark 'c' as passed → should pick 'a' (chain_order=2)
	db.Exec("INSERT INTO attempts (topic, task_id, difficulty, passed) VALUES ('L1.1', 'c', 1, 1)")
	result = pickLeastAttempted(db, "L1.1", tasks)
	if result == nil {
		t.Fatal("pickLeastAttempted returned nil after passing c")
	}
	if result.ID != "a" {
		t.Errorf("expected chain_order=2 task 'a', got %q", result.ID)
	}

	// Mark 'a' as passed → all chain done, should pick unchained 'd'
	db.Exec("INSERT INTO attempts (topic, task_id, difficulty, passed) VALUES ('L1.1', 'a', 1, 1)")
	result = pickLeastAttempted(db, "L1.1", tasks)
	if result == nil {
		t.Fatal("pickLeastAttempted returned nil after all chain passed")
	}
	if result.ID != "d" {
		t.Errorf("expected unchained task 'd', got %q", result.ID)
	}
}

// ── SQLite ───────────────────────────────────────────────────────────────────

func TestInitDB(t *testing.T) {
	dir := t.TempDir()
	origDB := dbPath
	dbPath = filepath.Join(dir, "test.db")
	defer func() { dbPath = origDB }()

	origTraining := trainingDir
	trainingDir = dir
	defer func() { trainingDir = origTraining }()

	db := openDB()
	defer db.Close()

	// Tables should exist
	tables := []string{"attempts", "topic_mastery", "sessions", "current_task"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestMasteryTracking(t *testing.T) {
	dir := t.TempDir()
	origDB := dbPath
	dbPath = filepath.Join(dir, "test.db")
	defer func() { dbPath = origDB }()

	origTraining := trainingDir
	trainingDir = dir
	defer func() { trainingDir = origTraining }()

	db := openDB()
	defer db.Close()

	// Insert mastery record
	db.Exec("INSERT INTO topic_mastery (topic, consecutive_passes, mastered) VALUES ('L1.1', 0, 0)")

	// Simulate pass
	db.Exec("UPDATE topic_mastery SET consecutive_passes = 1 WHERE topic = 'L1.1'")
	var cp int
	db.QueryRow("SELECT consecutive_passes FROM topic_mastery WHERE topic = 'L1.1'").Scan(&cp)
	if cp != 1 {
		t.Errorf("expected consecutive_passes=1, got %d", cp)
	}

	// Second pass → mastered
	db.Exec("UPDATE topic_mastery SET consecutive_passes = 2, mastered = 1 WHERE topic = 'L1.1'")
	var mastered int
	db.QueryRow("SELECT mastered FROM topic_mastery WHERE topic = 'L1.1'").Scan(&mastered)
	if mastered != 1 {
		t.Errorf("expected mastered=1, got %d", mastered)
	}

	// Fail → reset
	db.Exec("UPDATE topic_mastery SET consecutive_passes = 0 WHERE topic = 'L1.1'")
	db.QueryRow("SELECT consecutive_passes FROM topic_mastery WHERE topic = 'L1.1'").Scan(&cp)
	if cp != 0 {
		t.Errorf("expected consecutive_passes reset to 0, got %d", cp)
	}
}

// ── Selection algorithm ──────────────────────────────────────────────────────

func TestSelectNextTask(t *testing.T) {
	dir := t.TempDir()
	origDB := dbPath
	dbPath = filepath.Join(dir, "test.db")
	defer func() { dbPath = origDB }()

	origTraining := trainingDir
	trainingDir = dir
	defer func() { trainingDir = origTraining }()

	db := openDB()
	defer db.Close()

	bank := TaskBank{
		"site-settings":   {Name: "Topic A", Tasks: []Task{{ID: "SS-1", Difficulty: 1}}},
		"user-management": {Name: "Topic B", Tasks: []Task{{ID: "UM-1", Difficulty: 1}}},
		"pages":           {Name: "Topic C", Tasks: []Task{{ID: "PG-1", Difficulty: 1}}},
	}

	// Should pick first sorted topic (pages)
	topic, task := selectNextTask(db, bank)
	if topic != "pages" || task.ID != "PG-1" {
		t.Errorf("expected pages/PG-1, got %s/%s", topic, task.ID)
	}

	// Mark pages mastered → should pick site-settings
	db.Exec("INSERT INTO topic_mastery (topic, consecutive_passes, mastered) VALUES ('pages', 2, 1)")
	topic, task = selectNextTask(db, bank)
	if topic != "site-settings" {
		t.Errorf("expected site-settings, got %s", topic)
	}

	// Mark site-settings mastered → should pick user-management
	db.Exec("INSERT INTO topic_mastery (topic, consecutive_passes, mastered) VALUES ('site-settings', 2, 1)")
	topic, task = selectNextTask(db, bank)
	if topic != "user-management" {
		t.Errorf("expected user-management, got %s", topic)
	}

	// Mark all mastered → should return empty
	db.Exec("INSERT INTO topic_mastery (topic, consecutive_passes, mastered) VALUES ('user-management', 2, 1)")
	topic, task = selectNextTask(db, bank)
	if topic != "" || task != nil {
		t.Errorf("expected empty, got %s/%v", topic, task)
	}
}

// ── Verification engine (pure logic, no WP) ─────────────────────────────────

func TestRunVerifyUnknownType(t *testing.T) {
	checks := []map[string]any{
		{"type": "nonexistent_check"},
	}
	results := runVerify(checks)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("unknown check type should not pass")
	}
	if results[0].Error == "" {
		t.Error("expected error message for unknown check type")
	}
}

func TestCheckFileExists(t *testing.T) {
	dir := t.TempDir()

	origRoot := wpRoot
	wpRoot = dir
	defer func() { wpRoot = origRoot }()

	// File doesn't exist
	r := checkFileExists("nonexistent.txt")
	if r.Passed {
		t.Error("should fail for nonexistent file")
	}

	// Create file
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)
	r = checkFileExists("test.txt")
	if !r.Passed {
		t.Error("should pass for existing file")
	}
}

func TestCheckFileContains(t *testing.T) {
	dir := t.TempDir()

	origRoot := wpRoot
	wpRoot = dir
	defer func() { wpRoot = origRoot }()

	os.WriteFile(filepath.Join(dir, "test.php"), []byte("<?php echo 'hello world'; ?>"), 0644)

	r := checkFileContains("test.php", "hello world")
	if !r.Passed {
		t.Error("should find 'hello world'")
	}

	r = checkFileContains("test.php", "goodbye")
	if r.Passed {
		t.Error("should not find 'goodbye'")
	}

	r = checkFileContains("missing.php", "anything")
	if r.Passed {
		t.Error("should fail for missing file")
	}
}

func TestCheckFileNotContains(t *testing.T) {
	dir := t.TempDir()

	origRoot := wpRoot
	wpRoot = dir
	defer func() { wpRoot = origRoot }()

	os.WriteFile(filepath.Join(dir, "test.php"), []byte("WP-GUIDER-FAULT"), 0644)

	r := checkFileNotContains("test.php", "WP-GUIDER-FAULT")
	if r.Passed {
		t.Error("should fail when pattern found")
	}

	r = checkFileNotContains("test.php", "SOMETHING-ELSE")
	if !r.Passed {
		t.Error("should pass when pattern not found")
	}

	r = checkFileNotContains("missing.php", "anything")
	if !r.Passed {
		t.Error("should pass for missing file")
	}
}

// ── JSON output helpers ──────────────────────────────────────────────────────

func TestTaskBankJSON(t *testing.T) {
	// Verify actual task bank files are valid
	tasksDirs := []string{
		"out/.claude/references/tasks",
		filepath.Join("..", "..", "out", ".claude", "references", "tasks"),
	}

	var dir string
	for _, d := range tasksDirs {
		if _, err := os.Stat(d); err == nil {
			dir = d
			break
		}
	}
	if dir == "" {
		t.Skip("tasks/ directory not found (expected in repo root)")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read tasks dir: %v", err)
	}

	totalTasks := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("cannot read %s: %v", e.Name(), err)
		}
		var bank TaskBank
		if err := json.Unmarshal(data, &bank); err != nil {
			t.Fatalf("invalid %s: %v", e.Name(), err)
		}
		for topicKey, entry := range bank {
			if entry.Name == "" {
				t.Errorf("%s: topic %s missing name", e.Name(), topicKey)
			}
			for _, task := range entry.Tasks {
				totalTasks++
				if task.ID == "" {
					t.Errorf("%s: task missing id", e.Name())
				}
				if task.Description == "" {
					t.Errorf("task %s: missing description", task.ID)
				}
				if len(task.Verify) == 0 {
					t.Errorf("task %s: missing verify checks", task.ID)
				}
				for _, v := range task.Verify {
					if _, ok := v["type"]; !ok {
						t.Errorf("task %s: verify check missing 'type'", task.ID)
					}
				}
				if task.Chain != "" && task.ChainOrder < 0 {
					t.Errorf("task %s: negative chain_order", task.ID)
				}
			}
		}
	}
	t.Logf("Validated %d tasks across %d files", totalTasks, len(entries))
}
