#!/bin/bash
# 展示训练进度
set -e

PROGRESS_FILE="$HOME/.locwp/sites/wp-train/training/progress.json"

if [ ! -f "$PROGRESS_FILE" ]; then
  echo "还没有训练记录。输入 /wp-train 开始训练。"
  exit 0
fi

python3 << 'PYEOF'
import json

LAYER_NAMES = {
    1: "初识 WordPress",
    2: "内容管理",
    3: "文件系统与引导",
    4: "数据层",
    5: "主题引擎",
    6: "插件与扩展",
    7: "HTTP 与服务器层",
    8: "排障",
}

# Layer 内知识点数量（基于课纲）
LAYER_TOPICS = {1: 3, 2: 7, 3: 3, 4: 3, 5: 4, 6: 4, 7: 3, 8: 3}

import os
pf = os.path.expanduser("~/.locwp/sites/wp-train/training/progress.json")
with open(pf, "r") as f:
    data = json.load(f)

topics = data.get("topics", {})
stats = data.get("stats", {})

# 按 layer 统计
layer_mastered = {i: 0 for i in range(1, 9)}
layer_started = {i: 0 for i in range(1, 9)}
weak = []
recent = []

for topic_id, t in topics.items():
    try:
        layer = int(topic_id.split(".")[0].replace("L", ""))
    except (ValueError, IndexError):
        continue
    layer_started[layer] = layer_started.get(layer, 0) + 1
    if t.get("mastered"):
        layer_mastered[layer] = layer_mastered.get(layer, 0) + 1
        recent.append((t.get("last_attempt", ""), topic_id))
    elif t.get("attempts", 0) >= 2:
        weak.append((topic_id, t.get("consecutive_passes", 0), t.get("attempts", 0)))

print("═══ WordPress 训练进度 ═══\n")

total_topics = sum(LAYER_TOPICS.values())
total_mastered = sum(layer_mastered.values())

for i in range(1, 9):
    name = LAYER_NAMES[i]
    mastered = layer_mastered[i]
    total = LAYER_TOPICS[i]
    bar_len = 12
    filled = int(mastered / total * bar_len) if total > 0 else 0
    bar = "█" * filled + "░" * (bar_len - filled)

    if mastered == total:
        status = "完成"
    elif layer_started[i] > 0:
        status = f"{mastered}/{total} 掌握"
    else:
        status = "未开始"

    print(f"  Layer {i}: {name:<16} {bar}  {status}")

print(f"\n  总进度: {total_mastered}/{total_topics} 知识点 ({total_mastered/total_topics*100:.1f}%)")

ta = stats.get("total_attempts", 0)
tp = stats.get("total_passes", 0)
if ta > 0:
    print(f"  通过率: {tp}/{ta} ({tp/ta*100:.0f}%)")

if weak:
    print("\n─── 薄弱项 ───")
    for topic_id, cp, att in sorted(weak, key=lambda x: x[2], reverse=True):
        print(f"  ⚠ {topic_id} ({cp}/2 连续通过, 尝试 {att} 次)")

recent.sort(reverse=True)
if recent[:5]:
    print("\n─── 最近掌握 ───")
    for _, topic_id in recent[:5]:
        print(f"  ✅ {topic_id}")

print()
PYEOF
