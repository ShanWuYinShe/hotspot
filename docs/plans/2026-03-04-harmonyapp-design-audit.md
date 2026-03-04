# Harmonyapp Design Audit Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 全量审查 harmonyapp ArkTS 客户端，列出违反 spec/harmonyapp_rule.md 与 HarmonyOS 设计规范的实现/结构问题

**Architecture:** 先梳理全局规范和 AGENTS 约束，再按 Config → API → 组件库 → 页面 → 资源 → 构建链路的顺序逐一审查。每个阶段使用 `rg` 快速排查反模式，再用 `read` 精读典型文件，将发现记录到临时笔记中。

**Tech Stack:** HarmonyOS ArkTS、OpenHarmony UI DSL、DesignConstants、rg、codelinter

---

### Task 1: 规范基线梳理

**Files:**
- Read: `harmonyapp/AGENTS.md`
- Read: `spec/harmonyapp_rule.md`
- Read: `spec/harmonyapp_doc.md`
- Read: `spec/api_spec_types.md`

**Step 1:** 读取 `harmonyapp/AGENTS.md`，整理强制的架构/组件规则。

**Step 2:** 依次阅读 `spec/harmonyapp_rule.md`、`spec/harmonyapp_doc.md`、`spec/api_spec_types.md`，记录 UI 布局、API/类型和幂等要求。

**Step 3:** 将关键信息汇总到 `tmp/ai-notes/harmonyapp-audit-notes.md`，为后续审查提供引用。

### Task 2: API 常量与 HttpClient 结构审查

**Files:**
- Read: `harmonyapp/entry/src/main/ets/core/config/ApiEndpoints.ets`
- Read: `harmonyapp/entry/src/main/ets/core/api/ApiClient.ets`
- Read: `harmonyapp/entry/src/main/ets/core/api/HttpUtil.ets`
- Scan: `harmonyapp/entry/src/main/ets/**/*.ets`

**Step 1:** 检查 `ApiEndpoints.ets` 是否覆盖 spec 中的端点且命名规范。

**Step 2:** 阅读 `ApiClient.ets`、`HttpUtil.ets`，验证拦截器、幂等路径、token 处理与 spec 一致。

**Step 3:** 运行 `rg '"/api/v1/' harmonyapp/entry/src/main/ets -n`，确认不存在硬编码路径；若发现，记入笔记并定位文件。

### Task 3: 设计常量与组件库审查

**Files:**
- Read: `harmonyapp/entry/src/main/ets/core/theme/DesignConstants.ets`
- Read: `harmonyapp/entry/src/main/ets/common/components/**/*.ets`

**Step 1:** 审核 `DesignConstants` 定义，确认半径、颜色、字号等集中管理。

**Step 2:** 抽样阅读公共组件目录，检查是否复用 `DesignConstants`、是否存在重复组件或未抽象的重复样式。

**Step 3:** 将重复/冗余组件、违背规范的布局记录到审查笔记。

### Task 4: 页面与状态管理审查

**Files:**
- Read: `harmonyapp/entry/src/main/ets/pages/**/*.ets`
- Read: `harmonyapp/entry/src/main/ets/viewmodel/**/*.ets`

**Step 1:** 针对核心页面（Home、Profile、Login、Map 等）阅读布局代码，确认遵循滚动/弹窗规范，不滥用绝对定位。

**Step 2:** 检查 `viewmodel` 层是否存在重复状态、未复用 service、或复杂逻辑放在 UI 层的情况。

**Step 3:** 将每个页面发现的问题写入笔记（含文件路径、行号、违规原因）。

### Task 5: 资源与图标审查

**Files:**
- Read/List: `harmonyapp/entry/src/main/resources/base/**/*.json` / `media/*`
- Read: `harmonyapp/entry/src/main/ets/common/icons/**/*.ets`

**Step 1:** 列出资源文件，确认没有未使用/重复的媒体或文案。

**Step 2:** 检查图标/插画组件是否复用统一封装，避免散落在页面代码中。

**Step 3:** 更新笔记，指出需精简或合并的资源。

### Task 6: 工具验证与结果输出

**Files:**
- Command: `cd harmonyapp && node "$CODELINTER" -c code-linter.json5 entry/src/main/ets/`
- Notes: `tmp/ai-notes/harmonyapp-audit-notes.md`

**Step 1:** 在 `harmonyapp` 目录运行 `codelinter`，确保 ArkTS 代码无 lint 警告；记录报告结果。

**Step 2:** 汇总所有问题，生成最终审核报告（路径：`tmp/ai-notes/harmonyapp-audit-findings.md`），按照严重度排序，关联 spec 证据。

**Step 3:** 若没有代码变更，仅将笔记与 `git status` 结果同步告知用户；若需要提出修复建议，附上下一步行动列表。
