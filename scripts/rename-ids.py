#!/usr/bin/env python3
"""Rename all task IDs to unified {PREFIX}-{seq} format, remove NEW tags, remove coverage stat."""

import json
import os
import re
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DOCS = ROOT / "docs"
MATRIX = ROOT / "topic-matrix.html"
TASKS_JSON = ROOT / "docs-tasks.json"

# Topic → prefix mapping
PREFIX = {
    "domain": "DOM",
    "hosting": "HOST",
    "wp-install": "INST",
    "site-settings": "SS",
    "user-management": "UM",
    "pages": "PG",
    "posts-taxonomy": "PT",
    "media": "MD",
    "menus-nav": "MN",
    "theme": "TH",
    "elementor": "ELE",
    "zeroy": "ZY",
    "plugins-basic": "PB",
    "acf": "ACF",
    "seo": "SEO",
    "google-analytics": "GA",
    "wp-config": "WPC",
    "security": "SEC",
    "backup-maintenance": "BM",
    "troubleshooting": "TS",
}


def load_task_map():
    """Parse TASK_MAP from topic-matrix.html JS."""
    text = MATRIX.read_text("utf-8")
    m = re.search(r"const TASK_MAP\s*=\s*\[(.+?)\];", text, re.DOTALL)
    if not m:
        raise RuntimeError("Cannot find TASK_MAP in topic-matrix.html")
    raw = m.group(1)
    tasks = []
    for item in re.finditer(
        r"\{\s*id:\s*'([^']+)'"
        r".*?topic:\s*'([^']+)'"
        r".*?newDiff:\s*(\d+)"
        r".*?desc:\s*'([^']+)'"
        r".*?isNew:\s*(true|false)",
        raw,
    ):
        tasks.append({
            "id": item.group(1),
            "topic": item.group(2),
            "diff": int(item.group(3)),
            "desc": item.group(4),
            "isNew": item.group(5) == "true",
        })
    return tasks


def build_rename_map(tasks):
    """Build old_id → new_id mapping."""
    by_topic = defaultdict(list)
    for t in tasks:
        by_topic[t["topic"]].append(t)

    rename = {}
    for topic, topic_tasks in by_topic.items():
        prefix = PREFIX[topic]
        for i, t in enumerate(topic_tasks, 1):
            new_id = f"{prefix}-{i}"
            rename[t["id"]] = new_id
    return rename


def rename_html_files(rename):
    """Rename all HTML doc files."""
    renamed = 0
    for old_id, new_id in rename.items():
        # Find which topic dir this file is in
        for topic_dir in DOCS.iterdir():
            if not topic_dir.is_dir():
                continue
            old_path = topic_dir / f"{old_id}.html"
            if old_path.exists():
                new_path = topic_dir / f"{new_id}.html"
                old_path.rename(new_path)
                renamed += 1
                break
    print(f"  Renamed {renamed} HTML files")


def update_doc_contents(rename):
    """Update all links and references inside doc files."""
    patched = 0
    for topic_dir in DOCS.iterdir():
        if not topic_dir.is_dir():
            continue
        for fpath in topic_dir.glob("*.html"):
            if fpath.name == "index.html":
                continue
            text = fpath.read_text("utf-8")
            original = text

            # Replace href="OLD_ID.html" with href="NEW_ID.html"
            for old_id, new_id in rename.items():
                text = text.replace(f'href="{old_id}.html"', f'href="{new_id}.html"')

            # Replace old ID in tid spans: <span class="tid">OLD_ID</span>
            for old_id, new_id in rename.items():
                text = text.replace(f">{old_id}<", f">{new_id}<")
                # Also in plain text references like "本文同时覆盖: ACF-cpt-1, ACF-cpt-2"
                text = text.replace(f" {old_id},", f" {new_id},")
                text = text.replace(f" {old_id}<", f" {new_id}<")
                text = text.replace(f":{old_id}", f":{new_id}")

            if text != original:
                fpath.write_text(text, "utf-8")
                patched += 1
    print(f"  Updated links in {patched} doc files")


def update_matrix(rename):
    """Update TASK_MAP IDs in topic-matrix.html, remove NEW styling, remove coverage stat."""
    text = MATRIX.read_text("utf-8")

    # Replace IDs in TASK_MAP
    for old_id, new_id in rename.items():
        text = text.replace(f"id:'{old_id}'", f"id:'{new_id}'")

    # Remove isNew visual styling - make all tasks look the same
    # Change: ${t.isNew ? 'is-new' : ''} → empty
    text = text.replace("${t.isNew ? 'is-new' : ''}", "")

    # Remove .is-new CSS rules
    text = re.sub(r"\s*\.cell \.task-item\.is-new \{[^}]*\}", "", text)
    text = re.sub(r"\s*\.cell \.task-item\.is-new \.tid::after \{[^}]*\}", "", text)

    # Remove NEW legend item
    text = re.sub(r'\s*<div class="item"><div class="dot" style="background:var\(--new\)"></div> 新增题目 \(L0\)</div>', "", text)

    # Remove "已有题目" legend (no longer meaningful distinction)
    text = re.sub(r'\s*<div class="item"><div class="dot" style="background:var\(--green\)"></div> 已有题目</div>', "", text)

    # Change coverage stat: remove it from the stats section
    # Remove the coverage stat div and the new/existing stats
    # Simplify: just show total + topics
    text = re.sub(
        r"document\.getElementById\('stats'\)\.innerHTML = `[^`]+`;",
        """document.getElementById('stats').innerHTML = `
    <div class="stat total"><div class="num">${totalAll}</div><div class="label">总题目数</div></div>
    <div class="stat topics"><div class="num">${totalTopics}</div><div class="label">知识主题</div></div>
  `;""",
        text,
    )

    # Remove unused JS variables (totalExisting, totalNew, coverage, filledCells)
    text = re.sub(r"\s*const totalExisting = TASK_MAP\.filter\(t => !t\.isNew\)\.length;", "", text)
    text = re.sub(r"\s*const totalNew = TASK_MAP\.filter\(t => t\.isNew\)\.length;", "", text)
    text = re.sub(r"\s*let filledCells = 0, totalCells = 0;", "", text)
    text = re.sub(
        r"\s*ALL_TOPICS\.forEach\(topic => \{\s*DIFFS\.forEach\(d => \{\s*totalCells\+\+;\s*if \(TASK_MAP\.some\(t => t\.topic === topic\.id && t\.newDiff === d\.level\)\) filledCells\+\+;\s*\}\);\s*\}\);",
        "",
        text,
    )
    text = re.sub(r"\s*const coverage = Math\.round\(filledCells / totalCells \* 100\);", "", text)

    # Simplify count coloring: remove isNew-based logic, just use accent
    text = text.replace(
        "const allNew = cellTasks.length > 0 && cellTasks.every(t => t.isNew);",
        "",
    )
    text = text.replace(
        "const hasNew = cellTasks.some(t => t.isNew);",
        "",
    )
    text = text.replace(
        "const countClass = allNew ? 'c-new' : hasNew ? 'c-mixed' : 'c-existing';",
        "const countClass = 'c-existing';",
    )

    # Update doc links with new IDs
    for old_id, new_id in rename.items():
        text = text.replace(f"/{old_id}.html", f"/{new_id}.html")

    MATRIX.write_text(text, "utf-8")
    print("  ✓ Updated topic-matrix.html")


def update_tasks_json(rename):
    """Update docs-tasks.json with new IDs and remove isNew."""
    tasks = json.loads(TASKS_JSON.read_text("utf-8"))
    for t in tasks:
        if t["id"] in rename:
            t["id"] = rename[t["id"]]
        if "isNew" in t:
            del t["isNew"]
    TASKS_JSON.write_text(json.dumps(tasks, ensure_ascii=False, indent=2) + "\n", "utf-8")
    print(f"  ✓ Updated docs-tasks.json ({len(tasks)} entries)")


def update_topic_indexes(rename):
    """Update topic index pages with new IDs."""
    for topic_dir in DOCS.iterdir():
        if not topic_dir.is_dir():
            continue
        idx = topic_dir / "index.html"
        if not idx.exists():
            continue
        text = idx.read_text("utf-8")
        for old_id, new_id in rename.items():
            text = text.replace(f'href="{old_id}.html"', f'href="{new_id}.html"')
            text = text.replace(f">{old_id}<", f">{new_id}<")
        idx.write_text(text, "utf-8")
    print("  ✓ Updated topic index pages")


def update_main_index(rename):
    """Update docs/index.html search data with new IDs."""
    idx = DOCS / "index.html"
    text = idx.read_text("utf-8")
    for old_id, new_id in rename.items():
        # In inline JSON: "id": "OLD"
        text = text.replace(f'"id": "{old_id}"', f'"id": "{new_id}"')
        # In href templates
        text = text.replace(f"/{old_id}.html", f"/{new_id}.html")
    idx.write_text(text, "utf-8")
    print("  ✓ Updated docs/index.html")


def main():
    print("Loading tasks...")
    tasks = load_task_map()
    print(f"  Found {len(tasks)} tasks")

    print("\nBuilding rename map...")
    rename = build_rename_map(tasks)
    # Print sample
    samples = list(rename.items())[:5]
    for old, new in samples:
        print(f"  {old} → {new}")
    print(f"  ... ({len(rename)} total)")

    print("\nStep 1: Renaming HTML files...")
    rename_html_files(rename)

    print("\nStep 2: Updating links inside doc files...")
    update_doc_contents(rename)

    print("\nStep 3: Updating topic-matrix.html...")
    update_matrix(rename)

    print("\nStep 4: Updating docs-tasks.json...")
    update_tasks_json(rename)

    print("\nStep 5: Updating index pages...")
    update_topic_indexes(rename)
    update_main_index(rename)

    print("\nDone! All IDs unified to {PREFIX}-{seq} format.")
    print("NEW tags and coverage stat removed.")


if __name__ == "__main__":
    main()
