# WordPress 教练训练系统

一个基于 AI 的 WordPress 实操训练工具。你将和 AI 导师一起，通过动手做任务的方式，从零开始掌握 WordPress。

## 快速开始

### 1. 前置条件

- **macOS**（目前只支持 macOS）
- **[locwp](https://github.com/yansircc/locwp)**：本地 WordPress 环境管理工具
- **[Claude Code](https://claude.ai/code)**：AI 编程助手

确保 locwp 已安装并完成 setup：

```bash
locwp setup
```

### 2. 开始训练

把这个目录放到任意位置，用 Claude Code 打开：

```bash
cd wp-guider    # 或者你解压的目录名
claude .
```

进入 Claude Code 后，输入：

```
/wp-site
```

AI 会自动创建一个训练用的 WordPress 站点。

### 3. 开始做题

```
/wp-train
```

AI 会给你出一道题。去 WordPress 后台完成操作后，输入：

```
/wp-check
```

AI 会检查你的操作是否正确，给出反馈和延伸知识。

### 4. 查看进度

```
/wp-progress
```

### 5. 排障训练（进阶）

```
/wp-inject
```

AI 会故意把你的 WordPress 弄坏，让你练习诊断和修复。

## 训练大纲

八层递进，从基础操作到深度排障：

| 层级 | 内容 |
|------|------|
| Layer 1 | 初识 WordPress — 登录、设置、用户 |
| Layer 2 | 内容管理 — 页面、文章、菜单、主题、插件 |
| Layer 3 | 文件系统 — wp-content、wp-config、请求生命周期、Hook |
| Layer 4 | 数据层 — 数据库表结构、WP_Query、Options API |
| Layer 5 | 主题引擎 — 模板层级、子主题、functions.php |
| Layer 6 | 插件与扩展 — 插件架构、CPT、REST API |
| Layer 7 | 服务器层 — 固定链接、wp-cron、安全 |
| Layer 8 | 排障 — 白屏诊断、插件冲突、性能瓶颈 |

## 可用命令

| 命令 | 说明 |
|------|------|
| `/wp-site` | 创建或重置训练站点 |
| `/wp-train` | 获取下一道训练题 |
| `/wp-check` | 检查你的操作结果 |
| `/wp-progress` | 查看训练进度 |

以上是你需要掌握的全部命令。排障训练、站点还原等高级操作由 AI 自动处理，你只需要用自然语言告诉 AI 你想做什么就行。

## 训练站点信息

| 项目 | 值 |
|------|---|
| 后台地址 | https://wp-train.loc.wp/wp-admin/ |
| 用户名 | admin |
| 密码 | admin |

## 常见问题

**Q: 站点打不开怎么办？**
运行 `/wp-site` 重新创建站点。

**Q: 做错了想重来？**
对 AI 说"帮我还原到上一个检查点"，或者运行 `/wp-site` 完全重置。

**Q: 题目太简单/太难？**
告诉 AI 你想跳到某个层级，比如"我想练习 Layer 3 的内容"。
