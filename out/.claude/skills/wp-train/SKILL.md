---
name: wp-train
description: WordPress 训练出题。根据课纲和教练掌握度动态选题。当教练说"开始训练"、"出题"、"下一题"、"继续"时触发。
---

# wp-train

运行 `.claude/scripts/wp-train next` 获取下一题。

返回 JSON，包含 topic、difficulty、description、hints。

向教练展示格式：

```
─── 任务 [topic] 难度 ★☆☆ ───

{description}

完成后输入 /wp-check 检查结果。
```

难度星级：1=★☆☆，2=★★☆，3=★★★

如果返回 status=existing，说明有未完成的任务，提醒教练先完成或用 `.claude/scripts/wp-train next --force` 跳过。

如果返回 status=complete，恭喜教练全部通关。
