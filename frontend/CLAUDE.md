# CLAUDE.md

> **注意**：本项目为 **主前端**（SvelteKit），当前所有前端开发在此进行，**同时是 UI/UX 与设计系统的唯一事实源**。改前端样式前先阅读 `.claude/rules/` 下的设计规范。

## 构建 & 测试

- `bun run build` — 构建
- `bun check` — 类型检查（svelte-check）
- `bun lint` — 格式检查（prettier --check）
- `bun run format` — 格式化（prettier --write）

> **测试**：package.json 无 `test` 脚本，暂无自动化测试。

## 约束

- **API 路径**：统一从 `endpoints.ts` 引用。
- **请求封装**：所有请求走 `client.ts` 和 `services/*`。
- **HTTP 方法**：统一 POST。
- **响应判断**：业务成功唯一依据 `code === 0`。
- **分页**：cursor 只透传，不解析、不构造。

## UI/UX 设计规范

见 `.claude/rules/` 下按文件类型懒加载的规则：

| 文件               | 适用范围                                  | 内容                                                                                                                     |
| ------------------ | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `ui-layout.md`     | `src/**/*.svelte`                         | 页面布局系统（`.page-shell`、**容器查询原语** `.density-grid`/`.sidebar-layout`/`.switcher`、**platformStore 语义化派生量**、红线） |
| `design-tokens.md` | `src/**/*.{svelte,css}`                   | 设计 token（颜色、表面质感、圆角、字号、间距、动效、图标、**对比度/WCAG**）                                                               |
| `navigation.md`    | `src/**/*.svelte`, `config/navigation.ts` | 导航框架（设备导航模型、返回按钮、入口去重、路由收敛、审查清单）                                                         |

**响应式适配原则**：组件层用容器查询原语（`.density-grid` 等，组件按自身容器宽度自适应），视口层判断用语义化派生量（`useMasterDetail`/`useSidebarChrome`/`hideRedundantEntry`）。业务页禁止裸用 `isMobile/isTablet/isDesktop/isWide` 切换纯布局密度。

改代码前花 10 秒看对应规则，避免不一致。
