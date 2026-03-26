---
name: wp-site
description: 管理 WordPress 训练站点。创建、重置、删除训练环境。当教练说"创建站点"、"重置站点"、"删除站点"、"初始化"时触发。
---

# wp-site

管理训练用 WordPress 站点的生命周期。

## 操作

### 创建训练站

运行 `scripts/reset-site.sh`（创建全新站点）:
```bash
bash skills/wp-site/scripts/reset-site.sh
```

创建完成后告知教练:
- 后台: https://wp-train.loc.wp/wp-admin/
- 账号: admin / admin

### 重置站点

同样运行 `scripts/reset-site.sh`，它会自动检测并删除旧站点再重建。

重置前确认教练意图。进度文件（progress.json）不会自动清除，除非教练明确要求。

### 删除站点

```bash
locwp delete wp-train
```

### 预置内容

某些训练模块需要预置内容。运行 `scripts/seed-content.sh <layer>`:
```bash
bash skills/wp-site/scripts/seed-content.sh 3
```

Layer 1-2 用干净站点，Layer 3+ 需要预置页面/文章/菜单等。

### 查看站点状态

```bash
locwp list
```
