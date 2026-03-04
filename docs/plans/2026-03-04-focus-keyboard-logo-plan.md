# Focus & Logo Overhaul Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在 Web 与 HarmonyOS 中实现统一优雅的键盘失焦收起机制，并用新龙形白底 Logo 更新所有图标资源。

**Architecture:** Web 端新增 `KeyboardFocusManager` + Svelte action 管理输入焦点，触控失焦时自动 dispatch `keyboard:dismiss`；HarmonyOS 端实现 `KeyboardDismissManager` 单例监听 UIContext 焦点并请求隐藏控件收起软键盘；Logo 通过脚本批量生成 PNG/SVG，多平台静态资源/manifest/icon 同步更新。

**Tech Stack:** TypeScript (SvelteKit), Node.js (sharp/png-to-ico), ArkTS (HarmonyOS), Image assets

---

### Task 1: Web KeyboardFocusManager 基础设施

**Files:**
- Create: `frontend_svelte/src/lib/utils/keyboard-focus.ts`
- Modify: `frontend_svelte/src/hooks.client.ts`, `frontend_svelte/src/lib/attachments/index.ts`
- Test: `pnpm check`

**Step 1:** 在 `src/lib/utils/keyboard-focus.ts` 编写 manager：导出 `useKeyboardFocus` action、`initKeyboardFocusManager()`、`KeyboardDismissEvent` 常量。

**Step 2:** 在 `src/hooks.client.ts` 注册一次 `initKeyboardFocusManager(window)`。

**Step 3:** 更新 `src/lib/attachments/index.ts` 导出新的 action，便于组件引入。

**Step 4:** 运行 `pnpm check` 确保类型通过。

**Step 5:** `git add` 新文件 + 修改文件并单独提交 `feat: add web keyboard focus manager`。

### Task 2: Web 组件接入 & 行为验证

**Files:**
- Modify: `frontend_svelte/src/lib/components/ui/input/input.svelte`, `textarea.svelte`, `components/common/chat composer files`, `routes/*` 中直接使用 `<textarea>` 的页面（Chat users/groups、search modal 等）
- Test: `pnpm check`, manual mobile devtools test

**Step 1:** 在 Input/textarea 组件中引入 `useKeyboardFocus`，确保所有 TextInput 自动注册。

**Step 2:** 对未复用组件（例如 chat composer、location-picker 搜索框）直接添加 action。

**Step 3:** 在聊天页面监听 `window` 的 `keyboard:dismiss` 事件以停止 typing 提示。

**Step 4:** `pnpm check`、`pnpm test`（若存在）、`pnpm build`（可选）并在移动端预览器手动验证失焦即收起。

**Step 5:** `git add` 相关文件并提交 `feat: enforce keyboard dismissal on blur (web)`。

### Task 3: Harmony KeyboardDismissManager 基础

**Files:**
- Create: `harmonyapp/entry/src/main/ets/core/utils/KeyboardDismissManager.ets`
- Modify: `harmonyapp/entry/src/main/ets/App.ets`（或根组件）、`core/utils/index`（若需导出）
- Test: `codelinter`

**Step 1:** 实现单例，保存 `registeredInputIds`、`hiddenTargetId`、`init(uiContext)`、`register`、`unregister`、`dismiss()`。

**Step 2:** 在 `App.ets`（或入口 Nav）中：
  - 渲染隐藏 `Button`（id `GlobalKeyboardDismissTarget`）。
  - 在 `aboutToAppear` 或初始化中调用 `KeyboardDismissManager.getInstance().init(this.getUIContext())`。

**Step 3:** `codelinter` 验证新增文件无警告。

**Step 4:** `git add` 新文件 + App 修改并提交 `feat: add harmony keyboard dismiss manager`。

### Task 4: Harmony 输入统一接入

**Files:**
- Modify: `harmonyapp/entry/src/main/ets/features/chat/pages/ChatPage.ets`, `components/LoginPromptView.ets`, `features/profile/pages/ProfilePage.ets`, 其他含 TextInput 的关键页面（Search, LocationPicker, Group forms, Settings 等）
- Test: `codelinter`, manual device test

**Step 1:** 为每个 TextInput 指定 `id`（如 `chatComposerInput`），在 `aboutToAppear` 或 `build` 内 `KeyboardDismissManager.register(id)`。

**Step 2:** 清理旧的隐形 Button/手写 `requestFocus` 逻辑，改为调用 `KeyboardDismissManager.dismiss()`。

**Step 3:** Sheets/模态 `onTouch` 空白处调用 `dismiss()`。

**Step 4:** `codelinter` + 真机/模拟器验证：点击空白、返回键、Tab 切换都能收起。

**Step 5:** `git add` 所有 ArkTS 变更并提交 `feat: unify keyboard dismissal across inputs`。

### Task 5: Logo 生成脚本 & Web 资源更新

**Files:**
- Create: `scripts/gen-logo-assets.ts`（Node + sharp）
- Modify: `package.json` scripts（可加 `pnpm logo:build`）、`frontend_svelte/static/*`（favicon/manifest icons）、`frontend_svelte/static/manifest.json`
- Test: run script + `pnpm check` + Lighthouse icon audit

**Step 1:** 编写脚本读取源图（放在 `assets/logo/logo-master.png`），输出所需尺寸到 `frontend_svelte/static/`。

**Step 2:** 运行脚本生成 PNG/SVG/ICO；核对 `manifest.json` 中的 `icons` 数组。

**Step 3:** 更新 `README` 如需说明新脚本。

**Step 4:** `pnpm check` & `pnpm build`，使用 Chrome Lighthouse 验证 PWA icons。

**Step 5:** `git add` 脚本 + 静态文件 + manifest，提交 `feat: refresh web branding assets`。

### Task 6: Harmony 图标资源更新

**Files:**
- Replace: `harmonyapp/AppScope/resources/base/media/{foreground.png,background.png,app_icon.png}`, `layered_image.json`（如需），`harmonyapp/entry/src/main/resources/base/media/startIcon.png` 等
- Test: 资源检查 + `codelinter`

**Step 1:** 运行脚本或手动导出 Harmony 所需分辨率（192×192）。

**Step 2:** 替换 layered icon 前景/背景、startIcon。

**Step 3:** 在 README 或变更记录说明来源。

**Step 4:** `codelinter` + Harmony 模拟器 (Launcher + 启动页) 验证图标显示正确。

**Step 5:** `git add` 资源文件并提交 `chore: update harmony icons`。

### Task 7: 回归验证 & 总结

**Files:**
- Commands only

**Step 1:** Web：`pnpm check && pnpm build`，手动在移动 Chrome DevTools 验证聊天/表单行为。

**Step 2:** Harmony：`codelinter`，提醒用户手动运行 hvigorw 组装，记录结果。

**Step 3:** `rg '"/api/v1/' frontend_svelte/src/ harmonyapp/entry/src/main/ets/`，确认无硬编码。

**Step 4:** 更新 `CHANGELOG` 或 `docs`（如需要）。

**Step 5:** `git status` → `git add -A` → 准备提交（或分多次提交已在前面完成）。

---

**Execution Options:**
1. **Subagent-Driven（推荐）**：当前会话内按任务顺序执行，每个任务启用 `executing-plans` 子 Agent 并在步骤间复核。
2. **Parallel Session**：开新会话，切换工作树执行；本会话用于 code review。

请告知使用哪种方式。EOF