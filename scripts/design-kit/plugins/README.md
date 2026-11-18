# 通用 Figma 桌面桥：Figma Local Bridge

通用的本地开发与 AI Agent 桥接插件：
桌面插件由 UI iframe（WebSocket 客户端）与主线程沙盒（Plugin API 执行器）组成，
配合本地服务端，把云端 MCP 沙盒执行不了的 Plugin API 代码转发到桌面运行时执行（桌面侧字体、`saveVersionHistoryAsync` 版本历史等完整可用）。

**一次安装，跨项目通用**：只需在 Figma 桌面端导入一次，以后在任何项目中启动 Bridge 服务，该插件均可自动直连。

## 安装（一次性）

1. Figma 桌面端 → Plugins → Development → **Import plugin from manifest…** → 选 `scripts/design-kit/plugins/bridge/manifest.json`
2. 运行「Figma Local Bridge」，保持画布上的状态小窗口开启（显示绿色已连接；关闭即断，重开自动重连）

## 一键自动化 CLI（推荐）

项目内置了全自动化控制脚本 `figma-bridge.mjs`，支持自动后台拉起服务、管理进程与收发指令：

```bash
# 1. 查看状态（服务是否启动、桌面插件是否连接、打开的 Figma 文件名）
node scripts/design-kit/figma-bridge.mjs status

# 2. 执行任意 Figma Plugin API 代码（若后台服务未启动，将自动在后台拉起！）
node scripts/design-kit/figma-bridge.mjs eval "figma.root.name"
node scripts/design-kit/figma-bridge.mjs eval -f path/to/script.js

# 3. 打版本历史里程碑（自动拉起服务）
node scripts/design-kit/figma-bridge.mjs save-version -t "版本标题" -d "更新内容摘要..."

# 4. 后台服务启停管理
node scripts/design-kit/figma-bridge.mjs start     # 后台常驻启动
node scripts/design-kit/figma-bridge.mjs stop      # 停止后台服务
node scripts/design-kit/figma-bridge.mjs restart   # 重启服务

# 5. 协议与通信自检（CI / 本地秒级验证）
node scripts/design-kit/figma-bridge.mjs verify
```

## 跨项目复用指南（Other Projects）

其他项目想要接入本套 Bridge 通道，只需：
1. **直接复用已安装的桌面插件**：桌面端已导入的「Figma Local Bridge」默认监听 `ws://127.0.0.1:8787`，无需在每个项目重复导入！
2. **复制服务与 CLI**：将 `scripts/design-kit/plugins/bridge/server.mjs` 和 `scripts/design-kit/figma-bridge.mjs` 复制到新项目中（或放入通用 npm 工具库）。
3. 新项目直接运行 `node figma-bridge.mjs eval "..."` 即可操控当前正在编辑的任何 Figma 设计稿。

## 背景

云端 MCP 沙盒的 `saveVersionHistoryAsync` 未实现（官方
figma/mcp-server-guide `skills/figma-use/references/api-reference.md`「Not implemented」清单）、
REST `/v1/files/:key/versions` 只读——依赖完整 Plugin API 的操作只有桌面运行时能做，
本桥即为此通道。安全边界：服务端只监听 127.0.0.1，本机进程可经 /eval 在当前打开的
Figma 文件里执行 Plugin API——不要把 8787 暴露到局域网。修改 `code.js` 后在 Figma 里
重新运行一次插件即生效（开发插件每次运行从磁盘读取）。
