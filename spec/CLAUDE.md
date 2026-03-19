# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 这个目录是什么
`spec/` 是 HotSpot 的 API 契约和行为规范的权威来源。实现代码必须以这里的规范为准；除非用户明确要求，不要修改这里的文件。

## 读取顺序
- 查接口先看 `api_spec_index.md`，再看对应模块 spec。
- 不确定通用规则时先看 `api_spec_base.md`、`api_spec_types.md`、`api_spec_errors.md`。
- 如果多个 spec 冲突，按 `api_spec_base > api_spec_types > api_spec_errors > 各模块 spec` 的顺序判断。

## 引用方式
- 引用规范时使用文件名和小节标题。
- 不要用数字章节号。

## 常用操作
- 这里没有构建或测试命令；这里只做规范查阅。
