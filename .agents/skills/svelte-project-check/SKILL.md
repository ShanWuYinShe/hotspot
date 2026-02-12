---
name: svelte-project-check
description: Check a SvelteKit project for type errors, compilation issues, and IDE-visible diagnostics using svelte-check. Use when verifying code quality after changes, before commits, or when the user asks to check the SvelteKit/frontend project for errors.
compatibility: Requires Node.js, pnpm, svelte-check.
metadata:
  category: quality-check
---

# SvelteKit 项目检查

`pnpm build` 通过 ≠ 没有问题。IDE 中可见的错误和警告来自 svelte-check，它执行比 Vite build 更严格的检查。

## svelte-check（IDE 诊断的 CLI 等价）

```bash
cd frontend_svelte && pnpm check
```

捕获范围：
- TypeScript strict 模式类型错误
- Svelte 组件语义错误（props 类型、事件、slot 不匹配）
- Svelte 5 runes 使用错误（$state、$derived、$props）
- 可访问性警告（a11y）

项目 tsconfig.json 中的额外严格规则（`pnpm build` 不一定报告，但 `pnpm check` 和 IDE 会报告）：

| 规则 | 效果 |
|------|------|
| `noUnusedLocals` | 未使用的局部变量报错 |
| `noUnusedParameters` | 未使用的函数参数报错 |
| `noUncheckedIndexedAccess` | 数组/对象索引访问必须处理 undefined |
| `noFallthroughCasesInSwitch` | switch 必须有 break/return |
| `noImplicitOverride` | 重写方法必须标记 override |
| `noPropertyAccessFromIndexSignature` | 索引签名属性必须用括号访问 |

## 构建验证

```bash
cd frontend_svelte && pnpm build
```

svelte-check 不覆盖的问题：SSR 兼容性（浏览器 API 在服务端使用）、动态导入路径错误、Vite 构建优化问题。

## 解读输出

```
/path/to/file.svelte:42:10
Error: Type 'string' is not assignable to type 'number' (ts)

/path/to/file.svelte:15:3
Warning: <img> element should have an alt attribute (a11y)
```

- **Error** — 必须修复，IDE 中红色波浪线
- **Warning** — 应当修复，IDE 中黄色波浪线

## 常见陷阱

- `pnpm build` 过了但 `pnpm check` 报错 → 正常，svelte-check 更严格
- `$state` 只能在 `.svelte` 或 `.svelte.ts` 文件中使用，普通 `.ts` 文件中会报错
- `noUncheckedIndexedAccess` 要求对 `array[i]` 做 null 检查
- Svelte 5 中 `$: ` 响应式标签已弃用，应使用 `$derived`
