# Figma 桌面插件集

与 Figma 设计稿（fileKey 见 use-figma.sh，不入库）配套的本地开发插件。均为 Figma 桌面端
「Development」插件：Plugins → Development → Import plugin from manifest… 选择对应
`manifest.json` 导入后即可运行。

## save-version/ — 打版本历史

`saveVersionHistoryAsync` 在云端 MCP 沙盒未实现（官方
figma/mcp-server-guide `skills/figma-use/references/api-reference.md`「Not implemented」清单），
REST `/v1/files/:key/versions` 只读——本插件是唯一打版通道。

- 用法：改 `code.js` 顶部 `VERSION_TITLE` / `VERSION_DESCRIPTION` 两处 →
  Plugins → Development → 运行「HotSpot Save Design Version」→ 右上角通知确认写入。
- 打版纪律见 `.agents/skills/hotspot-figma-automation/SKILL.md`（每次设计改动后必须打版）。

## bridge/ — 通用 Plugin API 执行通道（自动）

桌面插件内常驻 WebSocket 客户端 + 本地 bun 服务端，把云端沙盒执行不了的 Plugin API
代码转发到桌面运行时执行（桌面侧字体、`saveVersionHistoryAsync` 等完整可用）。
与旧版 Gemini CLI 扩展的桌面桥同原理。

```bash
# 1) 启动本地服务端（常驻）
bun scripts/design-kit/plugins/bridge/server.mjs        # 监听 http://127.0.0.1:8787

# 2) Figma 桌面端运行「HotSpot Figma Bridge」插件，保持小窗口开启（断线自动重连）

# 3) 远程执行任意 Plugin API 代码（支持 await/return，同云端 use_figma 写法）
curl -s -X POST http://127.0.0.1:8787/eval \
  -H 'Content-Type: application/json' \
  -d '{"code":"await figma.saveVersionHistoryAsync(\"标题\", \"描述\"); return {ok:true}"}'

# 查连接态
curl -s http://127.0.0.1:8787/status
```

- 首次使用需在 Figma 桌面端导入并运行一次插件；`server.mjs` 写 `server.pid` 供停止用。
- 安全边界：服务端只监听 127.0.0.1，本机进程可经 /eval 在当前打开的 Figma 文件里执行
  Plugin API——不要把 8787 暴露到局域网。
