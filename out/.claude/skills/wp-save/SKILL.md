---
name: wp-save
description: 手动保存或恢复当前站点的命名存档。只影响当前操作的端口，不影响其他站点。当教练说"存档"、"保存一个"、"wp-save"、"读档"、"恢复到 X"时触发。
---

# wp-save

## 保存命名存档

教练提供存档名（如 `homepage-done`），运行：

```bash
.claude/scripts/wp-train checkpoint save <name>
```

告诉教练已保存，存档名是什么，当前是哪个站点（端口）。

## 恢复命名存档

教练说恢复到某个存档名，运行：

```bash
.claude/scripts/wp-train checkpoint restore <name>
```

告诉教练站点已恢复到该存档。

## 列出所有存档

```bash
.claude/scripts/wp-train checkpoint list
```

展示所有站点（main / elementor / zeroy）下的存档列表，包含创建时间。

---

**注意**：`latest` 是系统自动覆盖的存档（每次 wp-check 通过时更新），不建议手动操作。手动存档用自定义名字。
