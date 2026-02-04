# Frontend Development Guidelines

## Tech Stack & Utilities

- **Framework**: Nuxt v4
- **UI Library**: Nuxt UI v4
- **State Management**: Pinia
- **Utility Library**: VueUse (Preferred)

## Best Practices

### VueUse Usage (Preferred)

When implementing features that interact with browser APIs or require common stateful logic, **always prioritize `VueUse` composables** over native browser APIs or custom implementations.

#### Why?
- **Reactivity**: VueUse composables are designed to be reactive and integrate seamlessly with Vue's reactivity system (`ref`, `computed`, `watch`).
- **SSR Safety**: Many VueUse functions handle SSR compatibility out of the box (e.g., checking `window` existence).
- **Cleanup**: Automatic cleanup of event listeners and timers on component unmount.

#### Examples

| Feature | Native API / Custom | VueUse Equivalent |
| :--- | :--- | :--- |
| **Clipboard** | `navigator.clipboard.writeText` | `useClipboard()` |
| **Share** | `navigator.share` | `useShare()` |
| **Intervals** | `setInterval` / `clearInterval` | `useIntervalFn()` |
| **Timeouts** | `setTimeout` / `clearTimeout` | `useTimeoutFn()` |
| **Visibility** | `document.addEventListener('visibilitychange')` | `useDocumentVisibility()` |
| **Window Size** | `window.innerWidth` | `useWindowSize()` |
| **Dark Mode** | `window.matchMedia` | `useDark()` / `useColorMode()` |
| **Local Storage** | `localStorage.getItem` | `useStorage()` |
| **Intersection** | `IntersectionObserver` | `useIntersectionObserver()` |

#### Usage Example

```typescript
// ❌ Avoid: Native implementation
const copyText = (text) => {
  navigator.clipboard.writeText(text).then(() => {
    alert('Copied!')
  })
}

// ✅ Recommended: VueUse
import { useClipboard } from '@vueuse/core'

const { copy, copied, isSupported } = useClipboard()

const copyText = (text) => {
  if (isSupported.value) {
    copy(text)
    // 'copied' ref automatically updates for a short duration
  }
}
```
