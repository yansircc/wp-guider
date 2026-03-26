---
name: wp-site
description: 管理 WordPress 训练站点。创建、重置、删除训练环境。当教练说"创建站点"、"重置站点"、"删除站点"、"初始化"时触发。
---

# wp-site

运行 `.claude/scripts/wp-train init` 创建/重置训练站点。

命令会自动：删除旧站（如果存在）→ 创建新站 → git init wp-content → 初始化 SQLite。

返回 JSON，包含 URL、admin 地址、credentials。把这些信息告诉教练。
