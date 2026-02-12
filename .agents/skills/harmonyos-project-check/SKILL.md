---
name: harmonyos-project-check
description: Check a HarmonyOS ArkTS project for compilation errors and IDE-visible linter warnings using hvigorw and codelinter. Use when verifying code quality after changes, before commits, or when the user asks to check the HarmonyOS/harmonyapp project for errors.
compatibility: Requires DevEco Studio SDK (macOS default /Applications/DevEco-Studio.app/).
metadata:
  category: quality-check
---

# HarmonyOS 项目检查

编译通过 ≠ 没有问题。编译器和 Code Linter 是**独立工具**，检查不同维度，必须分别运行。

## 编译（捕获编译错误）

```bash
cd harmonyapp
DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk" \
  node "/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js" \
  clean --mode module -p product=default assembleHap \
  --info --analyze=ultrafine --parallel --incremental \
  --enable-build-script-type-check --no-daemon
```

捕获范围：ArkTS 语法错误、类型不匹配、缺少导入、组件接口不匹配、资源引用错误（`$r()` 引用不存在的资源）、HAP 打包问题。

## Code Linter（捕获 IDE 可见警告）

```bash
cd harmonyapp
DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk" \
  node "/Applications/DevEco-Studio.app/Contents/plugins/codelinter/run/index.js" \
  -c code-linter.json5 entry/src/main/ets/
```

捕获范围（对应 IDE 中的波浪线）：

| 规则集 | 检查内容 | 典型问题 |
|--------|----------|----------|
| `@performance/recommended` | 性能 | ForEach 缺少唯一 key、不必要的状态更新、低效渲染 |
| `@typescript-eslint/recommended` | TS 规范 | 未使用变量、类型安全、隐式 any |
| `@security/*` (12 条) | 加密安全 | 不安全的 AES/RSA/Hash/MAC/DH/DSA/3DES 用法 |

## 解读输出

codelinter 输出：

```
entry/src/main/ets/path/file.ets (line, col): warning rule-name: message
entry/src/main/ets/path/file.ets (line, col): error rule-name: message
```

- **error** — 必须修复，IDE 中红色波浪线
- **warning** — 应当修复，IDE 中黄色波浪线
- 规则名（如 `@typescript-eslint/no-unused-vars`）标明具体违规类型

## 常见陷阱

- 编译成功但 codelinter 报几十条警告 → 正常，它们是独立工具
- `@performance` 规则容易在 ForEach、LazyForEach、@State 变量使用上报警
- `@security` 规则只在使用加密 API 时触发
- codelinter 规则集来自 DevEco Studio 内置插件，不是 npm 安装的 ESLint
