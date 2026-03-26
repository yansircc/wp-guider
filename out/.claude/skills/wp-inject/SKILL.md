---
name: wp-inject
description: 注入 WordPress 故障用于排障训练。当教练说"排障训练"、"注入故障"、"模拟故障"、"练习排障"时触发。
---

# wp-inject

## 列出可用故障

```bash
.claude/scripts/wp-train inject
```

返回 JSON，包含所有故障类型、描述和修复提示。

## 注入故障

```bash
.claude/scripts/wp-train inject <type>
```

会自动保存 `pre-fault` 检查点，注入后返回故障描述和修复提示。

向教练展示：
- 故障症状（不要告诉具体原因）
- 提示"现在去诊断并修复"
- 提醒"修复后说「完成了」让我检查，或说「放弃」还原"

## 还原

教练放弃时：

```bash
.claude/scripts/wp-train checkpoint restore pre-fault
```

还原后告诉教练正确的排查思路和修复方法。
