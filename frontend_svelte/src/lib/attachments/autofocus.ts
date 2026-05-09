/**
 * Svelte 5 Attachment: 自动聚焦到元素
 *
 * 使用 Attachment Factory 模式，当 trigger 变化时自动重新聚焦
 *
 * @example
 * ```svelte
 * <script>
 *   let focusTrigger = $state(0);
 *
 *   function sendMessage() {
 *     // 发送消息...
 *     focusTrigger++; // 触发重新聚焦
 *   }
 * </script>
 *
 * <textarea {@attach autofocus(focusTrigger)}></textarea>
 * ```
 *
 * @param _trigger - 任何值，当此值变化时会触发重新聚焦
 * @returns Attachment 函数
 */
export function autofocus(_trigger: unknown) {
  return (element: HTMLElement) => {
    // 使用 requestAnimationFrame 确保在下一帧聚焦
    // 避免被其他操作（如按钮点击）打断
    requestAnimationFrame(() => {
      element.focus();
    });
  };
}
