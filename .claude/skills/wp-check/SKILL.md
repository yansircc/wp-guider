---
name: wp-check
description: 验证教练的 WordPress 操作结果，给出反馈并更新进度。当教练说"检查"、"完成了"、"做好了"时触发。
---

# wp-check

验证当前任务，给反馈，更新进度。

## 流程

1. 读当前任务: `cat ~/.locwp/sites/wp-train/training/current-task.json`（不存在则提示先 /wp-train）
2. 执行验证: 运行任务中的 verify 命令
3. 判定结果（通过 / 部分通过 / 未通过）
4. 更新进度: 运行 `scripts/update-progress.sh`
5. 给出反馈

## 快照辅助

验证前先跑 `scripts/snapshot.sh` 获取站点全貌，有助于判断教练做了什么:

```bash
bash skills/wp-check/scripts/snapshot.sh
```

## 反馈格式

### 通过
```
✅ 任务完成！

<简短评价 + 一个延伸知识点>

输入 /wp-train 获取下一题。
```

### 部分通过
```
⚠️ 接近了，但有个细节：

<指出问题，不直接给答案>
<排查方向提示>

修正后再次 /wp-check。
```

### 未通过
```
❌ 还没完成。

<当前状态 vs 预期的差距>
<一个具体的下一步提示>

完成后 /wp-check。
```

## 反馈原则

- 对了: 鼓励 + 补充一个教练不知道的相关知识
- 错了: 不给答案，给排查方向。第 3 次未通过才揭示答案
- 部分通过算未通过（consecutive_passes 不增加）
