---
name: wp-checkpoint
description: 管理训练站点快照。保存、还原、列出检查点。当教练说"保存进度"、"还原"、"恢复"、"回退"、"checkpoint"时触发。
---

# wp-checkpoint

## 保存

```bash
.claude/scripts/wp-train checkpoint save <name>
```

保存当前数据库 + wp-content 文件 + wp-config.php。告诉教练检查点已保存。

## 还原

```bash
.claude/scripts/wp-train checkpoint restore <name>
```

还原数据库、文件和配置到指定检查点。告诉教练站点已恢复。

## 列出

```bash
.claude/scripts/wp-train checkpoint list
```

返回所有检查点名称和创建时间。
