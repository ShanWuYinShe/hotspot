# Svelte 5 Attachments

这个目录包含符合 **Svelte 5+ 最佳实践**的 attachment 实现。

## 什么是 Attachments？

Attachments 是 Svelte 5.29+ 引入的新特性，用于替代传统的 Actions (`use:` directive)。它们具有以下优势：

- ✅ **完全响应式**：参数变化时自动重新运行
- ✅ **更简洁**：无需手动管理 `update` 和 `destroy` 生命周期
- ✅ **更灵活**：支持内联定义和组件传递
- ✅ **更可组合**：一个元素可以有多个 attachments

## 使用示例

### autofocus

自动聚焦到输入框，当 trigger 变化时重新聚焦。

```svelte
<script>
  import { autofocus } from '$lib/attachments';

  let focusTrigger = $state(0);

  function sendMessage() {
    // 发送消息逻辑...
    focusTrigger++; // 触发重新聚焦
  }
</script>

<!-- 使用 {@attach ...} 语法 -->
<textarea {@attach autofocus(focusTrigger)} bind:value={message}></textarea>

<button onclick={sendMessage} onmousedown={(e) => e.preventDefault()}> 发送 </button>
```

### 防止按钮抢夺焦点（条件化版本，推荐）

无脑 `onmousedown={(e) => e.preventDefault()}` 会同时屏蔽键盘 Tab 的焦点环 ——
键盘用户聚焦到按钮后看不到焦点。仅在 textarea 当前正被聚焦时才阻止默认行为：

```svelte
<button
  onclick={handleClick}
  onmousedown={(e) => {
    if (document.activeElement instanceof HTMLTextAreaElement) e.preventDefault();
  }}
>
  点击我
</button>
```

**原理**：
- `onmousedown` 在 `onclick` 之前触发；`preventDefault()` 阻止按钮获得焦点。
- 鼠标在 textarea 中输入时点击按钮 → 不抢焦点（保留输入连续性）。
- 键盘 Tab 聚焦按钮 → 不触发 preventDefault → 焦点环正常显示（满足 a11y）。

> 旧版 `onmousedown={(e) => e.preventDefault()}`（无条件 preventDefault）已被淘汰，
> 不要在新代码中 reintroduce。

## 与 Actions 的区别

| 特性             | Actions (`use:`)     | Attachments (`{@attach}`)     |
| ---------------- | -------------------- | ----------------------------- |
| 响应式           | 需手动实现 `update`  | 自动响应参数变化              |
| 生命周期         | `update` + `destroy` | 自动管理                      |
| 语法             | `use:action={param}` | `{@attach attachment(param)}` |
| 一个元素多个实例 | ❌                   | ✅                            |
| 传递给组件       | ❌                   | ✅                            |
| Svelte 版本      | < 5.29               | >= 5.29                       |

## 最佳实践

1. **Attachment Factory 模式**：返回一个接收元素的函数

   ```typescript
   export function myAttachment(param: string) {
     return (element: HTMLElement) => {
       // 设置逻辑
       return () => {
         // 清理逻辑（可选）
       };
     };
   }
   ```

2. **使用 requestAnimationFrame**：确保焦点操作不被打断

   ```typescript
   requestAnimationFrame(() => {
     element.focus();
   });
   ```

3. **防止按钮抢夺焦点**：使用条件化的 `onmousedown` + `preventDefault()`
   （见上文「条件化版本」）—— 不要无条件 `preventDefault()`，会破坏键盘焦点环。

## 参考资料

- [Svelte 5 Attachments 官方文档](https://svelte.dev/docs/svelte/@attach)
- [Svelte 5 Actions 文档](https://svelte.dev/docs/svelte/use)
- [Svelte 5 迁移指南](https://svelte.dev/docs/svelte/v5-migration-guide)
