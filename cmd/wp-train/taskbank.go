package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TaskBank is the top-level task bank structure: map of topic_key -> TopicEntry.
type TaskBank map[string]TopicEntry

type TopicEntry struct {
	Name  string `json:"name"`
	Tasks []Task `json:"tasks"`
}

type Task struct {
	ID          string           `json:"id"`
	Difficulty  int              `json:"difficulty"`
	Description string           `json:"description"`
	Verify      []map[string]any `json:"verify"`
	Hints       []string         `json:"hints"`
	OnPassNote  string           `json:"on_pass_note"`
	Chain       string           `json:"chain,omitempty"`
	ChainOrder  int              `json:"chain_order,omitempty"`
}

func loadTaskBank() TaskBank {
	bank := make(TaskBank)

	// Try tasks/ directory first (split by layer)
	tasksDir := filepath.Join(claudeDir, "references", "tasks")
	if entries, err := os.ReadDir(tasksDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(tasksDir, e.Name()))
			if err != nil {
				continue
			}
			var partial TaskBank
			if json.Unmarshal(data, &partial) == nil {
				for k, v := range partial {
					bank[k] = v
				}
			}
		}
		if len(bank) > 0 {
			return bank
		}
	}

	// Fallback: single task-bank.json
	data, err := os.ReadFile(taskBankPath)
	if err != nil {
		return nil
	}
	if json.Unmarshal(data, &bank) != nil {
		return nil
	}
	return bank
}

// sortedKeys returns sorted topic keys.
func sortedKeys(bank TaskBank) []string {
	keys := make([]string, 0, len(bank))
	for k := range bank {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
