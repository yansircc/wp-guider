package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ── inject ───────────────────────────────────────────────────────────────────

var faultTypes = map[string]struct {
	desc   string
	inject func()
	fix    string
}{
	"syntax-error": {
		desc:   "在 functions.php 注入 PHP 语法错误 → 白屏",
		inject: injectSyntaxError,
		fix:    "找到 functions.php 中的语法错误并修复",
	},
	"plugin-conflict": {
		desc:   "激活一个会产生致命错误的 mu-plugin → 500",
		inject: injectPluginConflict,
		fix:    "找到并移除 mu-plugins/ 下的问题插件",
	},
	"wrong-siteurl": {
		desc:   "把 siteurl 改错 → 后台重定向死循环",
		inject: injectWrongSiteurl,
		fix:    "通过 wp-cli 或数据库修正 siteurl",
	},
	"memory-limit": {
		desc:   "设置极低内存限制 → Fatal error: Allowed memory size",
		inject: injectMemoryLimit,
		fix:    "在 wp-config.php 中调整 WP_MEMORY_LIMIT",
	},
	"broken-db": {
		desc:   "改错数据库密码 → Error establishing a database connection",
		inject: injectBrokenDB,
		fix:    "检查 wp-config.php 中的 DB_PASSWORD 配置",
	},
	"broken-htaccess": {
		desc:   "写入无效的 rewrite 规则 → 500 Internal Server Error",
		inject: injectBrokenHtaccess,
		fix:    "检查并修复 .htaccess 文件",
	},
	"debug-off": {
		desc:   "关闭 WP_DEBUG 并在 functions.php 中引入一个 warning → 排查无日志可看",
		inject: injectDebugOff,
		fix:    "开启 WP_DEBUG 和 WP_DEBUG_LOG，在 debug.log 中找到问题",
	},
}

func cmdInject(args []string) {
	if len(args) < 1 {
		// List available fault types
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

	// Auto-checkpoint before injection
	log("Auto-saving checkpoint 'pre-fault'...")
	doCheckpointSave("pre-fault")

	log("Injecting fault: " + faultType + "...")
	ft.inject()

	jprintln(map[string]any{
		"status":      "ok",
		"fault":       faultType,
		"description": ft.desc,
		"fix_hint":    ft.fix,
		"checkpoint":  "pre-fault",
		"message":     "Fault injected. Use 'checkpoint restore pre-fault' to undo.",
	})
}

// ── Fault implementations ────────────────────────────────────────────────────

func injectSyntaxError() {
	functionsPath := activeFunctionsPhp()
	content, _ := os.ReadFile(functionsPath)
	// Append a syntax error at the end
	poison := "\n<?php\n// WP-GUIDER-FAULT: syntax-error\nfunction wp_guider_broken( {\n  return 'this will never work';\n}\n"
	os.WriteFile(functionsPath, append(content, []byte(poison)...), 0644)
}

func injectPluginConflict() {
	muDir := filepath.Join(wpContent, "mu-plugins")
	os.MkdirAll(muDir, 0755)
	poison := `<?php
/*
Plugin Name: WP Guider Fault - Plugin Conflict
Description: Deliberately causes a fatal error for training
*/
// WP-GUIDER-FAULT: plugin-conflict
class WP_Guider_Conflict {
    public function __construct() {
        // Call undefined function to trigger fatal error
        wp_guider_undefined_function_call();
    }
}
new WP_Guider_Conflict();
`
	os.WriteFile(filepath.Join(muDir, "wp-guider-fault.php"), []byte(poison), 0644)
}

func injectWrongSiteurl() {
	// Save original for reference
	original := wp("option get siteurl")
	log("Original siteurl: " + original)
	shellMust(fmt.Sprintf("locwp wp %s -- option update siteurl https://wrong-domain.example.com", siteName))
}

func injectMemoryLimit() {
	configPath := filepath.Join(wpRoot, "wp-config.php")
	content, _ := os.ReadFile(configPath)
	// Insert extremely low memory limit before "That's all, stop editing!"
	marker := "/* That's all, stop editing!"
	injection := "define('WP_MEMORY_LIMIT', '4M'); // WP-GUIDER-FAULT: memory-limit\n"
	replaced := strings.Replace(string(content), marker, injection+marker, 1)
	if replaced == string(content) {
		// Fallback: prepend after <?php
		replaced = strings.Replace(string(content), "<?php", "<?php\n"+injection, 1)
	}
	os.WriteFile(configPath, []byte(replaced), 0644)
}

func injectBrokenDB() {
	shellMust(fmt.Sprintf("locwp wp %s -- config set DB_PASSWORD 'wrong_password_wp_guider_fault' --type=constant", siteName))
}

func injectBrokenHtaccess() {
	htaccessPath := filepath.Join(wpRoot, ".htaccess")
	content, _ := os.ReadFile(htaccessPath)
	poison := "\n# WP-GUIDER-FAULT: broken-htaccess\nRewriteRule ^(.*)$ /nonexistent-loop/$1 [L]\n"
	os.WriteFile(htaccessPath, append(content, []byte(poison)...), 0644)
}

func injectDebugOff() {
	// Ensure debug is off
	shell(fmt.Sprintf("locwp wp %s -- config set WP_DEBUG false --raw 2>/dev/null || true", siteName))
	shell(fmt.Sprintf("locwp wp %s -- config set WP_DEBUG_LOG false --raw 2>/dev/null || true", siteName))

	// Add a warning-producing function
	functionsPath := activeFunctionsPhp()
	content, _ := os.ReadFile(functionsPath)
	poison := "\n// WP-GUIDER-FAULT: debug-off\nadd_action('init', function() {\n    $undefined_var = $nonexistent;\n    error_log('WP Guider: this warning is hidden because WP_DEBUG is off');\n});\n"
	os.WriteFile(functionsPath, append(content, []byte(poison)...), 0644)
}

// activeFunctionsPhp returns the path to the active theme's functions.php
func activeFunctionsPhp() string {
	theme := wp("option get stylesheet")
	p := filepath.Join(wpContent, "themes", theme, "functions.php")
	if !fileExists(p) {
		// Create if not exists
		os.WriteFile(p, []byte("<?php\n"), 0644)
	}
	return p
}
