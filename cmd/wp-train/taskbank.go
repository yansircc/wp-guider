package main

import (
	"encoding/json"
	"os"
	"sort"
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
}

func loadTaskBank() TaskBank {
	data, err := os.ReadFile(taskBankPath)
	if err != nil {
		return nil
	}
	var bank TaskBank
	if err := json.Unmarshal(data, &bank); err != nil {
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
