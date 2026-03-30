#!/usr/bin/env python3
"""Migrate exam task bank from L-layer to topic-based structure."""

import json
import os
import re
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EXAM_DIR = ROOT / "out" / ".claude" / "references" / "tasks"

# ── Section → Topic mapping ────────────────────────────────────────────
SECTION_TOPIC = {
    "L1.1": "site-settings", "L1.2": "site-settings", "L1.3": "user-management",
    "L2.1": "pages", "L2.2": "posts-taxonomy", "L2.3": "media",
    "L2.4": "pages",  # block editor tasks → pages (closest fit)
    "L2.5": "menus-nav", "L2.6": "theme", "L2.7": "plugins-basic",
    "L3.1": "seo", "L3.2": "pages",  # content layout → pages
    "L3.3": "backup-maintenance", "L3.4": "theme",
    "L3.5": "pages",  # customer project → pages
    "L4.1": "theme", "L4.2": "theme", "L4.3": "theme",  # widgets → theme
    "L5.1": "plugins-basic",
    "L5.2": "acf", "L5.3": "acf", "L5.4": "acf", "L5.5": "acf",
    "L5.6": "acf", "L5.7": "acf", "L5.8": "acf", "L5.9": "acf",
    "L5.10": "plugins-basic",  # form plugins → plugins-basic
    "L5.11": "security", "L5.14": "acf",
    "L6.1": "wp-config", "L6.2": "wp-config", "L6.3": "wp-config", "L6.4": "wp-config",
    "L7.1": "backup-maintenance", "L7.2": "security",
    "L7.3": "backup-maintenance", "L7.4": "backup-maintenance",
    "L8.1": "troubleshooting", "L8.2": "troubleshooting", "L8.3": "troubleshooting",
    "L8.4": "troubleshooting", "L8.5": "troubleshooting",
}
for i in range(1, 20):
    SECTION_TOPIC[f"ELE.{i}"] = "elementor"
    SECTION_TOPIC[f"ZY.{i}"] = "zeroy"

PREFIX = {
    "domain": "DOM", "hosting": "HOST", "wp-install": "INST",
    "site-settings": "SS", "user-management": "UM",
    "pages": "PG", "posts-taxonomy": "PT", "media": "MD", "menus-nav": "MN",
    "theme": "TH", "elementor": "ELE", "zeroy": "ZY",
    "plugins-basic": "PB", "acf": "ACF", "seo": "SEO", "google-analytics": "GA",
    "wp-config": "WPC", "security": "SEC", "backup-maintenance": "BM", "troubleshooting": "TS",
}

TOPIC_NAME = {
    "domain": "域名管理", "hosting": "主机空间", "wp-install": "WordPress 安装",
    "site-settings": "站点配置", "user-management": "用户管理",
    "pages": "页面管理", "posts-taxonomy": "文章与分类", "media": "媒体管理", "menus-nav": "菜单与导航",
    "theme": "主题定制", "elementor": "Elementor 建站", "zeroy": "ZeroY AI 建站",
    "plugins-basic": "插件管理", "acf": "ACF 自定义字段", "seo": "SEO 优化", "google-analytics": "Google 全家桶",
    "wp-config": "wp-config 与数据库", "security": "安全加固",
    "backup-maintenance": "备份与维护", "troubleshooting": "故障排查",
}


def load_old_exam():
    """Load all old exam JSON files."""
    all_sections = {}
    for f in sorted(EXAM_DIR.iterdir()):
        if not f.name.endswith(".json"):
            continue
        data = json.loads(f.read_text("utf-8"))
        for key, entry in data.items():
            all_sections[key] = entry
            all_sections[key]["_file"] = f.name
    return all_sections


def migrate():
    print("Loading old exam task bank...")
    old = load_old_exam()
    print(f"  Found {len(old)} sections, {sum(len(e['tasks']) for e in old.values())} tasks")

    # Group tasks by new topic
    by_topic = defaultdict(list)
    orphans = []

    for section_key, entry in old.items():
        topic = SECTION_TOPIC.get(section_key)
        if not topic:
            orphans.append((section_key, entry["name"], len(entry["tasks"])))
            continue
        for task in entry["tasks"]:
            by_topic[topic].append({
                "old_id": task["id"],
                "old_section": section_key,
                "old_section_name": entry["name"],
                "task": task,
            })

    if orphans:
        print(f"\n  WARNING: {len(orphans)} unmapped sections:")
        for key, name, count in orphans:
            print(f"    {key}: {name} ({count} tasks)")

    # Assign new IDs and build new JSON files
    print(f"\nMigrating to {len(by_topic)} topic files...")
    id_mapping = {}  # old_id → new_id
    total_tasks = 0

    for topic in sorted(by_topic.keys()):
        tasks = by_topic[topic]
        prefix = PREFIX[topic]
        name = TOPIC_NAME[topic]

        # Build new task list with sequential IDs
        new_tasks = []
        for i, item in enumerate(tasks, 1):
            new_id = f"{prefix}-{i}"
            old_task = item["task"]
            id_mapping[item["old_id"]] = new_id

            new_task = {
                "id": new_id,
                "difficulty": old_task["difficulty"],
                "description": old_task["description"],
                "verify": old_task["verify"],
                "hints": old_task.get("hints", []),
                "on_pass_note": old_task.get("on_pass_note", ""),
            }
            # Preserve chain info
            if old_task.get("chain"):
                new_task["chain"] = old_task["chain"]
                new_task["chain_order"] = old_task.get("chain_order", 0)

            new_tasks.append(new_task)

        # Write new topic file
        new_data = {topic: {"name": name, "tasks": new_tasks}}
        out_path = EXAM_DIR / f"{topic}.json"
        out_path.write_text(json.dumps(new_data, ensure_ascii=False, indent=2) + "\n", "utf-8")
        total_tasks += len(new_tasks)
        print(f"  {topic}.json: {len(new_tasks)} tasks ({prefix}-1 .. {prefix}-{len(new_tasks)})")

    # Update chain references that use old task IDs
    for topic_file in EXAM_DIR.iterdir():
        if not topic_file.name.endswith(".json"):
            continue
        text = topic_file.read_text("utf-8")
        changed = False
        for old_id, new_id in id_mapping.items():
            if old_id in text:
                text = text.replace(old_id, new_id)
                changed = True
        if changed:
            topic_file.write_text(text, "utf-8")

    # Delete old L*.json files (but NOT the newly created topic files)
    new_topic_files = {f"{topic}.json" for topic in by_topic.keys()}
    deleted = []
    for f in list(EXAM_DIR.iterdir()):
        if not f.name.endswith(".json"):
            continue
        if f.name in new_topic_files:
            continue  # Keep newly created files
        f.unlink()
        deleted.append(f.name)
    print(f"\n  Deleted old files: {', '.join(deleted)}")

    # Summary
    print(f"\nDone! {total_tasks} tasks across {len(by_topic)} topic files")
    print(f"ID mapping: {len(id_mapping)} entries")

    # Save mapping for reference
    mapping_path = ROOT / "scripts" / "exam-id-mapping.json"
    json.dump(id_mapping, mapping_path.open("w"), ensure_ascii=False, indent=2)
    print(f"Mapping saved to {mapping_path}")


if __name__ == "__main__":
    migrate()
