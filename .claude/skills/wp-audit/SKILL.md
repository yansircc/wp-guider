---
name: wp-audit
description: 提取 agent 决策链用于调试。查看最近会话中 agent 调了什么命令、拿到什么结果、做了什么判断。当说"audit"、"决策链"、"debug 会话"、"看看 agent 做了什么"时触发。
---

# wp-audit

运行 `bash scripts/audit.sh` 查看最近一次 agent 会话的决策链。

可选参数：
- `bash scripts/audit.sh <session-id>` — 指定会话
- `bash scripts/audit.sh -n 20` — 限制条数

输出格式：
- `[human]` 教练输入
- `[agent]` agent 的工具调用和推理
- `[tool]` 工具返回结果
- `★` 标记高亮所有 `wp-train` CLI 调用

用于排查 agent 为什么做了某个判断、是否正确调用了验证命令、反馈是否恰当。
