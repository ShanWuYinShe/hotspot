# HotSpot API 规范目录 (spec)

## 这里是什么
`spec/` 是 HotSpot 的 **API 契约唯一权威**(HTTP 端点 / 字段 / 类型 / 错误码 / WebSocket 事件 / envelope)。只有契约,不含实现细节。

## 入口
- 查接口、导航、路径前缀速查 → `api_spec_index.md`
- 通用规则 → `api_spec_base.md` / `api_spec_types.md` / `api_spec_errors.md`

## 纪律(只读)
- 实现规范、构建命令、功能边界**不在本目录**——在各子项目根的 `AGENTS.md`(`backend-go/`、`frontend/`、`harmonyapp/`)。
- 本目录默认只读。发现实现与契约冲突时,默认改实现;只有用户明确要求时才改契约。
- 引用契约用文件名或小节标题,不用数字章节号。
- 内部冲突优先级:`api_spec_base.md` > `api_spec_types.md` > `api_spec_errors.md` > 各模块 `api_spec_*.md`。
