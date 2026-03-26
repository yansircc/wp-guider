#!/bin/bash
# 更新训练进度
# 用法: update-progress.sh <topic> <passed|failed>
set -e

TOPIC="$1"
RESULT="$2"  # "passed" or "failed"
PROGRESS_FILE="$HOME/.locwp/sites/wp-train/training/progress.json"

if [ -z "$TOPIC" ] || [ -z "$RESULT" ]; then
  echo "Usage: update-progress.sh <topic> <passed|failed>"
  exit 1
fi

mkdir -p "$(dirname "$PROGRESS_FILE")"

# 初始化进度文件（如果不存在）
if [ ! -f "$PROGRESS_FILE" ]; then
  cat > "$PROGRESS_FILE" << 'EOF'
{
  "topics": {},
  "stats": {
    "total_attempts": 0,
    "total_passes": 0,
    "topics_mastered": 0,
    "current_layer": 1
  }
}
EOF
fi

# 用 python3 更新 JSON（macOS 自带）
python3 << PYEOF
import json
from datetime import datetime, timezone

with open("$PROGRESS_FILE", "r") as f:
    data = json.load(f)

topic = "$TOPIC"
passed = "$RESULT" == "passed"

topics = data.setdefault("topics", {})
stats = data.setdefault("stats", {"total_attempts": 0, "total_passes": 0, "topics_mastered": 0, "current_layer": 1})

t = topics.setdefault(topic, {
    "attempts": 0,
    "passes": 0,
    "consecutive_passes": 0,
    "mastered": False,
    "last_attempt": None
})

t["attempts"] += 1
stats["total_attempts"] += 1
t["last_attempt"] = datetime.now(timezone.utc).isoformat()

if passed:
    t["passes"] += 1
    t["consecutive_passes"] += 1
    stats["total_passes"] += 1
    if t["consecutive_passes"] >= 2:
        if not t["mastered"]:
            t["mastered"] = True
            stats["topics_mastered"] += 1
else:
    t["consecutive_passes"] = 0

# 更新 current_layer
max_layer = 1
for k, v in topics.items():
    if v.get("mastered"):
        layer = int(k.split(".")[0].replace("L", ""))
        max_layer = max(max_layer, layer)
stats["current_layer"] = max_layer

with open("$PROGRESS_FILE", "w") as f:
    json.dump(data, f, indent=2, ensure_ascii=False)

status = "✅ mastered" if t["mastered"] else f"🔄 {t['consecutive_passes']}/2"
print(f"Progress updated: {topic} → {status} (attempts: {t['attempts']})")
PYEOF
