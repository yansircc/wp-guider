---
name: wp-build
description: 构建 wp-guider 产物到 out/ 目录。编译 Go 二进制、校验完整性。当说"build"、"构建"、"编译"、"打包"时触发。
---

# wp-build

运行 `bash scripts/build.sh`。

脚本会自动：编译二进制 → 校验 8 个产物文件 → 冒烟验证 status → 输出摘要。
