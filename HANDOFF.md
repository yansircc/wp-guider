# Handoff 文档 — WP Guider 知识库生成进度

## 本次会话完成了什么

### 1. 题库重构：Topic × Difficulty 矩阵
- **文件**: `topic-matrix.html`（可直接浏览器打开）
- 将原线性 L1-L8 题库重构为 **20 个 Topic × 5 级 Difficulty** 的二维矩阵
- 总计 **252 道题**（含新增 L0 基础设施题目）
- 所有格子默认展开显示题目列表

### 2. 题目数据
- **文件**: `docs-tasks.json`（252 条，从 topic-matrix.html 提取）
- 每条含: id, topic, difficulty, desc, isNew

### 3. 知识库文档生成（进行中）
- **目录**: `docs/` 下按 topic 分子目录
- **共用样式**: `docs/style.css`
- **已生成 135 / ~170 文件**（合并后预估总数）

## 缺失的 Topic（需要继续生成）

| Topic | 需要文件数 | 当前状态 |
|-------|-----------|---------|
| google-analytics | ~5 | **0 文件，完全缺失** |
| seo | ~8 | **0 文件，完全缺失** |
| menus-nav | ~3 | **0 文件，完全缺失** |
| plugins-basic | ~2 | **0 文件，完全缺失** |
| theme | ~6 | **0 文件，完全缺失** |
| troubleshooting | ~8 | **0 文件，完全缺失** |
| wp-install | ~5 | **5 文件，已完成** (ccs1) |
| elementor | ~18 | 13 文件，差 ~5 |

## 已完成的 Topic

| Topic | 文件数 | 来源 |
|-------|--------|------|
| site-settings | 10 | subagent |
| user-management | 8 | subagent |
| wp-config | 10 | subagent |
| security | 4 | subagent + maestri |
| pages | 8 | subagent |
| posts-taxonomy | 6 | subagent + maestri |
| acf | 22 | subagent + maestri |
| zeroy | 28 | subagent + maestri |
| domain | 6 | maestri (ccs) |
| hosting | 6 | maestri (ccs) |
| backup-maintenance | 9 | maestri (ccs4) |
| media | 4 | maestri (ccs2) |
| elementor | 13 | subagent + maestri (ccs6) |

## 如何继续

### 方式 1：用 Maestri 邻居补充
Maestri 里有 6 个邻居 agent：ccs, ccs2, ccs3, ccs4, ccs5, ccs6。
它们都已 idle，可以直接 `maestri ask` 分配任务。

缺失 topic 的分配建议：
- **ccs**: wp-install（只生成了 1 个，需要补 4-5 个）
- **ccs2**: menus-nav + plugins-basic（~5 文件）
- **ccs3**: seo + google-analytics（~13 文件）
- **ccs4**: theme + troubleshooting（~14 文件）
- **ccs6**: elementor 补充（~5 文件）

### 方式 2：直接 ask 示例
```bash
maestri ask "ccs3" "继续你之前的任务。检查 /Users/fx/yansircc/wp-guider/docs/seo/ 和 docs/google-analytics/ 目录，这两个目录目前是空的。请读取 docs-tasks.json 筛选 topic=seo 和 topic=google-analytics 的题目，生成所有 HTML 文档。格式同之前的文件。"
```

### 方式 3：单独生成
也可以在新会话里直接写文件，参考 `docs/style.css` 和任意已有文件的格式。

## 下一步（文档全部生成后）

1. **更新 topic-matrix.html**：让矩阵里的每道题链接到对应的文档页面
2. **生成 index.html**：知识库首页，带搜索和导航
3. **每个 topic 目录生成 index.html**：该 topic 的文档列表页
4. **交叉链接**：每篇文档底部的「相关文档」和「上一篇/下一篇」导航

## 关键设计决策记录

- **题库结构**: Flat 数组，topic + difficulty 标签（非线性 L1-L8）
- **Difficulty 5 级**: L1入门 / L2基础 / L3进阶 / L4高级 / L5专家
- **不同 topic 的难度范围不同**（如域名只有 L1-L2，故障排查只有 L3-L5）
- **L0 验证方式**: 程序化优先（dig/curl）+ 截图补充
- **不用 Gutenberg**: Post 用经典编辑器+MD 插件，Page 用 Page Builder
- **不用 Widget**: 直接用 Theme Builder
- **SEO 双插件**: Yoast + SEOPress 都要掌握
- **安全以 Wordfence 为主**
- **备份用 WP Reset 快照**（更简单），迁移用 All-in-One Migration
- **ACF 工具**: acf-playground.vercel.app + acf-helper.vercel.app
- **连贯题可合并**: 非常相关的题合成一份文档，开头注明覆盖范围

## 文件清单

```
wp-guider/
├── topic-matrix.html          # 矩阵首页（可浏览器打开）
├── docs-tasks.json            # 252 道题的 JSON 数据
├── docs/
│   ├── style.css              # 共用样式
│   ├── domain/                # 6 files
│   ├── hosting/               # 6 files
│   ├── wp-install/            # 1 file (需补)
│   ├── site-settings/         # 10 files ✓
│   ├── user-management/       # 8 files ✓
│   ├── pages/                 # 8 files ✓
│   ├── posts-taxonomy/        # 6 files ✓
│   ├── media/                 # 4 files ✓
│   ├── menus-nav/             # 0 files (需生成)
│   ├── plugins-basic/         # 0 files (需生成)
│   ├── seo/                   # 0 files (需生成)
│   ├── google-analytics/      # 0 files (需生成)
│   ├── theme/                 # 0 files (需生成)
│   ├── acf/                   # 22 files ✓
│   ├── wp-config/             # 10 files ✓
│   ├── security/              # 4 files ✓
│   ├── backup-maintenance/    # 9 files ✓
│   ├── troubleshooting/       # 0 files (需生成)
│   ├── elementor/             # 13 files (需补)
│   └── zeroy/                 # 28 files ✓
```
