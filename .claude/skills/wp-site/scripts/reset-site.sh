#!/bin/bash
# 重置训练站点：删除旧站（如果存在）→ 创建新站 → 验证
set -e

SITE="wp-train"
TRAINING_DIR="$HOME/.locwp/sites/$SITE/training"

# 检查是否已存在
if locwp list 2>/dev/null | grep -q "^$SITE "; then
  echo ":: 删除旧站点 $SITE ..."
  locwp delete "$SITE"
fi

echo ":: 创建站点 $SITE ..."
locwp add "$SITE" --pass admin

echo ":: 验证站点 ..."
URL=$(locwp wp "$SITE" -- option get siteurl 2>/dev/null)
echo ":: 站点就绪: $URL"
echo ":: 后台: ${URL}/wp-admin/"
echo ":: 账号: admin / admin"

# 恢复 training 目录（如果之前有进度）
if [ -f "/tmp/wp-train-progress-backup.json" ]; then
  mkdir -p "$TRAINING_DIR"
  cp /tmp/wp-train-progress-backup.json "$TRAINING_DIR/progress.json"
  echo ":: 已恢复训练进度"
fi
