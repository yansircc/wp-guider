#!/bin/bash
# audit.sh — 提取 wp-guider agent 的决策链
# 用法: ./scripts/audit.sh [session-id] [-n lines]
#
# 不带参数: 最近一次会话
# session-id: 指定会话
# -n: 限制输出条数

set -e

PROJECT_HASH="-Users-yansir-code-52-wp-guider"
SESSIONS_DIR="$HOME/.claude/projects/$PROJECT_HASH"
N=50

# Parse args
SESSION_ID=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -n) N="$2"; shift 2 ;;
    *) SESSION_ID="$1"; shift ;;
  esac
done

# Find JSONL file
if [ -n "$SESSION_ID" ]; then
  JSONL="$SESSIONS_DIR/${SESSION_ID}.jsonl"
else
  JSONL=$(ls -t "$SESSIONS_DIR"/*.jsonl 2>/dev/null | head -1)
fi

if [ ! -f "$JSONL" ]; then
  echo "No session found" >&2
  exit 1
fi

SESSION=$(basename "$JSONL" .jsonl)
echo "Session: $SESSION"
echo "File: $JSONL"
echo ""

python3 - "$JSONL" "$N" << 'PYEOF'
import json, sys

jsonl_path = sys.argv[1]
max_lines = int(sys.argv[2])

entries = []

with open(jsonl_path) as f:
    for line in f:
        try:
            obj = json.loads(line)
        except json.JSONDecodeError:
            continue

        t = obj.get("type")
        ts = obj.get("timestamp", "")
        if ts:
            ts = ts[11:19]  # HH:MM:SS

        if t == "user":
            msg = obj.get("message", {})
            content = msg.get("content", "")
            if isinstance(content, str) and content.strip():
                entries.append({"ts": ts, "role": "human", "action": "message", "detail": content[:120]})
            elif isinstance(content, list):
                for block in content:
                    if not isinstance(block, dict):
                        continue
                    bt = block.get("type")
                    if bt == "text":
                        text = block.get("text", "")
                        # Skip system prompts and skill loading boilerplate
                        if text.startswith("Base directory") or text.startswith("<"):
                            continue
                        entries.append({"ts": ts, "role": "human", "action": "message", "detail": text[:120]})
                    elif bt == "tool_result":
                        c = block.get("content", "")
                        if isinstance(c, list) and c:
                            c = c[0].get("text", "") if isinstance(c[0], dict) else str(c[0])
                        c = str(c)
                        # Only show wp-train results and key tool results
                        if "wp-train" in c or "status" in c[:50] or "passed" in c[:50] or "{" in c[:20]:
                            entries.append({"ts": ts, "role": "tool", "action": "result", "detail": c[:200]})

        elif t == "assistant":
            msg = obj.get("message", {})
            content = msg.get("content", [])
            if isinstance(content, str):
                entries.append({"ts": ts, "role": "agent", "action": "reply", "detail": content[:120]})
            elif isinstance(content, list):
                for block in content:
                    if not isinstance(block, dict):
                        continue
                    bt = block.get("type")
                    if bt == "text":
                        text = block.get("text", "").strip()
                        if text:
                            entries.append({"ts": ts, "role": "agent", "action": "think", "detail": text[:120]})
                    elif bt == "tool_use":
                        name = block.get("name", "")
                        inp = block.get("input", {})
                        # Highlight wp-train calls
                        if name == "Bash":
                            cmd = inp.get("command", "")
                            if "wp-train" in cmd:
                                entries.append({"ts": ts, "role": "agent", "action": f"call:{name}", "detail": cmd[:150], "highlight": True})
                            else:
                                entries.append({"ts": ts, "role": "agent", "action": f"call:{name}", "detail": cmd[:100]})
                        elif name == "Skill":
                            entries.append({"ts": ts, "role": "agent", "action": f"call:{name}", "detail": json.dumps(inp, ensure_ascii=False)[:100]})
                        else:
                            entries.append({"ts": ts, "role": "agent", "action": f"call:{name}", "detail": json.dumps(inp, ensure_ascii=False)[:80]})

# Output
ROLE_COLORS = {
    "human": "\033[36m",   # cyan
    "agent": "\033[33m",   # yellow
    "tool":  "\033[32m",   # green
}
RESET = "\033[0m"
BOLD = "\033[1m"

shown = 0
for e in entries[-max_lines:]:
    role = e["role"]
    color = ROLE_COLORS.get(role, "")
    highlight = e.get("highlight", False)

    prefix = f'{e["ts"]} {color}[{role:5s}]{RESET}'
    action = e["action"]
    detail = e["detail"].replace("\n", " ↵ ")

    if highlight:
        print(f'{prefix} {BOLD}★ {action}: {detail}{RESET}')
    else:
        print(f'{prefix} {action}: {detail}')
    shown += 1

print(f"\n({shown} entries shown)")
PYEOF
