package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ── inject ───────────────────────────────────────────────────────────────────

type faultDef struct {
	desc         string
	inject       func()
	fix          string
	verifyChecks []map[string]any
}

var faultTypes = map[string]faultDef{
	"syntax-error": {
		desc:   "在 functions.php 注入 PHP 语法错误 → 白屏",
		inject: injectSyntaxError,
		fix:    "找到 functions.php 中的语法错误并修复",
		verifyChecks: []map[string]any{
			{"type": "wp_eval", "php_code": "echo 'alive';", "expected_output": "alive"},
		},
	},
	"plugin-conflict": {
		desc:   "激活一个会产生致命错误的 mu-plugin → 500",
		inject: injectPluginConflict,
		fix:    "找到并移除 mu-plugins/ 下的问题插件",
		verifyChecks: []map[string]any{
			{"type": "wp_eval", "php_code": "echo 'alive';", "expected_output": "alive"},
		},
	},
	"wrong-siteurl": {
		desc:   "把 siteurl 改错 → 后台重定向死循环",
		inject: injectWrongSiteurl,
		fix:    "通过 wp-cli 或数据库修正 siteurl",
		verifyChecks: []map[string]any{
			{"type": "option_contains", "key": "siteurl", "substring": "localhost"},
		},
	},
	"memory-limit": {
		desc:   "设置极低内存限制 → Fatal error: Allowed memory size",
		inject: injectMemoryLimit,
		fix:    "在 wp-config.php 中调整 WP_MEMORY_LIMIT",
		verifyChecks: []map[string]any{
			{"type": "file_not_contains", "path": "wp-config.php", "pattern": "WP-GUIDER-FAULT"},
			{"type": "wp_eval", "php_code": "echo 'alive';", "expected_output": "alive"},
		},
	},
	"broken-db": {
		desc:   "改错数据库密码 → Error establishing a database connection",
		inject: injectBrokenDB,
		fix:    "检查 wp-config.php 中的 DB_PASSWORD 配置",
		verifyChecks: []map[string]any{
			{"type": "wp_eval", "php_code": "echo 'alive';", "expected_output": "alive"},
		},
	},
	"broken-permalink": {
		desc:   "把固定链接改为无效结构 → 所有页面 404",
		inject: injectBrokenPermalink,
		fix:    "检查并修复固定链接设置，刷新 rewrite rules",
		verifyChecks: []map[string]any{
			{"type": "option_contains", "key": "permalink_structure", "substring": "postname"},
		},
	},
	"debug-off": {
		desc:   "关闭 WP_DEBUG 并在 functions.php 中引入一个 warning → 排查无日志可看",
		inject: injectDebugOff,
		fix:    "开启 WP_DEBUG 和 WP_DEBUG_LOG，在 debug.log 中找到问题",
		verifyChecks: []map[string]any{
			{"type": "config_equals", "key": "WP_DEBUG", "expected": "1"},
			{"type": "config_equals", "key": "WP_DEBUG_LOG", "expected": "1"},
		},
	},
}

func cmdInject(args []string) {
	if len(args) < 1 {
		types := make([]map[string]any, 0)
		for k, v := range faultTypes {
			types = append(types, map[string]any{
				"type":        k,
				"description": v.desc,
				"fix_hint":    v.fix,
			})
		}
		jprintln(map[string]any{"status": "ok", "available_faults": types})
		return
	}

	faultType := args[0]
	ft, ok := faultTypes[faultType]
	if !ok {
		jprintln(map[string]any{"status": "error", "message": "Unknown fault type: " + faultType})
		return
	}

	log("Auto-saving checkpoint 'pre-fault'...")
	doCheckpointSave("pre-fault")

	log("Injecting fault: " + faultType + "...")
	ft.inject()

	// Create current_task so /wp-check can verify the fix
	db := openDB()
	defer db.Close()
	taskRecord := map[string]any{
		"topic":        "troubleshooting",
		"topic_name":   "故障排查",
		"task_id":      "fault-" + faultType,
		"difficulty":   3,
		"description":  ft.desc + " — 请诊断并修复",
		"hints":        []string{ft.fix},
		"on_pass_note": "排障核心：先看错误信息（error.log/debug.log），缩小范围（逐步排除），验证修复。",
		"verify":       ft.verifyChecks,
	}
	recordJSON, _ := json.Marshal(taskRecord)
	db.Exec("INSERT OR REPLACE INTO current_task (id, task_json, issued_at) VALUES (1, ?, ?)", string(recordJSON), nowISO())

	jprintln(map[string]any{
		"status":      "ok",
		"fault":       faultType,
		"description": ft.desc,
		"fix_hint":    ft.fix,
		"checkpoint":  "pre-fault",
		"message":     "Fault injected. Coach should diagnose and fix. Use 'checkpoint restore pre-fault' to undo.",
	})
}

// ── Fault implementations ────────────────────────────────────────────────────

func injectSyntaxError() {
	p := activeFunctionsPhp()
	content, _ := os.ReadFile(p)
	poison := "\n<?php\n// WP-GUIDER-FAULT: syntax-error\nfunction wp_guider_broken( {\n  return 'this will never work';\n}\n"
	os.WriteFile(p, append(content, []byte(poison)...), 0644)
}

func injectPluginConflict() {
	muDir := filepath.Join(wpContent, "mu-plugins")
	os.MkdirAll(muDir, 0755)
	poison := `<?php
/*
Plugin Name: WP Guider Fault - Plugin Conflict
*/
// WP-GUIDER-FAULT: plugin-conflict
wp_guider_undefined_function_call();
`
	os.WriteFile(filepath.Join(muDir, "wp-guider-fault.php"), []byte(poison), 0644)
}

func injectWrongSiteurl() {
	original := wp("option get siteurl")
	log("Original siteurl: " + original)
	shellMust(fmt.Sprintf("locwp wp %s -- option update siteurl https://wrong-domain.example.com", siteName))
}

func injectMemoryLimit() {
	configPath := filepath.Join(wpRoot, "wp-config.php")
	content, _ := os.ReadFile(configPath)
	marker := "/* That's all, stop editing!"
	injection := "define('WP_MEMORY_LIMIT', '4M'); // WP-GUIDER-FAULT: memory-limit\n"
	replaced := strings.Replace(string(content), marker, injection+marker, 1)
	if replaced == string(content) {
		replaced = strings.Replace(string(content), "<?php", "<?php\n"+injection, 1)
	}
	os.WriteFile(configPath, []byte(replaced), 0644)
}

func injectBrokenDB() {
	shellMust(fmt.Sprintf("locwp wp %s -- config set DB_PASSWORD 'wrong_password_wp_guider_fault' --type=constant", siteName))
}

func injectBrokenPermalink() {
	shellMust(fmt.Sprintf("locwp wp %s -- rewrite structure '/broken/%%postname%%/%%nonsense%%/' --hard", siteName))
	shellMust(fmt.Sprintf("locwp wp %s -- rewrite flush --hard", siteName))
}

func injectDebugOff() {
	shell(fmt.Sprintf("locwp wp %s -- config set WP_DEBUG false --raw 2>/dev/null || true", siteName))
	shell(fmt.Sprintf("locwp wp %s -- config set WP_DEBUG_LOG false --raw 2>/dev/null || true", siteName))
	p := activeFunctionsPhp()
	content, _ := os.ReadFile(p)
	poison := "\n// WP-GUIDER-FAULT: debug-off\nadd_action('init', function() {\n    $undefined_var = $nonexistent;\n    error_log('WP Guider: this warning is hidden because WP_DEBUG is off');\n});\n"
	os.WriteFile(p, append(content, []byte(poison)...), 0644)
}

func activeFunctionsPhp() string {
	theme := wp("option get stylesheet")
	p := filepath.Join(wpContent, "themes", theme, "functions.php")
	if !fileExists(p) {
		os.WriteFile(p, []byte("<?php\n"), 0644)
	}
	return p
}
