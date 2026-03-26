# WP Guider — 开发指南

WordPress 培训系统。Go CLI + 声明式验证 + SQLite 状态 + Claude Code Skills。

## 目录结构

```
cmd/wp-train/       # Go 源码（CLI 入口）
scripts/            # 开发工具（audit.sh）
out/                # 消费者产物（构建输出）
  └── .claude/      # 压缩包内容，直接给教练用
.claude/            # 开发者配置（你正在看的）
  └── skills/
      └── wp-build/ # 构建 skill
```

## 构建

```bash
/wp-build
```

或手动：
```bash
go build -o out/.claude/scripts/wp-train ./cmd/wp-train/
```

## 关键依赖

- Go 1.23+
- `modernc.org/sqlite`（纯 Go SQLite，零 CGO）
- `locwp` CLI（运行时依赖）

## 消费者产物

`out/.claude/` 是完整的教练训练环境，包含：
- `scripts/wp-train` — 编译后二进制
- `references/task-bank.json` — 题库
- `skills/` — 四个 Claude Code skills
- `CLAUDE.md` — Guider 人设

教练拿到 `out/` 后，放到任意目录，用 Claude Code 打开即可开始训练。

## 调试

```bash
# 查看 agent 决策链
./scripts/audit.sh

# 指定会话
./scripts/audit.sh <session-id>

# 限制条数
./scripts/audit.sh -n 20
```
