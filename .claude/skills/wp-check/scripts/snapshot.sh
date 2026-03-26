#!/bin/bash
# 站点状态快照：一次性获取 WordPress 站点的关键状态
set -e

SITE="${1:-wp-train}"

echo "=== WordPress Snapshot: $SITE ==="
echo ""

echo "--- Pages ---"
locwp wp "$SITE" -- post list --post_type=page --fields=ID,post_title,post_status,post_name --format=json 2>/dev/null || echo "[]"
echo ""

echo "--- Posts ---"
locwp wp "$SITE" -- post list --post_type=post --fields=ID,post_title,post_status,post_date --format=json 2>/dev/null || echo "[]"
echo ""

echo "--- Themes ---"
locwp wp "$SITE" -- theme list --fields=name,status,version --format=json 2>/dev/null || echo "[]"
echo ""

echo "--- Plugins ---"
locwp wp "$SITE" -- plugin list --fields=name,status,version --format=json 2>/dev/null || echo "[]"
echo ""

echo "--- Menus ---"
MENUS=$(locwp wp "$SITE" -- menu list --format=json 2>/dev/null || echo "[]")
echo "$MENUS"
# 列出每个菜单的项
echo "$MENUS" | python3 -c "
import json, sys, subprocess
menus = json.load(sys.stdin)
for m in menus:
    mid = m.get('term_id', m.get('id', ''))
    if mid:
        print(f'\n--- Menu Items (ID={mid}, {m.get(\"name\",\"\")}) ---')
        sys.stdout.flush()
        subprocess.run(['locwp', 'wp', '$SITE', '--', 'menu', 'item', 'list', str(mid), '--format=json'], check=False)
" 2>/dev/null || true
echo ""

echo "--- Key Options ---"
for key in blogname blogdescription siteurl home show_on_front page_on_front page_for_posts permalink_structure template stylesheet; do
  val=$(locwp wp "$SITE" -- option get "$key" 2>/dev/null || echo "(not set)")
  printf "  %-20s = %s\n" "$key" "$val"
done
echo ""

echo "--- Categories ---"
locwp wp "$SITE" -- term list category --fields=term_id,name,slug,count --format=json 2>/dev/null || echo "[]"
echo ""

echo "--- Tags ---"
locwp wp "$SITE" -- term list post_tag --fields=term_id,name,slug,count --format=json 2>/dev/null || echo "[]"
echo ""

echo "--- Users ---"
locwp wp "$SITE" -- user list --fields=ID,user_login,roles --format=json 2>/dev/null || echo "[]"
echo ""

echo "--- Rewrite Rules (top 10) ---"
locwp wp "$SITE" -- rewrite list --format=json 2>/dev/null | python3 -c "import json,sys; rules=json.load(sys.stdin); [print(f'  {r[\"match\"]} → {r[\"query\"]}') for r in rules[:10]]" 2>/dev/null || echo "  (none)"
echo ""

echo "--- WP Debug ---"
for key in WP_DEBUG WP_DEBUG_LOG WP_DEBUG_DISPLAY; do
  val=$(locwp wp "$SITE" -- config get "$key" 2>/dev/null || echo "(not set)")
  printf "  %-20s = %s\n" "$key" "$val"
done
