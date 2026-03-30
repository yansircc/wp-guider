#!/usr/bin/env python3
"""Build docs navigation: links, indexes, prev/next, related docs."""

import json
import os
import re
from collections import defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DOCS = ROOT / "docs"
MATRIX = ROOT / "topic-matrix.html"
TASKS_JSON = ROOT / "docs-tasks.json"

# ── Categories (mirrors topic-matrix.html CATEGORIES) ──────────────────
CATEGORIES = [
    {
        "name": "🌐 基础设施",
        "topics": [
            {"id": "domain", "name": "域名管理", "sub": "注册 · DNS · 迁移"},
            {"id": "hosting", "name": "主机空间", "sub": "Hostinger · 面板 · FTP"},
            {"id": "wp-install", "name": "WordPress 安装", "sub": "安装 · 域名绑定 · SSL"},
        ],
    },
    {
        "name": "⚙️ 站点设置",
        "topics": [
            {"id": "site-settings", "name": "站点配置", "sub": "标题 · 时区 · 固定链接"},
            {"id": "user-management", "name": "用户管理", "sub": "角色 · 权限 · 个人资料"},
        ],
    },
    {
        "name": "📝 内容管理",
        "topics": [
            {"id": "pages", "name": "页面管理", "sub": "创建 · 层级 · 模板"},
            {"id": "posts-taxonomy", "name": "文章与分类", "sub": "文章 · 分类 · 标签"},
            {"id": "media", "name": "媒体管理", "sub": "上传 · 图库 · 尺寸"},
            {"id": "menus-nav", "name": "菜单与导航", "sub": "创建 · 排序 · 位置"},
        ],
    },
    {
        "name": "🎨 外观定制",
        "topics": [
            {"id": "theme", "name": "主题定制", "sub": "选型 · Customizer · Logo"},
            {"id": "elementor", "name": "Elementor 建站", "sub": "布局 · Widget · 模板"},
            {"id": "zeroy", "name": "ZeroY AI 建站", "sub": "页面 · 模板 · 交互"},
        ],
    },
    {
        "name": "🔌 插件与扩展",
        "topics": [
            {"id": "plugins-basic", "name": "插件管理", "sub": "安装 · 选型 · 表单"},
            {"id": "acf", "name": "ACF 自定义字段", "sub": "CPT · 分类法 · 字段组"},
            {"id": "seo", "name": "SEO 优化", "sub": "Yoast · 标题 · 描述"},
            {"id": "google-analytics", "name": "Google 全家桶", "sub": "GSC · GA4 · 数据分析"},
        ],
    },
    {
        "name": "🛠️ 运维与安全",
        "topics": [
            {"id": "wp-config", "name": "wp-config 与数据库", "sub": "常量 · 文件结构 · SQL"},
            {"id": "security", "name": "安全加固", "sub": "登录 · 权限 · 防护"},
            {"id": "backup-maintenance", "name": "备份与维护", "sub": "备份 · 更新 · Cron"},
            {"id": "troubleshooting", "name": "故障排查", "sub": "Debug · 冲突 · 白屏"},
        ],
    },
]

DIFF_LABELS = {1: "L1 入门", 2: "L2 基础", 3: "L3 进阶", 4: "L4 高级", 5: "L5 专家"}
DIFF_COLORS = {1: "d1", 2: "d2", 3: "d3", 4: "d4", 5: "d5"}

ALL_TOPICS = {t["id"]: t for cat in CATEGORIES for t in cat["topics"]}


def load_task_map():
    """Parse TASK_MAP from topic-matrix.html JS."""
    text = MATRIX.read_text("utf-8")
    # Extract the TASK_MAP array
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
        tasks.append(
            {
                "id": item.group(1),
                "topic": item.group(2),
                "diff": int(item.group(3)),
                "desc": item.group(4),
                "isNew": item.group(5) == "true",
            }
        )
    return tasks


def group_by_topic(tasks):
    """Group tasks by topic, preserving TASK_MAP order."""
    by_topic = defaultdict(list)
    for t in tasks:
        by_topic[t["topic"]].append(t)
    return dict(by_topic)


# ── Step 1: Patch topic-matrix.html ────────────────────────────────────

def patch_matrix(tasks):
    text = MATRIX.read_text("utf-8")

    # Add link styling if not already present
    link_css = ".cell .task-item a { color: inherit; text-decoration: none; display: block; } .cell .task-item a:hover { color: var(--accent); }"
    if ".task-item a {" not in text:
        text = text.replace(
            ".cell .task-item.is-new .tid::after",
            link_css + "\n  .cell .task-item.is-new .tid::after",
        )

    # Wrap task items in links
    old_template = '`<div class="task-item ${t.isNew ? \'is-new\' : \'\'}"><span class="tid">${t.id}</span> ${t.desc}</div>`'
    new_template = '`<div class="task-item ${t.isNew ? \'is-new\' : \'\'}"><a href="docs/${t.topic}/${t.id}.html"><span class="tid">${t.id}</span> ${t.desc}</a></div>`'

    if new_template not in text:
        text = text.replace(old_template, new_template)

    MATRIX.write_text(text, "utf-8")
    print(f"  ✓ Patched topic-matrix.html")


# ── Step 2: Generate docs/index.html ───────────────────────────────────

def gen_main_index(tasks, by_topic):
    total = len(tasks)
    topic_count = len(by_topic)

    # Build category cards HTML
    cards_html = ""
    for cat in CATEGORIES:
        cards_html += f'<div class="category-section"><h2>{cat["name"]}</h2><div class="topic-grid">\n'
        for t in cat["topics"]:
            tid = t["id"]
            topic_tasks = by_topic.get(tid, [])
            count = len(topic_tasks)
            diffs = sorted(set(tk["diff"] for tk in topic_tasks))
            diff_tags = " ".join(
                f'<span class="tag diff {DIFF_COLORS[d]}">{DIFF_LABELS[d]}</span>'
                for d in diffs
            )
            cards_html += f"""<a href="{tid}/index.html" class="topic-card">
  <div class="tc-name">{t["name"]}</div>
  <div class="tc-sub">{t["sub"]}</div>
  <div class="tc-meta"><span class="tc-count">{count} 篇</span>{diff_tags}</div>
</a>\n"""
        cards_html += "</div></div>\n"

    # Build search data (inline JSON)
    search_data = json.dumps(
        [{"id": t["id"], "topic": t["topic"], "diff": t["diff"], "desc": t["desc"]} for t in tasks],
        ensure_ascii=False,
    )

    html = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>WP Guider 知识库</title>
<link rel="stylesheet" href="style.css">
<style>
  .kb-header {{ text-align: center; margin-bottom: 40px; }}
  .kb-header h1 {{ font-size: 32px; margin-bottom: 8px; }}
  .kb-header p {{ color: var(--text2); }}
  .kb-stats {{ display: flex; gap: 24px; justify-content: center; margin-bottom: 32px; }}
  .kb-stats .stat {{ text-align: center; }}
  .kb-stats .num {{ font-size: 28px; font-weight: 700; color: var(--accent); }}
  .kb-stats .label {{ font-size: 12px; color: var(--text2); }}
  .search-wrap {{ max-width: 560px; margin: 0 auto 40px; }}
  .search-box {{ width: 100%; padding: 14px 20px; background: var(--surface); border: 1px solid var(--border); border-radius: 10px; color: var(--text); font-size: 15px; outline: none; }}
  .search-box:focus {{ border-color: var(--accent); }}
  .search-box::placeholder {{ color: var(--text3); }}
  .search-results {{ max-width: 560px; margin: -24px auto 40px; display: none; }}
  .search-results.active {{ display: block; }}
  .sr-item {{ display: block; padding: 10px 16px; background: var(--surface); border: 1px solid var(--border); border-top: none; text-decoration: none; color: var(--text2); font-size: 13px; }}
  .sr-item:first-child {{ border-top: 1px solid var(--border); border-radius: 10px 10px 0 0; }}
  .sr-item:last-child {{ border-radius: 0 0 10px 10px; }}
  .sr-item:hover {{ background: var(--surface2); color: var(--text); }}
  .sr-item .sr-id {{ color: var(--text3); font-family: monospace; font-size: 11px; }}
  .sr-item .sr-topic {{ color: var(--accent); font-size: 11px; margin-left: 8px; }}
  .category-section {{ margin-bottom: 40px; }}
  .category-section h2 {{ font-size: 20px; margin-bottom: 16px; }}
  .topic-grid {{ display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; }}
  .topic-card {{ display: block; padding: 20px; background: var(--surface); border: 1px solid var(--border); border-radius: 12px; text-decoration: none; transition: border-color 0.15s; }}
  .topic-card:hover {{ border-color: var(--accent); }}
  .tc-name {{ font-size: 16px; font-weight: 600; color: var(--text); margin-bottom: 4px; }}
  .tc-sub {{ font-size: 12px; color: var(--text3); margin-bottom: 12px; }}
  .tc-meta {{ display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }}
  .tc-count {{ font-size: 13px; color: var(--accent); font-weight: 600; }}
  .back-link {{ display: inline-block; margin-bottom: 24px; color: var(--text2); text-decoration: none; font-size: 13px; }}
  .back-link:hover {{ color: var(--accent); }}
</style>
</head>
<body>
<div class="container" style="max-width:960px">
  <a href="../topic-matrix.html" class="back-link">← 返回矩阵视图</a>
  <div class="kb-header">
    <h1>WP Guider 知识库</h1>
    <p>WordPress 培训系统 · {total} 篇文档 · {topic_count} 个主题</p>
  </div>
  <div class="kb-stats">
    <div class="stat"><div class="num">{total}</div><div class="label">文档总数</div></div>
    <div class="stat"><div class="num">{topic_count}</div><div class="label">知识主题</div></div>
    <div class="stat"><div class="num">5</div><div class="label">难度等级</div></div>
  </div>
  <div class="search-wrap">
    <input type="text" class="search-box" placeholder="搜索文档… 输入关键词、任务 ID 或主题名" id="search">
  </div>
  <div class="search-results" id="results"></div>
  {cards_html}
</div>
<script>
const TASKS = {search_data};
const searchEl = document.getElementById('search');
const resultsEl = document.getElementById('results');
searchEl.addEventListener('input', function() {{
  const q = this.value.trim().toLowerCase();
  if (q.length < 2) {{ resultsEl.className = 'search-results'; resultsEl.innerHTML = ''; return; }}
  const hits = TASKS.filter(t => t.id.toLowerCase().includes(q) || t.desc.toLowerCase().includes(q) || t.topic.toLowerCase().includes(q)).slice(0, 15);
  if (hits.length === 0) {{ resultsEl.className = 'search-results'; resultsEl.innerHTML = ''; return; }}
  resultsEl.className = 'search-results active';
  resultsEl.innerHTML = hits.map(t =>
    `<a class="sr-item" href="${{t.topic}}/${{t.id}}.html"><span class="sr-id">${{t.id}}</span><span class="sr-topic">${{t.topic}}</span> ${{t.desc}}</a>`
  ).join('');
}});
</script>
</body>
</html>"""

    (DOCS / "index.html").write_text(html, "utf-8")
    print(f"  ✓ Generated docs/index.html")


# ── Step 3: Generate topic index pages ─────────────────────────────────

def gen_topic_indexes(by_topic):
    for topic_id, tasks in by_topic.items():
        meta = ALL_TOPICS.get(topic_id)
        if not meta:
            continue

        # Group by difficulty
        by_diff = defaultdict(list)
        for t in tasks:
            by_diff[t["diff"]].append(t)

        list_html = ""
        for diff in sorted(by_diff.keys()):
            dtasks = by_diff[diff]
            list_html += f'<h3 class="diff-heading"><span class="tag diff {DIFF_COLORS[diff]}">{DIFF_LABELS[diff]}</span> ({len(dtasks)} 篇)</h3>\n'
            list_html += '<div class="doc-list">\n'
            for t in dtasks:
                list_html += f'<a href="{t["id"]}.html" class="dl-item"><span class="dl-id">{t["id"]}</span>{t["desc"]}</a>\n'
            list_html += "</div>\n"

        # Find category name
        cat_name = ""
        for cat in CATEGORIES:
            if any(t["id"] == topic_id for t in cat["topics"]):
                cat_name = cat["name"]
                break

        html = f"""<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{meta["name"]} - WP Guider 知识库</title>
<link rel="stylesheet" href="../style.css">
<style>
  .topic-header {{ margin-bottom: 32px; }}
  .topic-header h1 {{ font-size: 28px; margin-bottom: 4px; }}
  .topic-header .sub {{ color: var(--text3); font-size: 14px; }}
  .topic-header .cat {{ color: var(--text2); font-size: 13px; margin-bottom: 8px; }}
  .diff-heading {{ font-size: 16px; font-weight: 600; margin: 28px 0 12px; display: flex; align-items: center; gap: 10px; }}
  .doc-list {{ display: flex; flex-direction: column; gap: 4px; }}
  .dl-item {{ display: block; padding: 12px 16px; background: var(--surface); border: 1px solid var(--border); border-radius: 8px; text-decoration: none; color: var(--text2); font-size: 14px; transition: border-color 0.15s; }}
  .dl-item:hover {{ border-color: var(--accent); color: var(--text); }}
  .dl-id {{ color: var(--text3); font-family: monospace; font-size: 12px; margin-right: 10px; }}
  .back-link {{ display: inline-block; margin-bottom: 24px; color: var(--text2); text-decoration: none; font-size: 13px; }}
  .back-link:hover {{ color: var(--accent); }}
</style>
</head>
<body>
<nav class="topnav">
  <a href="../../topic-matrix.html" class="brand">WP Guider</a>
  <span class="sep">/</span>
  <a href="../index.html">知识库</a>
  <span class="sep">/</span>
  <span class="current">{meta["name"]}</span>
</nav>
<div class="container">
  <div class="topic-header">
    <div class="cat">{cat_name}</div>
    <h1>{meta["name"]}</h1>
    <div class="sub">{meta["sub"]} · 共 {len(tasks)} 篇文档</div>
  </div>
  {list_html}
</div>
</body>
</html>"""

        topic_dir = DOCS / topic_id
        topic_dir.mkdir(exist_ok=True)
        (topic_dir / "index.html").write_text(html, "utf-8")

    print(f"  ✓ Generated {len(by_topic)} topic index pages")


# ── Step 4: Inject prev/next + related into doc files ──────────────────

def inject_navigation(by_topic):
    patched = 0
    skipped = 0

    for topic_id, tasks in by_topic.items():
        for i, task in enumerate(tasks):
            fpath = DOCS / topic_id / f"{task['id']}.html"
            if not fpath.exists():
                skipped += 1
                continue

            html = fpath.read_text("utf-8")

            # Skip if already patched
            if 'class="doc-nav"' in html:
                continue

            # Build prev/next
            nav_html = '\n  <div class="doc-nav">\n'
            if i > 0:
                prev_t = tasks[i - 1]
                nav_html += f'    <a href="{prev_t["id"]}.html" class="prev"><div class="label">上一篇</div><div class="title">{prev_t["desc"]}</div></a>\n'
            else:
                nav_html += '    <span></span>\n'
            if i < len(tasks) - 1:
                next_t = tasks[i + 1]
                nav_html += f'    <a href="{next_t["id"]}.html" class="next"><div class="label">下一篇</div><div class="title">{next_t["desc"]}</div></a>\n'
            nav_html += "  </div>\n"

            # Build related (same topic, different difficulty, max 4)
            related = [
                t for t in tasks
                if t["id"] != task["id"] and t["diff"] != task["diff"]
            ]
            # Prefer adjacent difficulties
            related.sort(key=lambda t: abs(t["diff"] - task["diff"]))
            related = related[:4]

            related_html = ""
            if related:
                related_html = '  <div class="related"><h3>相关文档</h3><div class="related-list">\n'
                for r in related:
                    related_html += f'    <a href="{r["id"]}.html">{r["desc"]}</a>\n'
                related_html += "  </div></div>\n"

            # Inject before closing </div></body></html>
            # Find the last </div> before </body>
            insertion = nav_html + related_html
            html = re.sub(
                r"(</div>\s*</body>)",
                insertion + r"\1",
                html,
                count=1,
            )

            # Update breadcrumb: topic link → topic index
            # Pattern: <a href="../../topic-matrix.html">TopicName</a>
            html = re.sub(
                r'<a href="../../topic-matrix\.html">([^<]+)</a>\s*<span class="sep">/</span>\s*<span class="current">',
                r'<a href="index.html">\1</a>\n  <span class="sep">/</span>\n  <span class="current">',
                html,
            )

            fpath.write_text(html, "utf-8")
            patched += 1

    print(f"  ✓ Patched {patched} doc files (skipped {skipped} missing)")


# ── Main ───────────────────────────────────────────────────────────────

def main():
    print("Building docs navigation...")
    tasks = load_task_map()
    print(f"  Loaded {len(tasks)} tasks from topic-matrix.html")

    by_topic = group_by_topic(tasks)
    print(f"  Found {len(by_topic)} topics")

    print("\nStep 1: Patching topic-matrix.html links...")
    patch_matrix(tasks)

    print("\nStep 2: Generating docs/index.html...")
    gen_main_index(tasks, by_topic)

    print("\nStep 3: Generating topic index pages...")
    gen_topic_indexes(by_topic)

    print("\nStep 4: Injecting prev/next + related into doc files...")
    inject_navigation(by_topic)

    print("\nDone!")


if __name__ == "__main__":
    main()
