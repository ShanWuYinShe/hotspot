# feature / testing

## 何时读取
在实现完成后、进入本地校验阶段时读取本文件。

## 校验要求
先运行受影响子项目的本地校验：

- `backend/`：`cargo test`
- `backend-go/`：`go test ./...`
- `frontend_svelte/`：`pnpm check && pnpm build`
- `harmonyapp/`：运行当前环境可用的 Harmony 校验命令；若工具链缺失且无法完成必要校验，则视为阻塞

## 规则
- 补齐与本次改动直接相关的必要测试。
- 只要某个受影响子项目存在本地可执行校验，就不能跳过。
- 若测试失败，优先修复实现或测试本身，而不是绕过校验。
