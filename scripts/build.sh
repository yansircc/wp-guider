#!/bin/bash
# build.sh — 构建消费者产物到 out/.claude/
set -e

cd "$(dirname "$0")/.."

echo ":: 编译 wp-train ..."
go build -o out/.claude/scripts/wp-train ./cmd/wp-train/

echo ":: 校验产物完整性 ..."
REQUIRED=(
  "out/.claude/scripts/wp-train"
  "out/.claude/references/task-bank.json"
  "out/.claude/skills/wp-site/SKILL.md"
  "out/.claude/skills/wp-train/SKILL.md"
  "out/.claude/skills/wp-train/references/curriculum.md"
  "out/.claude/skills/wp-check/SKILL.md"
  "out/.claude/skills/wp-progress/SKILL.md"
  "out/.claude/skills/wp-inject/SKILL.md"
  "out/.claude/skills/wp-checkpoint/SKILL.md"
  "out/.claude/CLAUDE.md"
)

MISSING=0
for f in "${REQUIRED[@]}"; do
  if [ ! -s "$f" ]; then
    echo "   ✘ $f"
    MISSING=$((MISSING + 1))
  fi
done

if [ "$MISSING" -gt 0 ]; then
  echo ":: $MISSING 个文件缺失或为空"
  exit 1
fi

echo ":: 冒烟验证 ..."
STATUS=$(out/.claude/scripts/wp-train status 2>&1) || { echo "   ✘ status 命令失败"; exit 1; }
echo "$STATUS" | python3 -c "import json,sys; json.load(sys.stdin)" 2>/dev/null || { echo "   ✘ status 输出非法 JSON"; exit 1; }

# 摘要
BIN_SIZE=$(du -h out/.claude/scripts/wp-train | cut -f1)
TASK_COUNT=$(python3 -c "import json; d=json.load(open('out/.claude/references/task-bank.json')); print(sum(len(t['tasks']) for t in d.values()))")
echo ""
echo ":: 构建完成"
echo "   二进制: $BIN_SIZE"
echo "   题目数: $TASK_COUNT"
echo "   产物: out/.claude/"
