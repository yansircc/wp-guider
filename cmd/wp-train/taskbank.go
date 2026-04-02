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
	SiteProfile string           `json:"site_profile,omitempty"`
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

// topicOrder defines the canonical curriculum progression.
// Topics not in this list fall to the end in alphabetical order.
var topicOrder = []string{
	// 基础设施
	"domain", "hosting", "wp-install",
	// 站点设置
	"site-settings", "user-management",
	// 内容管理
	"pages", "media", "menus-nav", "posts-taxonomy",
	// 外观定制 — theme 先，再 elementor，最后 zeroy
	"theme", "elementor", "zeroy",
	// 插件与扩展
	"plugins-basic", "acf", "seo", "google-analytics",
	// 运维与安全
	"wp-config", "security", "backup-maintenance", "troubleshooting",
}

// sortedKeys returns topic keys in curriculum order.
func sortedKeys(bank TaskBank) []string {
	ordered := make([]string, 0, len(bank))
	seen := make(map[string]bool)
	for _, k := range topicOrder {
		if _, ok := bank[k]; ok {
			ordered = append(ordered, k)
			seen[k] = true
		}
	}
	// Append any remaining keys alphabetically
	var rest []string
	for k := range bank {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(ordered, rest...)
}
