# feature / integration

## 何时读取
仅在以下场景读取并执行本文件：

- 改动涉及数据库写入
- 改动涉及数据库迁移或 Schema 变化
- 改动涉及跨服务联调
- 需要真实环境验证而非仅本地静态/单元校验

## 执行要求
1. 运行 `./deploy/scripts/verify.sh --env-name <name>`。
2. 若需要进一步检查环境状态，可使用 `./deploy/scripts/verify-inspect.sh --env-name <name>`。
3. 只有在隔离验证通过后，才允许继续整理提交和合并回主目录。
4. 若隔离验证持续失败且无法自主修复，按 `../stop-conditions.md` 处理。
