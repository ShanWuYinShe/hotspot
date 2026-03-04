# Focus & Logo Overhaul Design

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:writing-plans before implementation.

**Goal:**
- 统一 Web + HarmonyOS 的键盘行为：只要当前焦点不在任何输入组件，立即收起虚拟键盘，且实现优雅、可维护。
- 将新龙形 Logo（白底位图）替换到 Web 前端与 HarmonyOS 全部图标资源（favicon、PWA、启动图、layered icon 等），保持白底视觉。

**Architecture Summary:**
- Web：新增 `KeyboardFocusManager`（Svelte store + action）集中处理输入注册、指针事件监听、触摸设备检测与键盘收起广播；组件使用装饰器接入。
- Harmony：实现 `KeyboardDismissManager` 单例，监听 `UIContext` 焦点变化，在失焦时请求隐藏焦点以关闭键盘；提供 helper 供全部 TextInput 注册。
- Logo：使用源图生成高分辨 PNG，再导出所需尺寸；Web 静态资源/manifest 更新；Harmony layered icon、startIcon 与任何引用更新。

**Tech Stack:**
- SvelteKit (TypeScript, runes, actions)
- HarmonyOS ArkTS（UIContext, FocusController）
- Node scripts / ImageMagick (via `sharp` or `sips`) for资产生成

---

## Keyboard Behavior Design

### Web Frontend
1. **KeyboardFocusManager module** (`src/lib/utils/keyboard-focus.ts`)
   - Maintains `registeredInputs: WeakSet<HTMLElement>` + `currentFocus: HTMLElement | null`.
   - Provides `createKeyboardFocusAction()` returning Svelte action `useKeyboardFocus`。
   - On action init: add element to set; bind `focusin`, `focusout` events to update state。
   - Global listeners (mounted once via `hooks.client` or root layout):
     - `window.addEventListener('pointerdown'/'pointerup','touchstart','focusout')`.
     - For pointer events, if `event.pointerType === 'mouse'` and `!matchMedia('(pointer:coarse)').matches`，忽略，避免桌面副作用。
     - 在 `focusout` + `requestAnimationFrame` 后检查 `document.activeElement` 是否 STILL in set；若否，则 `document.activeElement?.blur()` 并 `window.dispatchEvent(new CustomEvent('keyboard:dismiss'))`。
   - Expose helper `isTextInput(node: Element)` to cover `input[type=text|email|password|search|tel|number], textarea, contenteditable`.
2. **Component Integration**
   - TextInput/textarea wrappers（`$lib/components/ui/input`, `textarea`, chat composer等）添加 `{@attach useKeyboardFocus()}`。
   - 页面级特殊组件（聊天输入、搜索条等）统一引入 action。
   - 监听 `keyboard:dismiss` 事件（如 chat composer）以触发滚动/停止 typing。
3. **Edge Handling**
   - Modal/Dialog 里 focus trap 依旧工作：action 只在 `focusout` 之后 rAF 检测 activeElement。若 focus 转移到另一个输入，manager detect set membership => 不收起。
   - Programmatic focus（例如 autofocus action）在 register 后立即 focus，rAF 逻辑不会错误触发。

### HarmonyOS App
1. **KeyboardDismissManager** (`entry/src/main/ets/core/utils/KeyboardDismissManager.ets`)
   - Singleton storing `registeredInputIds: Set<string>`，`hiddenFocusId: string`。
   - On `init(uiContext: UIContext)`: register global focus listener `uiContext.on('focus', handler)`。
   - Handler：
     - `const id = focusController.getFocusedElementId();`
     - If `!registeredInputIds.has(id)`, call `focusController.requestFocus(hiddenFocusId)` to dismiss keyboard。
   - Expose `register(componentId: string)` / `unregister(componentId: string)` / `pause()` (optional to skip during special flows)。
2. **Hidden Target**
   - 在根组件（如 `App.ets`）渲染一个不可见 `Button` with `id='GlobalKeyboardDismissTarget'`，注册到 manager，用于 `requestFocus`.
3. **Input Mixins**
   - Create helper `withKeyboardManagedInput(TextInputBuilder)` or decorator `@KeyboardManagedInput('id')` 以减少重复。
   - Existing TextInputs（ChatPage、LoginPromptView、Profile、Search、LocationPicker 等）引入 helper，`onFocus`/`onBlur` 只负责 UI，自身不再保存隐藏 Button。
4. **Event Hooks**
   - ChatPage：`dismissKeyboardOnOutsideTap()` 调用 manager `dismiss()` 而非定位按钮。
   - Sheets/Modal：在 `onTouch` blank area 统一 `KeyboardDismissManager.dismiss()`.

## Logo Replacement Design

### Asset Generation Workflow
1. **Source Processing**
   - 将提供的 768×768 白底图作为 master；用 `sharp`（Node 脚本 `scripts/gen-logo-assets.ts`）生成：
     - Base 1024×1024 PNG（白底 + 龙）
     - Transparent cut (alpha) 版本供 layered foreground 使用（通过抠出白底 stylized circle + dragon）。
2. **Web Outputs (`frontend_svelte/static/`)**
   - `favicon-16.png`, `favicon-32.png`, `favicon-48.png`, `favicon-64.png`, `favicon-96.png`, `favicon-128.png`。
   - `logo_16/32/64/180/192/256/512.png`（现有文件直接覆盖）。
   - `android-chrome-192x192.png`, `android-chrome-512x512.png`, `apple-touch-icon.png`（保持白底）。
   - `favicon.svg`: `<svg width=512>` 内嵌 `<image href="data:image/png;base64,..."/>`；`favicon.ico` 使用多尺寸打包（`png-to-ico` 或 `icotool`）。
   - 更新 `src/app.html`/`static/manifest.json` 若路径变化。
3. **HarmonyOS Outputs**
   - Layered icon：
     - `AppScope/resources/base/media/background.png`: 108x108 (or 192) 纯白 + 轻阴影。
     - `foreground.png`: 108x108 PNG 包含带透明背景的龙+红圈（放大 80%）
     - `AppScope/resources/base/media/app_icon.png` 用于 fallback。
   - Start icon：`entry/src/main/resources/base/media/startIcon.png` 生成 192×192 白底版本。
   - 若存在 `entry/src/main/resources/base/media/icon.png` 等，也一并替换。
   - 调整 `hvigorfile.ts` 或配置无需修改，只保证资源路径一致。

### Accessibility & QA
- Web：更新 Storybook/Docs 中的品牌展示、OpenGraph 预览（`static/og.png` 如需）。
- Harmony：运行 `codelinter`、`ResourceManager` 校验 icon 引用；实机确认 Launcher 深/浅色背景下边缘不失真。

---

## Validation Plan
1. Web：`pnpm check`、`pnpm test`（如有）、`pnpm build`（验证 manifest/静态资源）；移动端（DevTools）手动测试点击空白处键盘收起。  
2. Harmony：`codelinter`，手动在模拟器/真机检查 Chat、LoginPrompt、Profile、Search 等输入场景，确认焦点离开即关闭键盘。  
3. Asset diff：截图 & 视觉对比；校验 manifest 中 icon path 通过 Lighthouse PWA 检测。

---

## Open Questions
- 是否需要同时保留透明底的备用图？如后续需要，可在脚本中附加参数生成。
- 若未来支持深色模式 icon，可追加 `foreground-dark.png`。

