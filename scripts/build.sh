#!/bin/bash
# build.sh — 构建消费者产物到 out/.claude/
set -e

cd "$(dirname "$0")/.."

echo ":: 编译 wp-train ..."
go build -o out/.claude/scripts/wp-train ./cmd/wp-train/

echo ":: 校验产物完整性 ..."
MISSING=0

check() {
  if [ ! -s "$1" ]; then
    echo "   ✘ $1"
    MISSING=$((MISSING + 1))
  fi
}

# 二进制
check "out/.claude/scripts/wp-train"

# 核心配置
check "out/.claude/CLAUDE.md"
check "out/README.md"

# 题库（动态扫描 tasks/ 目录）
TASK_DIR="out/.claude/references/tasks"
TASK_FILES=$(find "$TASK_DIR" -name "*.json" 2>/dev/null | wc -l | tr -d ' ')
if [ "$TASK_FILES" -eq 0 ]; then
  echo "   ✘ $TASK_DIR/ 无 JSON 文件"
  MISSING=$((MISSING + 1))
fi

# Skills（动态扫描）
for skill_dir in out/.claude/skills/*/; do
  check "${skill_dir}SKILL.md"
done

# 课纲
check "out/.claude/skills/wp-train/references/curriculum.md"

if [ "$MISSING" -gt 0 ]; then
  echo ":: $MISSING 个文件缺失或为空"
  exit 1
fi

echo ":: 冒烟验证 ..."
STATUS=$(out/.claude/scripts/wp-train status 2>&1) || { echo "   ✘ status 命令失败"; exit 1; }
echo "$STATUS" | python3 -c "import json,sys; json.load(sys.stdin)" 2>/dev/null || { echo "   ✘ status 输出非法 JSON"; exit 1; }

# 摘要
BIN_SIZE=$(du -h out/.claude/scripts/wp-train | cut -f1)
TASK_COUNT=$(python3 -c "
import json, os
total = 0
for f in os.listdir('$TASK_DIR'):
    if f.endswith('.json'):
        d = json.load(open(f'$TASK_DIR/{f}'))
        total += sum(len(t['tasks']) for t in d.values())
print(total)
")
SKILL_COUNT=$(ls -d out/.claude/skills/*/SKILL.md 2>/dev/null | wc -l | tr -d ' ')

echo ""
echo ":: 构建完成"
echo "   二进制: $BIN_SIZE"
echo "   题库: $TASK_FILES 个文件, $TASK_COUNT 题"
echo "   Skills: $SKILL_COUNT 个"
echo "   产物: out/.claude/"
