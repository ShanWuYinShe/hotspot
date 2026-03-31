# HotSpot Frontend

基于 SvelteKit + Svelte 5 的 Web 前端应用。

## 技术栈

- **框架**：SvelteKit + Svelte 5
- **样式**：Tailwind CSS
- **UI 组件**：bits-ui + lucide-svelte
- **通知**：svelte-sonner

## 验证命令

格式化 / 语法检查 / 编译通过受控脚本执行（必须在 feature worktree 中）：

```bash
python3 .claude/scripts/frontend_dev.py --task all
```

测试与最终验证通过 deploy/ 隔离执行：

```bash
./deploy/scripts/verify.sh --env-name <name>
```

## 规范与文档

- 完整协作规范见 [CLAUDE.md](CLAUDE.md)
- 本地优先工作流见根目录 [CLAUDE.md](../CLAUDE.md)
- API 规范见 [spec/](../spec/)
