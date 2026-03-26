---
name: wp-train
description: WordPress 训练出题。根据课纲和教练掌握度动态生成实操任务。当教练说"开始训练"、"出题"、"下一题"、"继续"时触发。
---

# wp-train

根据课纲和教练掌握度动态出题。

## 流程

1. 读进度: `cat ~/.locwp/sites/wp-train/training/progress.json`（不存在则为新教练）
2. 读课纲: 读 `skills/wp-train/references/curriculum.md`
3. 选知识点（优先级: 薄弱项 > 未开始最低层 > 已掌握的进阶变体）
4. 生成任务并写入 current-task.json
5. 向教练展示任务

## 任务文件

写入 `~/.locwp/sites/wp-train/training/current-task.json`:

```json
{
  "topic": "L1.1.1",
  "difficulty": 1,
  "description": "任务描述...",
  "hint": "第一次提示",
  "verify": ["locwp wp wp-train -- ..."],
  "expected": "判定标准...",
  "issued_at": "ISO timestamp"
}
```

教练看到的格式:

```
─── 任务 [L1.1.1] 难度 ★☆☆ ───

<任务描述>

完成后输入 /wp-check 检查结果。
```

## 出题规则

- 一次一题，明确可验证
- 难度 1 = 操作题，难度 2 = 理解原理，难度 3 = 排障/综合
- 同一知识点连续通过 2 次标记掌握
- 贴近实战场景，不出背诵题
- verify 命令和 expected 对教练不可见
