# HotSpot Frontend

基于 SvelteKit + Svelte 5 的 Web 前端应用。

## 技术栈

- **框架**：SvelteKit + Svelte 5
- **样式**：Tailwind CSS
- **UI 组件**：bits-ui + lucide-svelte
- **通知**：svelte-sonner

## 验证与协作

- 仓库级 workflow、skill 入口和风险边界以根目录 [CLAUDE.md](../CLAUDE.md) 为准。
- 普通分析、代码阅读、局部低副作用开发动作通常可以直接完成。
- 页面/UI/交互的最终确认优先走隔离入口：

```bash
./deploy/scripts/verify.sh --env-name <name>
./deploy/scripts/verify-inspect.sh --env-name <name>
```

- 验证截图、调试输出和中间文件默认写到当前 worktree 的 `.local/`。

## 规范与文档

- 完整协作规范见 [CLAUDE.md](CLAUDE.md)
- 本地 workflow 见根目录 [CLAUDE.md](../CLAUDE.md)
- API 规范见 [spec/](../spec/)
