---
name: wp-train
description: WordPress 训练出题。根据课纲和教练掌握度动态选题。当教练说"开始训练"、"出题"、"下一题"、"继续"时触发。
---

# wp-train

运行 `.claude/scripts/wp-train next` 获取下一题。

返回 JSON，包含 topic、difficulty、description、hints、context。

向教练展示格式：

```
─── 任务 [topic_name] 难度 ★☆☆ ───

{description}

请在浏览器中打开 wp-admin 后台完成操作。
完成后输入 /wp-check 检查结果。
```

如果 JSON 中包含 `chain` 字段，这是一个项目任务链，展示为：

```
─── 「{chain}」项目 [{chain_step}/{chain_total}] ───

{description}

完成后输入 /wp-check 检查结果。
```

难度星级：1=★☆☆，2=★★☆，3=★★★

利用 `context` 中的信息提供上下文：
- `context.task_attempts > 0` 时提醒"这道题你之前尝试过 N 次"
- `context.weak_topics` 非空时可提及薄弱项

如果返回 status=existing，说明有未完成的任务，提醒教练先完成或用 `.claude/scripts/wp-train next --force` 跳过。

如果返回 status=complete，恭喜教练全部通关。
