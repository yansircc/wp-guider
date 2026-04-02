---
name: wp-checkpoint
description: 管理训练站点快照。保存、还原、列出检查点。当教练说"保存进度"、"还原"、"恢复"、"回退"、"恢复最新"、"checkpoint"时触发。
---

# wp-checkpoint

## 恢复到最新自动存档

网站崩了或做爆了，一键恢复到上一次 wp-check 通过时的状态：

```bash
.claude/scripts/wp-train checkpoint restore-last
```

只恢复当前操作的站点（main / elementor / zeroy），不影响其他端口。

## 保存手动检查点

```bash
.claude/scripts/wp-train checkpoint save <name>
```

## 恢复指定检查点

```bash
.claude/scripts/wp-train checkpoint restore <name>
```

## 列出所有检查点

```bash
.claude/scripts/wp-train checkpoint list
```

按站点分组显示所有存档名称和创建时间。
