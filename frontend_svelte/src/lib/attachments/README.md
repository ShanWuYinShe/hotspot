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

### 防止按钮抢夺焦点

在按钮上添加 `onmousedown` 事件并调用 `preventDefault()`：

```svelte
<button onclick={handleClick} onmousedown={(e) => e.preventDefault()}> 点击我 </button>
```

**原理**：`onmousedown` 在 `onclick` 之前触发。通过 `preventDefault()` 阻止默认行为，按钮就不会获得焦点。

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

3. **防止按钮抢夺焦点**：使用 `onmousedown` + `preventDefault()`
   ```svelte
   <button onmousedown={(e) => e.preventDefault()}>
   ```

## 参考资料

- [Svelte 5 Attachments 官方文档](https://svelte.dev/docs/svelte/@attach)
- [Svelte 5 Actions 文档](https://svelte.dev/docs/svelte/use)
- [Svelte 5 迁移指南](https://svelte.dev/docs/svelte/v5-migration-guide)
