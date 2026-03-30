---
description: "查看知识库。启动 Web 服务浏览题库矩阵和文档。当教练说"知识库"、"查文档"、"打开矩阵"、"看题库"时触发。"
---

# 知识库 Web 服务

启动本地 HTTP 服务，让教练在浏览器中查看题库矩阵和知识文档。

## 操作步骤

1. **检查端口**：先检查 8080 是否已被占用
   ```bash
   lsof -i :8080 | grep LISTEN
   ```

2. **启动服务**（如果未运行）：
   ```bash
   python3 -m http.server 8080 &>/dev/null &
   ```

3. **获取 IP**（局域网共享）：
   ```bash
   ipconfig getifaddr en0 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}'
   ```

4. **输出链接**：
   ```
   知识库已启动：
   - 题库矩阵：http://localhost:8080/topic-matrix.html
   - 知识库首页：http://localhost:8080/docs/index.html
   - 局域网访问：http://{LAN_IP}:8080/docs/index.html
   ```

## 注意

- 服务从项目根目录启动（`topic-matrix.html` 和 `docs/` 在根目录）
- 用完后可手动关闭：`kill $(lsof -t -i:8080)` 或直接忽略，退出终端自动关闭
- 如果 8080 被占用，换 8081 或其他端口
