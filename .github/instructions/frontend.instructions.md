# HotSpot 项目前端开发规范

---

> **本规范与后端、通用规范的关系**：
> - 本文件专注前端 Vue3/Nuxt3 开发规范，包括组件管理、状态管理、样式规范、API调用等
> - 配合 `backend.instructions.md`（后端Rust开发规范）和 `project.instructions.md`（通用项目规范）
> - 三个规范文件风格统一、内容互补，共同确保项目代码质量和开发效率

---

## 1. 依赖管理规范

### 包管理操作规则
- **禁止直接修改 package.json 文件**，必须通过包管理器命令操作
- 使用 `pnpm add xxx` 增加依赖
- 使用 `pnpm add -D xxx` 增加开发依赖
- 使用 `pnpm remove xxx` 删除依赖
- 使用包管理器确保添加的依赖和特性是真实存在的，避免安装不存在的包或版本

### 依赖安全与兼容性要求
- 引入新依赖前，需联网查询该依赖包的最新信息，确认是否仍在维护、是否有安全问题
- 特别注意检查依赖包是否被弃用或有替代方案，避免使用过时技术
- 定期检查并更新项目依赖，保持与生态系统同步

## 2. 项目架构规范

### 目录结构和代码组织原则
- 前端代码在 `frontend` 目录下
- 组件放在 `components` 目录，按功能分类
- 页面放在 `pages` 目录
- 布局放在 `layouts` 目录
- 状态管理在 `stores` 目录
- 公共工具函数在 `utils` 目录
- 类型定义在 `types` 目录
- 钩子函数在 `composables` 目录

### 前端项目清理规范

#### Agent模式代码生成清理要求
- **总结文档清理**：删除Agent模式生成的所有总结文档（如*README*.md、*COMPLETION*.md等）
- **演示页面清理**：删除为验证功能而创建的临时演示页面（如auto-user-demo.vue）
- **占位服务清理**：删除仅含TODO注释或占位代码的服务文件，妥善处理相关组件引用
- **构建验证**：清理后必须确保 `npm run build` 无错误

#### 严禁保留的前端文件类型
- **Agent总结文档**：AUTO_USER_REGISTRATION_README.md等功能总结文档
- **临时演示组件**：auto-user-demo.vue等验证性页面组件
- **占位服务文件**：comment.service.ts等仅含基础结构的服务文件
- **测试验证页面**：debug-*.vue、test-*.vue等临时调试组件
- **过渡性配置**：临时的路由配置、中间件配置等

#### 组件引用清理原则
- **引用完整性检查**：删除服务文件时，检查所有引用该服务的组件
- **功能降级处理**：将引用已删除服务的功能改为"功能暂未开放"提示
- **类型定义清理**：清理未使用的类型定义和接口
- **导入语句清理**：移除无效的import语句

#### 清理验证标准
- **构建成功**：执行 `npm run build` 确保无编译错误
- **类型检查**：确保TypeScript类型检查通过
- **运行时测试**：验证关键页面和功能正常运行
- **依赖清理**：移除package.json中未使用的依赖包

## 3. 命名规范

### 组件命名规范
- 组件文件使用 PascalCase 命名（如：`UserProfile.vue`）
- 通用组件放在 `components/common` 目录下
- 业务组件按功能模块放在对应目录下
- 组件名应为多个单词（避免与 HTML 元素冲突），如 Loading、Error 等通用组件可例外

### 路由命名规范
- 使用 Nuxt Pages 自动路由
- 需中间件的页面用 `definePageMeta` 定义
- 路由命名用 kebab-case（如：`user-profile`）
- 布局通过 `layouts` 目录下文件定义

## 4. 布局与样式规范
- 使用 SCSS 作为 CSS 预处理器
- 使用 Naive UI 主题系统
- 组件样式使用 `scoped` 属性隔离
- 公共样式放在 `assets/scss` 目录下
- **SCSS导入策略**：
  - 组件级样式：在 Vue 组件中通过 `@use` 手动导入
  - 全局基础样式（重置样式、字体等）：可在 `nuxt.config.ts` 中注册
  - 变量和混入：在实际使用处手动导入，明确依赖关系
- 使用 CSS 变量定义颜色和尺寸，避免硬编码
- 样式文件组织简洁，按功能分类

### 样式变量和主题规范
- **CSS变量优先**：所有颜色、尺寸、阴影等样式属性优先使用CSS变量
- **语义化命名**：变量名应反映用途而非具体值（如 `--color-text-primary` 而非 `--color-white`）
- **对比度要求**：文字与背景对比度必须满足WCAG AA标准
- **深色主题适配**：所有组件必须在深色主题下保持良好的可读性
- **阴影使用规范**：
  - 文字阴影用于提升可读性：`text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.8)`
  - 盒子阴影用于层次感：使用变量定义的标准阴影值
- **边框规范**：重要边界使用2px边框，状态变化使用颜色区分

### 图标使用规范
- **主要图标库**：使用 Naive UI 内置图标系统和 `@vicons` 生态系统
- **使用方式**：通过通用 Icon 组件调用图标，不直接使用图标组件
- **命名规范**：优先使用 `@vicons/ionicons5` 图标，格式为 `ion:xxx-outline`
- **图标生态**：支持多个图标库
  - `@vicons/ionicons5` - 主要图标库
  - `@vicons/material` - Material Design 图标
  - `@vicons/antd` - Ant Design 图标
  - `@vicons/carbon` - Carbon Design 图标
- **文档参考**：图标使用参考 `frontend/docs/icon-usage.md` 文档

## 5. 状态管理规范
- 使用 Pinia 进行状态管理
- Store 命名应为具体领域名词（如：`auth`、`user`）
- 每个 Store 单独一个文件，职责单一
- 异步操作用 actions 处理
- 使用 `pinia-plugin-persistedstate` 进行持久化

### Store数据结构规范
- **字段命名一致性**：确保store中的数据字段与后端API返回字段一致
- **类型安全**：所有store方法的参数和返回值必须有明确的类型定义
- **方法命名规范**：
  - 获取数据：`fetch*`（如 `fetchNearbyUsers`、`fetchNearbyGroups`）
  - 加载更多：`load*`（如 `loadMoreUsers`）
  - 搜索数据：`search*`（如 `searchUsers`）
- **API调用封装**：store中的异步操作必须正确处理加载状态和错误状态
- **数据转换**：当后端数据格式与前端展示格式不同时，在store中进行数据转换

### 认证和Token存储调试规范

#### 持久化配置规范
- **插件统一**: 只使用官方的 `pinia-plugin-persistedstate`，禁止同时使用多个持久化插件或自定义插件
- **配置简化**: 优先使用 `persist: true` 简化配置，避免复杂的自定义序列化配置
- **nuxt.config.ts配置**: 使用 `piniaPluginPersistedstate` 配置项启用调试模式
- **插件冲突避免**: 确保移除所有自定义持久化插件文件，避免配置冲突

#### 数据类型处理规范
- **Date对象处理**: 注意Date对象在JSON序列化/反序列化时的类型转换问题
- **序列化策略**: 如需自定义序列化，优先考虑在store中处理而非持久化配置中
- **类型安全**: 确保持久化数据的类型与store定义一致

#### 调试策略规范
- **分步调试法**: 将认证流程分解为独立步骤验证
  1. API调用成功性验证
  2. Store数据设置验证  
  3. localStorage存储验证
  4. 页面刷新后数据恢复验证
- **日志完善**: 在关键认证函数中添加详细控制台日志
  - API响应数据记录
  - Store状态变化记录
  - 持久化前后数据对比
- **备份机制**: 在认证数据设置时，同时创建backup localStorage存储作为调试参考

#### 问题定位指南
**当API请求成功但token存储失败时，按优先级检查：**
1. **持久化插件配置**: 检查nuxt.config.ts中的配置是否正确
2. **Store配置**: 验证persist配置是否与插件兼容
3. **数据设置**: 确认setAuthData函数正确执行并设置数据
4. **存储触发**: 验证Pinia状态变化是否触发持久化
5. **Date序列化**: 检查Date对象是否正确序列化/反序列化
6. **浏览器环境**: 确认process.client条件判断正确

#### 调试工具开发规范
- **系统性测试页面**: 创建步骤化的测试页面验证完整认证流程
- **原生HTML工具**: 提供不依赖Vue/Nuxt的调试工具进行底层验证
- **实时状态监控**: 在调试页面中实时显示store状态和localStorage内容
- **错误日志收集**: 完善的错误信息收集和展示机制

#### 终端管理规范
- **单终端操作**: 前端开发过程中尽量在同一终端中操作，避免开启多个终端
- **服务器管理**: 开发者自行管理前端服务器启动，工具不主动启动服务器
## 6. 代码质量管理规范

### 代码风格规范
- 使用 TypeScript 编写代码
- 使用 Composition API 和 `<script setup lang="ts">` 语法
- 遵循 ESLint 规则
- **零警告政策**：项目必须保持编译和 lint 无警告状态
- 所有属性、方法、变量用驼峰命名法
- 方法名用动词开头（如：`fetchData`、`updateUser`）
- 保持代码简洁，避免过度嵌套和复杂逻辑
- 禁止定义未被使用的函数或组件，定义的功能必须有实际调用处（允许导出供外部调用的工具函数/类型）
- 积极解决编译警告和错误，不允许屏蔽警告，优先修复问题根源

### Composition API 强制要求
- **全面采用 Composition API**：所有新代码必须使用 Composition API 模式，禁止使用 Options API
- **服务层 Composition API 化**：所有 `services/` 目录下的服务必须使用 Composition API 格式编写
  - 禁止使用 class 类写法（如 `class ApiService`、`class AuthService` 等）
  - 必须使用函数式 Composition API
- **组件 Composition API 化**：所有 Vue 组件必须使用 `<script setup lang="ts">` 语法
- **状态管理 Composition API 化**：Pinia stores 必须使用 Composition API 语法（`defineStore(() => {})`）
- **兼容性零容忍**：严禁保留任何 Options API 或 class 写法的兼容代码
- **函数命名规范**：
  - 服务层函数：`use*`（如 `useLogin`、`useGetUsers`、`useSendMessage`）
  - 工具函数：`use*`（如 `useFormatDate`、`useValidateEmail`）
  - 状态管理：`use*Store`（如 `useAuthStore`、`useUserStore`）
- **UI 库集成规范**：
  - Naive UI 的 Composition API（如 `useMessage`、`useDialog`）只能在 Vue 组件的 `<script setup>` 中调用
  - 服务层需要 UI 库 API 时，通过参数传递方式获取（如 `initializeWebSocket(config, messageApi)`）
  - 禁止在服务层直接调用 UI 库的 Composition API

### 假数据和演示代码管理规范
- **生产代码禁用假数据**：生产环境的页面和组件不允许使用 mock 数据、演示数据或硬编码的测试数据
- **真实API集成要求**：所有业务页面和组件必须接入真实的后端API，通过store获取数据
- **数据字段标准化**：确保前后端数据字段命名一致
  - 用户ID：统一使用 `user_id`
  - 群组ID：统一使用 `group_id` 
  - 点赞数：统一使用 `like_count`
  - 评论数：统一使用 `comment_count`
- **合理例外情况**：
  - 专门的组件演示页面（如 `demo.vue`）可以包含演示数据
  - 开发调试工具（如 `useMockDynamicWebSocket.ts`）可以保留
  - 单元测试和集成测试中的测试数据
- **Composition API 集成要求**：
  - 所有业务逻辑必须通过 `use*` 函数实现
  - 数据获取统一使用 `useFetch` 或 `useLazyFetch` 调用服务层
  - 状态管理统一使用 `use*Store` 函数

### UI可读性和对比度规范
- **文字对比度**：确保文字与背景有足够的对比度，主文本使用纯白色 `#ffffff`
- **文字阴影**：在复杂背景上添加文字阴影提升可读性
  - 全局文字：`text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.8)`
  - 标题文字：`text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.8)`
- **边框可见性**：重要边框使用2px宽度，焦点状态使用高对比度颜色
- **深色主题优化**：背景色使用极深色调，确保与文字形成强烈对比
- **颜色变量使用**：优先使用CSS变量定义的语义化颜色，避免硬编码颜色值

### 临时文件管理规范
- **允许创建临时文件**：在开发和验证过程中可以创建临时文件
- **即时清理原则**：完成验证或测试后必须立即删除临时文件
- **专门目录管理**：临时文件应放在专门的 `temp/` 或 `temp-tests/` 目录中
- **禁止提交临时文件**：临时文件不应提交到版本控制系统
- **清理检查**：代码完成后必须检查并删除所有临时文件

### 项目清理和维护规范
- **无用文件清理**：定期检查并删除未使用的文件、组件和代码
- **备份文件管理**：开发过程中产生的备份文件（如 `*-old.vue`）必须及时清理
- **Mock数据清理**：项目发布前必须清理所有mock数据和演示代码
- **依赖清理**：删除未使用的依赖包，保持 `package.json` 整洁
- **代码注释维护**：及时更新或删除过时的注释，保持代码注释的准确性

### 数据集成质量控制
- **API集成验证**：确保所有页面的数据来源都是真实API
- **空状态处理**：所有数据加载状态都要有相应的空状态提示
- **错误状态处理**：API调用失败时要有合适的错误提示和重试机制
- **加载状态管理**：所有异步操作都要有加载状态指示器

## 7. 导入与组件使用规范

### 组件导入规范
- **组件导入**：所有组件必须手动导入，禁用组件自动导入
  - 手动导入组件有助于提高代码可移植性和版本兼容性
  - 明确组件依赖关系，便于代码维护和重构
  - 避免因 Nuxt 版本更新导致的组件导入不兼容问题
- **Naive UI 导入**：所有 Naive UI  组件和图标必须手动导入
  - Naive UI 组件需要手动导入（如：`import { NButton } from 'naive-ui'`）
  - Naive UI 图标需要手动导入
  - 禁止使用自动导入，确保明确的依赖关系
- **API 导入**：保留 Vue 3、Nuxt 3、Composables 等 API 的自动导入
  - Vue 3 API（`ref`、`computed`、`watch`、`onMounted` 等）自动导入
  - Nuxt 3 API（`useAsyncData`、`useFetch`、`navigateTo` 等）自动导入
  - 项目 Composables（`~/composables` 下的函数）自动导入
- **组件引用规范**：
  - 所有 Vue 组件必须在 `<script setup>` 中显式导入
  - 项目组件使用相对路径或 `~/components` 路径导入

## 8. API 调用规范
- **使用 Nuxt 的 Fetch API 发起请求**：优先使用 `$fetch`、`useFetch`、`useLazyFetch` 等 Nuxt 3 原生 API
- API 调用统一在 `services` 目录下封装，页面/组件禁止直接写 fetch/axios 请求
- 接口基础 URL 通过环境变量配置
- 接口返回数据类型要定义清楚
- 错误处理要完善
- **服务层实现规范**：
  - 服务层函数使用 `$fetch` 进行 HTTP 请求
  - 统一错误处理和响应格式化
  - 支持请求拦截器和响应拦截器
  - 页面层使用 `useFetch` 或 `useLazyFetch` 调用服务层函数
- **Composition API 服务层架构**：
  - 所有服务必须采用函数式 Composition API 写法
  - 每个 API 端点对应一个 `use*` 函数（如 `useLogin`、`useGetUserProfile`、`useCreateGroup`）
  - 服务函数返回响应式数据和状态管理（loading、error、data）
  - 复杂服务可以组合多个基础 `use*` 函数实现

## 9. 性能优化规范
- 组件合理拆分，避免过大组件
- 列表使用虚拟滚动优化
- 大数据操作用 Web Worker
- 图片使用懒加载
- 按需导入第三方库
- 删除不必要依赖，减少打包体积
- 使用轻量级工具函数，避免引入大型库

## 10. Nuxt 配置与框架规范

### Nuxt 配置规范
- 使用 `nuxt.config.ts` 进行全局配置
- 插件放在 `plugins` 目录下
- 中间件放在 `middleware` 目录下
- 静态资源放在 `public` 目录下
- 合理配置构建选项，优化打包体积
- 使用按需导入减少不必要代码

### Nuxt 文件处理规范
- **不要直接替换** Nuxt 自动生成的文件
- 在 Nuxt 生成文件基础上修改和扩展，保留原始结构和元数据
- 修改生成文件时，确保不影响 Nuxt 核心功能
- 使用 `<script setup>` 和 `<template>` 添加自定义功能
- 如需添加中间件或页面元数据，使用 `definePageMeta` 而非替换文件

## 11. Nuxt 文件命名与使用规范

#### 适用目录
- 仅允许在 Nuxt 约定的以下目录下使用 `.client.ts`、`.server.ts`、`.client.vue`、`.server.vue` 等后缀：
  - `pages/`
  - `layouts/`
  - `middleware/`
  - `plugins/`
  - `composables/`
  - `components/`
  - `app.vue`（顶层文件可用）

#### 规范说明
- 上述目录下，`.client.ts`/`.client.vue` 只在客户端加载和执行，`.server.ts`/`.server.vue` 只在服务端加载和执行
- 在 `utils/`、`services/`、`stores/` 等自定义目录下，**禁止**使用 `.client.ts`、`.server.ts` 等后缀，否则 Nuxt 不识别，可能导致运行时错误或 SSR 问题。如需区分环境，相关逻辑应迁移到 composables/ 目录。
- 需区分环境的逻辑，必须放在 Nuxt 约定目录下，并用 `.client` 或 `.server` 后缀
- 普通工具、服务、数据文件请直接用常规命名（如 `api.ts`、`xxx.service.ts`）
- **所有创建的文件必须被实际使用**，不允许存在未被引用的文件或函数实现
- 如果需要特定环境（客户端/服务端）的实现，应直接使用对应的 `.client` 或 `.server` 文件，严禁同时存在未被使用的通用实现

#### 使用示例
- 正确：`components/Comments.client.vue`、`plugins/analytics.client.ts`、`composables/useWindowSize.client.ts`、`utils/api.ts`
- 错误：`utils/api.client.ts`、`services/auth.server.ts`

#### 配对使用案例
- 同时存在 `.client` 和 `.server` 版本时，Nuxt 会：
  - 服务端渲染用 `.server` 版本
  - 客户端渲染/激活用 `.client` 版本

## 12. 后端异步任务系统协作规范

### 异步操作处理原则

#### 用户注册流程优化
- **立即反馈**: 用户注册完成后立即显示成功状态，不等待位置初始化等异步任务
- **渐进增强**: 异步任务完成后通过WebSocket或轮询更新用户状态
- **容错设计**: 异步任务失败时不影响用户正常使用，提供重试选项

#### 后端任务队列感知设计
```typescript
// 注册成功后的处理逻辑
const handleRegistrationSuccess = async (userData: UserLoginResult) => {
  // 1. 立即更新用户状态
  authStore.setAuthData(userData)
  
  // 2. 显示注册成功消息
  ElMessage.success('注册成功！正在为您初始化账户...')
  
  // 3. 导航到主页面（不等待异步任务）
  await navigateTo('/')
  
  // 4. 通过WebSocket监听位置初始化完成通知（可选）
  // websocketStore.subscribe('location_initialized', handleLocationReady)
}
```

#### WebSocket通知处理
- **任务状态通知**: 监听后端任务队列发送的状态更新通知
- **用户体验增强**: 异步任务完成时显示相应的成功消息或状态更新
- **错误处理**: 监听任务失败通知，提供重试或替代方案

### API调用优化规范

#### 非阻塞接口设计
- **分离关注点**: 主要业务操作（注册、登录）与辅助功能（位置初始化、通知发送）分离
- **容错处理**: API调用失败时提供降级方案，避免功能完全不可用
- **状态管理**: 合理管理加载状态，区分主要操作和辅助操作的加载状态

#### 异步状态展示
```vue
<template>
  <div class="registration-status">
    <!-- 主要状态：注册是否完成 -->
    <div v-if="registrationLoading" class="loading">
      正在注册您的账户...
    </div>
    
    <!-- 成功状态 -->
    <div v-else-if="registrationSuccess" class="success">
      <h3>注册成功！</h3>
      
      <!-- 辅助状态：异步任务进度 -->
      <div class="async-tasks">
        <div class="task-item">
          <span>位置初始化</span>
          <TaskStatus :status="locationStatus" />
        </div>
        <div class="task-item">
          <span>欢迎通知</span>
          <TaskStatus :status="notificationStatus" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
// 主要状态
const registrationLoading = ref(false)
const registrationSuccess = ref(false)

// 异步任务状态
const locationStatus = ref('pending') // pending | processing | completed | failed
const notificationStatus = ref('pending')

// WebSocket监听异步任务状态
const { wsStatus } = useWebSocket()
watch(wsStatus, (message) => {
  if (message.type === 'location_initialized') {
    locationStatus.value = 'completed'
  }
  // 处理其他异步任务状态...
})
</script>
```

### 错误处理与用户体验

#### 分层错误处理
- **主要流程错误**: 直接影响用户操作的错误（如注册失败）必须阻断流程并显示错误
- **辅助功能错误**: 不影响主要功能的错误（如位置初始化失败）应友好提示但不阻断
- **重试机制**: 为异步任务失败提供手动重试选项

#### 用户反馈设计
```typescript
// 错误处理示例
const handleAsyncTaskError = (taskType: string, error: Error) => {
  switch (taskType) {
    case 'location_init':
      // 位置初始化失败 - 不阻断用户使用
      ElNotification({
        title: '位置初始化失败',
        message: '您可以稍后在设置中手动设置位置',
        type: 'warning',
        duration: 5000,
        showClose: true
      })
      break
      
    case 'welcome_notification':
      // 欢迎通知发送失败 - 静默处理
      console.warn('欢迎通知发送失败:', error)
      break
      
    default:
      // 其他异步任务错误
      ElMessage.warning('某些后台任务执行失败，但不影响正常使用')
  }
}
```

### 状态同步与数据一致性

#### 乐观更新策略
- **即时响应**: 用户操作后立即更新UI状态，不等待所有异步任务完成
- **状态回滚**: 如果关键异步任务失败，提供状态回滚机制
- **数据同步**: 定期从服务器同步数据，确保客户端状态准确

#### Store状态管理
```typescript
// 异步任务感知的状态管理
export const useAsyncTaskStore = defineStore('asyncTask', () => {
  const pendingTasks = ref<Map<string, AsyncTask>>(new Map())
  
  const addTask = (taskId: string, taskType: string) => {
    pendingTasks.value.set(taskId, {
      id: taskId,
      type: taskType,
      status: 'pending',
      createdAt: Date.now()
    })
  }
  
  const updateTaskStatus = (taskId: string, status: AsyncTaskStatus) => {
    const task = pendingTasks.value.get(taskId)
    if (task) {
      task.status = status
      task.updatedAt = Date.now()
    }
  }
  
  const removeCompletedTasks = () => {
    pendingTasks.value.forEach((task, id) => {
      if (task.status === 'completed' && 
          Date.now() - task.updatedAt > 30000) { // 30秒后清理
        pendingTasks.value.delete(id)
      }
    })
  }
  
  return {
    pendingTasks: readonly(pendingTasks),
    addTask,
    updateTaskStatus,
    removeCompletedTasks
  }
})
```

### 性能优化考虑

#### 避免过度轮询
- **WebSocket优先**: 优先使用WebSocket接收异步任务状态更新
- **智能轮询**: 如果WebSocket不可用，使用指数退避的轮询策略
- **状态缓存**: 合理缓存异步任务状态，避免重复查询

#### 渐进式功能激活
```typescript
// 功能渐进式激活示例
export const useProgressiveFeatures = () => {
  const featuresReady = ref({
    location: false,
    notifications: false,
    recommendations: false
  })
  
  const checkFeatureReadiness = async () => {
    try {
      const status = await $fetch('/api/user/features-status')
      featuresReady.value = status
    } catch (error) {
      console.warn('无法获取功能状态，使用默认配置')
    }
  }
  
  const enableFeatureWhenReady = (feature: string, callback: () => void) => {
    watch(() => featuresReady.value[feature], (ready) => {
      if (ready) {
        callback()
      }
    }, { immediate: true })
  }
  
  return {
    featuresReady: readonly(featuresReady),
    checkFeatureReadiness,
    enableFeatureWhenReady
  }
}
```

### 开发调试支持

#### 异步任务调试工具
- **任务状态面板**: 开发环境下显示当前用户的异步任务执行状态
- **手动重试**: 提供手动触发异步任务的调试接口
- **模拟失败**: 测试环境下模拟异步任务失败场景

#### 监控和日志
```typescript
// 异步任务监控
export const useAsyncTaskMonitoring = () => {
  const logTaskEvent = (event: 'start' | 'success' | 'error', taskType: string, details?: any) => {
    if (process.dev) {
      console.log(`[AsyncTask] ${event.toUpperCase()}: ${taskType}`, details)
    }
    
    // 生产环境下发送到监控系统
    if (process.env.NODE_ENV === 'production') {
      // 发送监控数据...
    }
  }
  
  return { logTaskEvent }
}
```



## 12.1 后端架构合理性协作规范

### 前后端架构优化协作原则

#### 架构合理性验证参与
- **业务逻辑审查**：前端开发过程中发现后端设计不合理时，主动提出优化建议
- **真实场景验证**：基于前端用户体验需求，验证后端架构设计的合理性
- **功能重复识别**：在API调用过程中识别重复功能，建议后端简化

#### 已完成的协作优化案例（2025年7月15日）

**notification存储系统优化**：
```typescript
// ❌ 不合理的前端调用模式
// 问题：通知需要同时调用WebSocket和数据库API
await this.notificationService.saveToDatabase(notification)
await this.websocketService.sendRealtime(notification)

// ✅ 优化后的合理设计
// 只需要实时WebSocket通知，无需数据库操作
await this.websocketService.sendNotification(notification)
```

**优化收益**：
- **前端代码简化**：减少了通知相关的API调用复杂度
- **实时性提升**：通知推送更加及时，无需等待数据库操作
- **状态管理简化**：前端无需管理通知的持久化状态

#### 前端驱动的架构优化原则

**用户体验驱动后端设计**：
- **API简洁性**：复杂的后端逻辑应该封装在单一API中，避免前端调用多个API组合功能
- **实时性优先**：对于需要实时反馈的功能，优先使用WebSocket而非轮询数据库
- **错误处理统一**：后端应提供一致的错误响应格式，便于前端统一处理

**协作沟通机制**：
- **周期性架构审查**：前后端定期（月度）进行架构合理性联合审查
- **问题及时反馈**：前端发现后端设计问题时，通过规范化流程反馈
- **优化方案共同制定**：重大架构调整前，前后端共同评估影响和实施方案

### 前端代码中的后端协作最佳实践

#### API调用合理性判断
```typescript
// ✅ 合理的API设计调用
// 单一职责，功能明确
const user = await userService.createUser(userData)

// ❌ 不合理的设计模式
// 需要多次API调用完成单一业务逻辑
const user = await userService.createUser(userData)
await notificationService.saveWelcomeNotification(user.id)
await userService.initializeLocation(user.id)
```

#### 状态管理与后端状态同步
```typescript
// 优化前后端状态管理配合
export const useOptimizedUserFlow = () => {
  const handleUserAction = async (action: UserAction) => {
    // 乐观更新：立即更新UI状态
    updateUIState(action)
    
    try {
      // 调用优化的后端API（单一调用完成复杂逻辑）
      const result = await backendService.handleUserAction(action)
      
      // 确认更新
      confirmUIState(result)
    } catch (error) {
      // 回滚UI状态
      rollbackUIState()
      handleError(error)
    }
  }
}
```

---

## 13. 用户体验优化规范

### 自动临时用户注册策略

#### 核心理念
- **零门槛体验**：用户首次访问应用时无需主动注册，自动创建临时用户身份
- **渐进式引导**：通过临时用户体验核心功能，再引导升级为正式用户
- **无缝过渡**：临时用户数据在升级为正式用户时应完整保留

#### 实现规范

##### 自动检测与注册逻辑
```typescript
// composables/useAutoUserRegistration.ts
export const useAutoUserRegistration = () => {
  const authStore = useAuthStore()
  const router = useRouter()
  
  const checkAndCreateTempUser = async () => {
    // 检测用户登录状态
    if (authStore.isAuthenticated) {
      return authStore.currentUser
    }
    
    // 检查本地是否有临时用户信息
    const tempUserData = localStorage.getItem('temp_user_backup')
    if (tempUserData) {
      try {
        const userData = JSON.parse(tempUserData)
        await authStore.setAuthData(userData)
        console.log('恢复本地临时用户身份')
        return userData
      } catch (error) {
        console.warn('临时用户数据无效，将创建新的临时用户')
      }
    }
    
    // 自动创建临时用户
    try {
      console.log('检测到新用户，正在创建临时账户...')
      const tempUser = await authStore.createTempUser()
      
      if (tempUser) {
        ElMessage({
          message: '欢迎使用 HotSpot！已为您创建临时账户，您可以立即开始探索',
          type: 'success',
          duration: 4000,
          showClose: true
        })
        return tempUser
      }
    } catch (error) {
      console.error('创建临时用户失败:', error)
      // 创建失败时的降级处理
      ElMessage({
        message: '欢迎使用 HotSpot！您可以浏览公开内容或注册账户享受完整功能',
        type: 'info',
        duration: 5000,
        showClose: true
      })
    }
    
    return null
  }
  
  return {
    checkAndCreateTempUser
  }
}
```

##### 页面级集成规范
```typescript
// layouts/default.vue 或 app.vue 中的集成
export default {
  async setup() {
    const { checkAndCreateTempUser } = useAutoUserRegistration()
    const authStore = useAuthStore()
    
    // 页面加载时自动检查用户状态
    onMounted(async () => {
      // 仅在客户端执行
      if (process.client) {
        await checkAndCreateTempUser()
      }
    })
    
    // 监听路由变化，确保关键页面有用户身份
    const router = useRouter()
    router.beforeEach(async (to) => {
      // 需要用户身份的页面列表
      const requiresUser = ['/profile', '/my-groups', '/settings']
      
      if (requiresUser.some(path => to.path.startsWith(path))) {
        if (!authStore.isAuthenticated) {
          await checkAndCreateTempUser()
        }
      }
    })
  }
}
```

#### 临时用户功能权限管理

##### 功能分级策略
```typescript
// stores/userCapability.ts
export const useUserCapabilityStore = defineStore('userCapability', () => {
  const authStore = useAuthStore()
  
  const capabilities = computed(() => {
    const user = authStore.currentUser
    
    if (!user) {
      // 游客权限
      return {
        canViewPublicContent: true,
        canCreateGroups: false,
        canSendMessages: false,
        canCreateMoments: false,
        canLikeContent: false,
        maxGroupJoins: 0
      }
    }
    
    if (user.is_temporary) {
      // 临时用户权限
      return {
        canViewPublicContent: true,
        canCreateGroups: true,
        canSendMessages: true,
        canCreateMoments: true,
        canLikeContent: true,
        maxGroupJoins: 3, // 限制加入群组数量
        hasExpiryReminder: true // 显示升级提醒
      }
    }
    
    // 正式用户权限
    return {
      canViewPublicContent: true,
      canCreateGroups: true,
      canSendMessages: true,
      canCreateMoments: true,
      canLikeContent: true,
      maxGroupJoins: Infinity,
      canInviteUsers: true,
      canManageGroups: true
    }
  })
  
  return { capabilities }
})
```

##### 升级引导策略
```typescript
// composables/useUserUpgradeGuide.ts
export const useUserUpgradeGuide = () => {
  const authStore = useAuthStore()
  const { capabilities } = useUserCapabilityStore()
  
  const showUpgradePrompt = (trigger: 'time' | 'action' | 'limit') => {
    if (!authStore.currentUser?.is_temporary) return
    
    const messages = {
      time: '您已使用临时账户一段时间，升级为正式用户可享受更多功能！',
      action: '升级为正式用户，解锁完整的社交功能！',
      limit: '您已达到临时账户限制，升级后可享受无限制使用！'
    }
    
    ElNotification({
      title: '升级账户',
      message: messages[trigger],
      type: 'info',
      duration: 0, // 不自动关闭
      showClose: true,
      customClass: 'upgrade-notification',
      onClick: () => {
        navigateTo('/auth?mode=upgrade')
      }
    })
  }
  
  const checkUpgradeOpportunities = () => {
    if (!authStore.currentUser?.is_temporary) return
    
    // 基于使用时长的提醒
    const registerTime = new Date(authStore.currentUser.created_at).getTime()
    const hoursSinceRegister = (Date.now() - registerTime) / (1000 * 60 * 60)
    
    if (hoursSinceRegister > 2) { // 使用超过2小时
      showUpgradePrompt('time')
    }
  }
  
  return {
    showUpgradePrompt,
    checkUpgradeOpportunities
  }
}
```

#### 数据迁移与保留策略

##### 临时用户数据保护
```typescript
// stores/auth.ts 中的升级方法扩展
export const useAuthStore = defineStore('auth', () => {
  // ...existing code...
  
  const upgradeTempUser = async (upgradeData: {
    loginId: string
    nickname: string
    password: string
  }) => {
    if (!currentUser.value?.is_temporary) {
      throw new Error('当前用户不是临时用户')
    }
    
    try {
      // 调用升级接口，后端会保留临时用户的所有数据
      const upgradeResult = await authApi.upgradeTempUser({
        temp_user_id: currentUser.value.user_id,
        login_id: upgradeData.loginId,
        nickname: upgradeData.nickname,
        password: upgradeData.password
      })
      
      if (upgradeResult) {
        // 更新本地用户信息
        await setAuthData(upgradeResult)
        
        // 清理临时用户备份数据
        localStorage.removeItem('temp_user_backup')
        
        ElMessage.success('升级成功！您的所有数据已保留，现在可以享受完整功能！')
        return upgradeResult
      }
    } catch (error) {
      console.error('升级账户失败:', error)
      throw error
    }
  }
  
  return {
    // ...existing returns...
    upgradeTempUser
  }
})
```

#### 用户界面优化

##### 状态指示器
```vue
<!-- components/common/UserStatusIndicator.vue -->
<template>
  <div class="user-status-indicator">
    <div v-if="authStore.currentUser?.is_temporary" class="temp-user-badge">
      <Icon name="ep:user" class="icon" />
      <span>临时用户</span>
      <el-button 
        type="primary" 
        size="small" 
        @click="navigateTo('/auth?mode=upgrade')"
        class="upgrade-btn"
      >
        升级
      </el-button>
    </div>
    <div v-else-if="authStore.currentUser" class="regular-user-badge">
      <Icon name="ep:user-filled" class="icon" />
      <span>{{ authStore.currentUser.nickname }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton } from 'naive-ui'
import Icon from '~/components/common/Icon.vue'

const authStore = useAuthStore()
</script>

<style scoped>
.temp-user-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  background: rgba(255, 193, 7, 0.1);
  border: 1px solid rgba(255, 193, 7, 0.3);
  border-radius: 16px;
  font-size: 12px;
}

.upgrade-btn {
  padding: 2px 8px;
  height: 20px;
  font-size: 10px;
}
</style>
```

##### 功能限制提示
```vue
<!-- components/common/FeatureGate.vue -->
<template>
  <div class="feature-gate">
    <slot v-if="hasPermission" />
    <div v-else class="permission-prompt">
      <Icon name="ep:lock" class="lock-icon" />
      <p>{{ promptMessage }}</p>
      <el-button 
        type="primary" 
        size="small"
        @click="handleUpgradeClick"
      >
        {{ upgradeButtonText }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  feature: string
  promptMessage?: string
  upgradeButtonText?: string
}

const props = withDefaults(defineProps<Props>(), {
  promptMessage: '升级账户以使用此功能',
  upgradeButtonText: '立即升级'
})

const { capabilities } = useUserCapabilityStore()
const authStore = useAuthStore()

const hasPermission = computed(() => {
  // 根据不同功能检查权限
  switch (props.feature) {
    case 'createGroup':
      return capabilities.value.canCreateGroups
    case 'sendMessage':
      return capabilities.value.canSendMessages
    // 添加更多功能检查...
    default:
      return true
  }
})

const handleUpgradeClick = () => {
  if (authStore.currentUser?.is_temporary) {
    navigateTo('/auth?mode=upgrade')
  } else {
    navigateTo('/auth?mode=register')
  }
}
</script>
```

### 性能与安全考虑

##### 防重复创建机制
```typescript
// 防止短时间内重复创建临时用户
let tempUserCreationPromise: Promise<any> | null = null

export const createTempUserSafely = async () => {
  if (tempUserCreationPromise) {
    return tempUserCreationPromise
  }
  
  tempUserCreationPromise = authStore.createTempUser()
  
  try {
    const result = await tempUserCreationPromise
    return result
  } finally {
    tempUserCreationPromise = null
  }
}
```

##### 数据清理策略
```typescript
// 定期清理过期的临时用户数据
export const useTempUserCleanup = () => {
  const cleanupExpiredData = () => {
    const tempUserData = localStorage.getItem('temp_user_backup')
    if (tempUserData) {
      try {
        const userData = JSON.parse(tempUserData)
        const createTime = new Date(userData.created_at).getTime()
        const daysSinceCreate = (Date.now() - createTime) / (1000 * 60 * 60 * 24)
        
        // 临时用户数据超过7天自动清理
        if (daysSinceCreate > 7) {
          localStorage.removeItem('temp_user_backup')
          console.log('已清理过期的临时用户数据')
        }
      } catch (error) {
        localStorage.removeItem('temp_user_backup')
      }
    }
  }
  
  // 页面加载时检查
  onMounted(cleanupExpiredData)
  
  return { cleanupExpiredData }
}
```

### 实施指导原则

1. **用户体验优先**：确保自动注册过程对用户透明且不打扰
2. **数据安全**：临时用户数据应当安全存储，避免泄露
3. **性能优化**：避免重复创建和不必要的API调用
4. **渐进增强**：通过实际使用价值引导用户升级，而非强制要求
5. **错误处理**：自动注册失败时提供合适的降级体验

### 测试验证要点

- [ ] 新用户首次访问自动创建临时账户
- [ ] 临时用户功能权限正确限制
- [ ] 升级流程数据完整保留
- [ ] 页面刷新后临时用户身份正确恢复
- [ ] 过期数据自动清理
- [ ] 网络异常时的降级体验

---

## 14. 样式规范

### 游戏UI设计规范

#### 技能栏设计原则
- **图标一致性**：技能栏中的所有按钮必须有有效的图标，使用MDI图标库
- **功能模块化**：每个技能按钮对应一个明确的功能模块
- **交互反馈**：技能按钮应有明确的激活/非激活状态，支持拖拽排序
- **弹窗式操作**：复杂功能（如地图）采用弹窗形式，避免占用固定屏幕空间

#### Minimap设计规范
- **游戏化体验**：借鉴游戏中的minimap设计，通过技能栏按钮触发弹窗显示
- **空间优化**：移除固定位置的minimap，避免界面拥挤
- **功能整合**：将"探索附近"功能整合到minimap弹窗中，避免功能重复
- **位置优先**：minimap弹窗应提供更好的位置信息展示和交互

#### 技能栏图标配置要求
- **location**: `mdi:map-marker-radius` - 位置管理
- **users**: `mdi:account-group` - 附近用户  
- **messages**: `mdi:message-text` - 消息中心
- **groups**: `mdi:account-multiple` - 我的群组
- **profile**: `mdi:account-circle` - 个人资料
- **settings**: `mdi:cog` - 设置
- **map**: `mdi:map` - 地图视图（新增，替代固定minimap）
- **create**: `mdi:plus` - 创建群组

#### 功能整合原则
- **去重化**：移除重复功能，如"探索附近"功能已被minimap覆盖
- **模块化**：每个功能模块有独立的入口和界面
- **响应式**：所有UI组件支持移动端和桌面端适配

### SCSS/CSS响应式设计规范
- **断点系统**：使用项目中预定义的SCSS混入处理响应式设计
  ```scss
  // 使用现有的SCSS混入
  @use '~/assets/scss/mixins' as *;
  
  .component {
    // 默认样式（桌面端）
    width: 400px;
    
    // 平板端
    @include respond-below(1024px) {
      width: 300px;
    }
    
    // 手机端
    @include respond-below(768px) {
      width: 100%;
    }
  }
  ```

- **CSS类控制策略**：使用CSS类而非JavaScript控制元素显示
  ```scss
  // 响应式显示类
  .compact-only {
    display: block;
    
    @include respond-above(768px) {
      display: none;
    }
  }
  
  .full-only {
    display: none;
    
    @include respond-above(768px) {
      display: block;
    }
  }
  
  .tablet-only {
    display: none;
    
    @include respond-between(768px, 1024px) {
      display: block;
    }
  }
  ```

- **组件响应式样式模式**：在Vue组件中的标准响应式写法
  ```vue
  <template>
    <div class="responsive-component">
      <div class="content">内容</div>
      <div class="sidebar desktop-only">侧边栏</div>
    </div>
  </template>
  
  <style scoped lang="scss">
  @use '~/assets/scss/mixins' as *;
  
  .responsive-component {
    display: flex;
    gap: 1rem;
    
    @include respond-below(768px) {
      flex-direction: column;
    }
    
    .content {
      flex: 1;
      
      @include respond-below(768px) {
        order: 2;
      }
    }
    
    .sidebar {
      width: 300px;
      
      @include respond-below(1024px) {
        width: 250px;
      }
    }
  }
  </style>
  ```

- **禁止的响应式模式**：严禁使用以下JavaScript响应式检测方式
  ```javascript
  // ❌ 禁止的做法
  const isCompact = ref(window.innerWidth < 768)
  const isTablet = computed(() => screen.width >= 768 && screen.width < 1024)
  
  // ❌ 禁止的组件属性
  :fullscreen="isCompact"
  :class="{ 'compact-layout': isCompact }"
  v-if="!isFullScreen"
  ```

- **推荐的响应式模式**：正确的CSS优先响应式设计
  ```vue
  <template>
    <!-- ✅ 推荐的做法 -->
    <n-modal class="responsive-modal">
      <div class="modal-content">内容</div>
    </n-modal>
  </template>
  
  <style scoped lang="scss">
  @use '~/assets/scss/mixins' as *;
  
  .responsive-modal {
    :deep(.n-modal) {
      width: 80%;
      max-width: 600px;
      
      @include respond-below(768px) {
        width: 95%;
        height: 90vh;
        margin: 5vh auto;
      }
    }
  }
  </style>
  ```