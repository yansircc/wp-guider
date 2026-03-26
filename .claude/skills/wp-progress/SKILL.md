---
name: wp-progress
description: 展示 WordPress 训练进度和掌握情况。当教练说"进度"、"状态"、"掌握了多少"时触发。
---

# wp-progress

展示训练进度。

## 流程

运行 `scripts/show-progress.sh` 获取格式化输出:

```bash
bash skills/wp-progress/scripts/show-progress.sh
```

读 `skills/wp-train/references/curriculum.md` 获取知识点总数，与进度对比。

## 建议逻辑

- 有薄弱项（attempts >= 3 且未掌握）→ 建议巩固
- 当前层全部掌握 → 恭喜，建议进入下一层
- 通过率 < 60% → 建议放慢节奏
- 全部掌握 → 毕业
