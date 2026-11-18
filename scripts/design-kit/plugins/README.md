# Figma 桌面插件：HotSpot Figma Bridge

与 Figma 设计稿（fileKey 见 use-figma.sh，不入库）配套的本地开发插件，**只有一个**：
桌面插件内常驻 WebSocket 客户端 + 本地 bun 服务端，把云端 MCP 沙盒执行不了的 Plugin API
代码转发到桌面运行时执行（桌面侧字体、`saveVersionHistoryAsync` 等完整可用）。
与旧版 Gemini CLI 扩展的桌面桥同原理。

## 安装（一次性）

1. Figma 桌面端 → Plugins → Development → **Import plugin from manifest…** → 选本目录 `manifest.json`
2. 运行「HotSpot Figma Bridge」，保持画布上的绿色小窗口开启（关闭即断，重开自动重连）
3. 本地启动服务端（常驻）：`bun scripts/design-kit/plugins/bridge/server.mjs`（监听 127.0.0.1:8787）

## 用法

```bash
# 打版本历史（内置命令；每次设计改动后必须打，纪律见 .agents/skills/hotspot-figma-automation/SKILL.md）
curl -s -X POST http://127.0.0.1:8787/save-version -H 'Content-Type: application/json' \
  -d '{"title":"细节精修版 · 主题 (M月D日 HH:MM)","description":"【本次版本更新内容摘要】…"}'

# 执行任意 Plugin API 代码（支持 await/return，同云端 use_figma 写法；如迁移 PingFang → Noto）
curl -s -X POST http://127.0.0.1:8787/eval -H 'Content-Type: application/json' \
  -d '{"code":"return {name: figma.root.name}"}'

# 查连接态
curl -s http://127.0.0.1:8787/status
```

## 背景

云端 MCP 沙盒的 `saveVersionHistoryAsync` 未实现（官方
figma/mcp-server-guide `skills/figma-use/references/api-reference.md`「Not implemented」清单）、
REST `/v1/files/:key/versions` 只读——依赖完整 Plugin API 的操作只有桌面运行时能做，
本桥即为此通道。安全边界：服务端只监听 127.0.0.1，本机进程可经 /eval 在当前打开的
Figma 文件里执行 Plugin API——不要把 8787 暴露到局域网。修改 `code.js` 后在 Figma 里
重新运行一次插件即生效（开发插件每次运行从磁盘读取）。
