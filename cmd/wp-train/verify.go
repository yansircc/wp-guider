package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// CheckResult is a single verification result.
type CheckResult struct {
	Check    string `json:"check"`
	Passed   bool   `json:"passed"`
	Actual   string `json:"actual,omitempty"`
	Expected string `json:"expected,omitempty"`
	Error    string `json:"error,omitempty"`
}

// runVerify executes all checks and returns results.
func runVerify(checks []map[string]any) []CheckResult {
	var results []CheckResult
	for _, c := range checks {
		typ, _ := c["type"].(string)
		r := runSingleCheck(typ, c)
		r.Check = typ
		results = append(results, r)
	}
	return results
}

func getStr(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func getFloat(m map[string]any, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

func runSingleCheck(typ string, params map[string]any) CheckResult {
	switch typ {
	case "option_equals":
		return checkOptionEquals(getStr(params, "key"), getStr(params, "expected"))
	case "option_contains":
		return checkOptionContains(getStr(params, "key"), getStr(params, "substring"))
	case "post_exists":
		status := getStr(params, "status")
		if status == "" {
			status = "publish"
		}
		return checkPostExists(getStr(params, "post_type"), getStr(params, "title"), status)
	case "post_count":
		return checkPostCount(getStr(params, "post_type"), int(getFloat(params, "min")))
	case "term_exists":
		return checkTermExists(getStr(params, "taxonomy"), getStr(params, "name"))
	case "theme_active":
		return checkThemeActive(getStr(params, "theme"))
	case "plugin_active":
		return checkPluginActive(getStr(params, "plugin"))
	case "plugin_installed":
		return checkPluginInstalled(getStr(params, "plugin"))
	case "menu_exists":
		return checkMenuExists(getStr(params, "name"))
	case "menu_has_items":
		return checkMenuHasItems(getStr(params, "menu"), int(getFloat(params, "min_count")))
	case "file_exists":
		return checkFileExists(getStr(params, "path"))
	case "file_contains":
		return checkFileContains(getStr(params, "path"), getStr(params, "pattern"))
	case "file_not_contains":
		return checkFileNotContains(getStr(params, "path"), getStr(params, "pattern"))
	case "config_equals":
		return checkConfigEquals(getStr(params, "key"), getStr(params, "expected"))
	case "user_meta":
		return checkUserMeta(getStr(params, "user"), getStr(params, "key"), getStr(params, "expected"))
	case "git_diff_includes":
		return checkGitDiffIncludes(getStr(params, "path_pattern"))
	case "git_diff_not_empty":
		return checkGitDiffNotEmpty()
	case "wp_eval":
		return checkWpEval(getStr(params, "php_code"), getStr(params, "expected_output"))
	case "db_query":
		return checkDBQuery(getStr(params, "sql"), getStr(params, "expected"))
	default:
		return CheckResult{Passed: false, Error: "unknown check type: " + typ}
	}
}

func checkOptionEquals(key, expected string) CheckResult {
	actual, err := shell(fmt.Sprintf("locwp wp %s -- option get %s", sitePort, key))
	if err != nil {
		return CheckResult{Passed: false, Error: err.Error()}
	}
	return CheckResult{Passed: actual == expected, Actual: actual, Expected: expected}
}

func checkOptionContains(key, substring string) CheckResult {
	actual, err := shell(fmt.Sprintf("locwp wp %s -- option get %s", sitePort, key))
	if err != nil {
		return CheckResult{Passed: false, Error: err.Error()}
	}
	return CheckResult{Passed: strings.Contains(actual, substring), Actual: actual, Expected: "contains '" + substring + "'"}
}

func checkPostExists(postType, title, status string) CheckResult {
	posts := wpJSON(fmt.Sprintf("post list --post_type=%s", postType))
	for _, p := range posts {
		if str(p["post_title"]) == title {
			if status != "" && str(p["post_status"]) != status {
				return CheckResult{Passed: false, Actual: "status=" + str(p["post_status"]), Expected: "status=" + status}
			}
			return CheckResult{Passed: true, Actual: title, Expected: title}
		}
	}
	var titles []string
	for _, p := range posts {
		titles = append(titles, str(p["post_title"]))
	}
	return CheckResult{Passed: false, Actual: fmt.Sprintf("no post titled '%s' (found: %v)", title, titles), Expected: title}
}

func checkPostCount(postType string, min int) CheckResult {
	posts := wpJSON(fmt.Sprintf("post list --post_type=%s", postType))
	count := 0
	for _, p := range posts {
		if str(p["post_status"]) == "publish" {
			count++
		}
	}
	return CheckResult{Passed: count >= min, Actual: fmt.Sprint(count), Expected: fmt.Sprintf(">=%d", min)}
}

func checkTermExists(taxonomy, name string) CheckResult {
	terms := wpJSON(fmt.Sprintf("term list %s", taxonomy))
	var names []string
	for _, t := range terms {
		n := str(t["name"])
		names = append(names, n)
		if n == name {
			return CheckResult{Passed: true, Actual: name, Expected: name}
		}
	}
	return CheckResult{Passed: false, Actual: fmt.Sprint(names), Expected: name}
}

func checkThemeActive(theme string) CheckResult {
	themes := wpJSON("theme list")
	for _, t := range themes {
		if str(t["status"]) == "active" {
			actual := str(t["name"])
			return CheckResult{Passed: actual == theme, Actual: actual, Expected: theme}
		}
	}
	return CheckResult{Passed: false, Actual: "(none)", Expected: theme}
}

func checkPluginActive(plugin string) CheckResult {
	plugins := wpJSON("plugin list")
	for _, p := range plugins {
		if str(p["name"]) == plugin && str(p["status"]) == "active" {
			return CheckResult{Passed: true, Actual: "active", Expected: "active"}
		}
	}
	return CheckResult{Passed: false, Actual: "not active", Expected: "active"}
}

func checkPluginInstalled(plugin string) CheckResult {
	plugins := wpJSON("plugin list")
	for _, p := range plugins {
		if str(p["name"]) == plugin {
			return CheckResult{Passed: true, Actual: "installed", Expected: "installed"}
		}
	}
	return CheckResult{Passed: false, Actual: "not installed", Expected: "installed"}
}

func checkMenuExists(name string) CheckResult {
	menus := wpJSON("menu list")
	var names []string
	for _, m := range menus {
		n := str(m["name"])
		names = append(names, n)
		if n == name {
			return CheckResult{Passed: true, Actual: name, Expected: name}
		}
	}
	return CheckResult{Passed: false, Actual: fmt.Sprint(names), Expected: name}
}

func checkMenuHasItems(menu string, minCount int) CheckResult {
	menus := wpJSON("menu list")
	for _, m := range menus {
		if str(m["name"]) == menu {
			// Get term_id
			var mid string
			if v, ok := m["term_id"]; ok {
				mid = fmt.Sprint(v)
			}
			if mid == "" {
				mid = fmt.Sprint(m["id"])
			}
			items := wpJSON(fmt.Sprintf("menu item list %s", mid))
			count := len(items)
			return CheckResult{Passed: count >= minCount, Actual: fmt.Sprint(count), Expected: fmt.Sprintf(">=%d", minCount)}
		}
	}
	return CheckResult{Passed: false, Actual: "menu not found", Expected: fmt.Sprintf("menu '%s' with >=%d items", menu, minCount)}
}

func checkFileExists(path string) CheckResult {
	full := fmt.Sprintf("%s/%s", wpRoot, path)
	exists := fileExists(full)
	return CheckResult{Passed: exists, Actual: fmt.Sprint(exists), Expected: "exists"}
}

func checkFileContains(path, pattern string) CheckResult {
	full := fmt.Sprintf("%s/%s", wpRoot, path)
	data, err := os.ReadFile(full)
	if err != nil {
		return CheckResult{Passed: false, Actual: "file not found", Expected: "contains /" + pattern + "/"}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return CheckResult{Passed: false, Error: "invalid regex: " + err.Error()}
	}
	found := re.Match(data)
	actual := "not found"
	if found {
		actual = "found"
	}
	return CheckResult{Passed: found, Actual: actual, Expected: "contains /" + pattern + "/"}
}

func checkFileNotContains(path, pattern string) CheckResult {
	full := fmt.Sprintf("%s/%s", wpRoot, path)
	data, err := os.ReadFile(full)
	if err != nil {
		return CheckResult{Passed: true, Actual: "file not found", Expected: "not contains /" + pattern + "/"}
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return CheckResult{Passed: false, Error: "invalid regex: " + err.Error()}
	}
	found := re.Match(data)
	actual := "not found"
	if found {
		actual = "found"
	}
	return CheckResult{Passed: !found, Actual: actual, Expected: "not contains /" + pattern + "/"}
}

func checkConfigEquals(key, expected string) CheckResult {
	actual, err := shell(fmt.Sprintf("locwp wp %s -- config get %s", sitePort, key))
	if err != nil {
		return CheckResult{Passed: false, Error: err.Error()}
	}
	return CheckResult{Passed: actual == expected, Actual: actual, Expected: expected}
}

func checkUserMeta(user, key, expected string) CheckResult {
	out, err := shell(fmt.Sprintf("locwp wp %s -- user meta get %s %s --format=json", sitePort, user, key))
	if err != nil {
		return CheckResult{Passed: false, Error: err.Error()}
	}
	var vals []any
	json.Unmarshal([]byte(out), &vals)
	actual := "(empty)"
	if len(vals) > 0 {
		actual = fmt.Sprint(vals[0])
	}
	return CheckResult{Passed: actual == expected, Actual: actual, Expected: expected}
}

func checkGitDiffIncludes(pathPattern string) CheckResult {
	out, _ := shell(fmt.Sprintf("cd %s && git diff --name-only HEAD 2>/dev/null || git diff --name-only", wpContent))
	if out == "" {
		return CheckResult{Passed: false, Actual: "no matching changes", Expected: "changes matching /" + pathPattern + "/"}
	}
	re, err := regexp.Compile(pathPattern)
	if err != nil {
		return CheckResult{Passed: false, Error: "invalid regex: " + err.Error()}
	}
	var matched []string
	for _, f := range strings.Split(out, "\n") {
		if re.MatchString(f) {
			matched = append(matched, f)
		}
	}
	if len(matched) == 0 {
		return CheckResult{Passed: false, Actual: "no matching changes", Expected: "changes matching /" + pathPattern + "/"}
	}
	return CheckResult{Passed: true, Actual: fmt.Sprint(matched), Expected: "changes matching /" + pathPattern + "/"}
}

func checkGitDiffNotEmpty() CheckResult {
	out, _ := shell(fmt.Sprintf("cd %s && git diff --stat 2>/dev/null", wpContent))
	if strings.TrimSpace(out) == "" {
		return CheckResult{Passed: false, Actual: "no changes", Expected: "some changes"}
	}
	actual := out
	if len(actual) > 200 {
		actual = actual[:200]
	}
	return CheckResult{Passed: true, Actual: actual, Expected: "some changes"}
}

func checkWpEval(phpCode, expectedOutput string) CheckResult {
	// Use double quotes and escape any internal double quotes in phpCode
	escaped := strings.ReplaceAll(phpCode, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, `$`, `\$`)
	actual, err := shell(fmt.Sprintf(`locwp wp %s -- eval "%s"`, sitePort, escaped))
	if err != nil {
		return CheckResult{Passed: false, Error: err.Error()}
	}
	return CheckResult{Passed: strings.TrimSpace(actual) == strings.TrimSpace(expectedOutput), Actual: strings.TrimSpace(actual), Expected: strings.TrimSpace(expectedOutput)}
}

func checkDBQuery(sql, expected string) CheckResult {
	// Try wp db query first (MySQL), fallback to wp eval with $wpdb (SQLite compatible)
	actual, err := shell(fmt.Sprintf(`locwp wp %s -- db query "%s"`, sitePort, sql))
	if err != nil {
		// Fallback: use $wpdb->get_results() via wp eval
		escapedSQL := strings.ReplaceAll(sql, "'", "\\'")
		phpCode := fmt.Sprintf(`global $wpdb; $r = $wpdb->get_results("%s"); foreach($r as $row) { echo implode("\t", (array)$row) . "\n"; }`, escapedSQL)
		actual, err = shell(fmt.Sprintf(`locwp wp %s -- eval '%s'`, sitePort, phpCode))
		if err != nil {
			return CheckResult{Passed: false, Error: "db query failed on both MySQL and SQLite: " + err.Error()}
		}
	}
	if len(actual) > 500 {
		actual = actual[:500]
	}
	return CheckResult{Passed: strings.Contains(actual, expected), Actual: actual, Expected: "contains '" + expected + "'"}
}

// str converts an any to string.
func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
