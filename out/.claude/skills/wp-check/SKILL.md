---
name: wp-check
description: 验证教练的 WordPress 操作结果，给出反馈并更新进度。当教练说"检查"、"完成了"、"做好了"时触发。
---

# wp-check

运行 `.claude/scripts/wp-train verify` 验证当前任务。

返回 JSON，包含 status（passed/failed）、results（每项断言的 pass/fail）、mastery、hints。

## 反馈规则

**passed**: 告诉教练 ✅ 完成，转述 on_pass_note（延伸知识），提示 /wp-train 获取下一题。

**failed**: 告诉教练 ❌ 或 ⚠️，根据 results 中的 actual vs expected 指出问题，转述 hints 给出排查方向。不要直接告诉教练答案。如果同一任务已失败 3 次以上，可以给出更直接的引导。
