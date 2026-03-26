#!/bin/bash
# 按训练层级预置内容
set -e

SITE="wp-train"
LAYER="${1:-1}"

echo ":: 为 Layer $LAYER 预置内容 ..."

case "$LAYER" in
  1|2|3)
    echo ":: Layer 1-3 使用干净站点，无需预置"
    ;;
  4)
    echo ":: 预置页面和文章 ..."
    locwp wp "$SITE" -- post create --post_type=page --post_title="Home" --post_status=publish --porcelain
    locwp wp "$SITE" -- post create --post_type=page --post_title="About" --post_status=publish --porcelain
    locwp wp "$SITE" -- post create --post_type=page --post_title="Contact" --post_status=publish --porcelain
    locwp wp "$SITE" -- post create --post_type=page --post_title="Blog" --post_status=publish --porcelain

    # 设置首页为静态页面
    HOME_ID=$(locwp wp "$SITE" -- post list --post_type=page --title="Home" --format=ids)
    BLOG_ID=$(locwp wp "$SITE" -- post list --post_type=page --title="Blog" --format=ids)
    locwp wp "$SITE" -- option update show_on_front page
    locwp wp "$SITE" -- option update page_on_front "$HOME_ID"
    locwp wp "$SITE" -- option update page_for_posts "$BLOG_ID"

    # 分类和文章
    locwp wp "$SITE" -- term create category "News" --porcelain
    locwp wp "$SITE" -- term create category "Tutorial" --porcelain
    locwp wp "$SITE" -- term create post_tag "WordPress" --porcelain
    locwp wp "$SITE" -- term create post_tag "PHP" --porcelain

    NEWS_ID=$(locwp wp "$SITE" -- term list category --name="News" --format=ids)
    for i in 1 2 3; do
      locwp wp "$SITE" -- post create --post_title="Sample Post $i" --post_status=publish --post_category="$NEWS_ID" --porcelain
    done

    echo ":: 预置完成：4 页面 + 2 分类 + 2 标签 + 3 文章，静态首页已设置"
    ;;
  5)
    echo ":: Layer 5 复用 Layer 4 预置 ..."
    bash "$(dirname "$0")/seed-content.sh" 4
    ;;
  6|7)
    echo ":: Layer 6-7 复用 Layer 4 预置 + 额外插件 ..."
    bash "$(dirname "$0")/seed-content.sh" 4

    locwp wp "$SITE" -- plugin install query-monitor --activate
    echo ":: 已安装 Query Monitor"
    ;;
  8)
    echo ":: Layer 8 排障训练，复用 Layer 6 预置 + 开启调试 ..."
    bash "$(dirname "$0")/seed-content.sh" 6

    locwp wp "$SITE" -- config set WP_DEBUG true --raw
    locwp wp "$SITE" -- config set WP_DEBUG_LOG true --raw
    echo ":: 已开启 WP_DEBUG + WP_DEBUG_LOG"
    ;;
  *)
    echo ":: 未知 Layer: $LAYER"
    exit 1
    ;;
esac
