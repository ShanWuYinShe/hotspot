# 统一格式化系统使用指南

## 🎯 设计原则

**严禁在 Vue 文件中实现格式化逻辑！**

所有格式化功能必须通过统一的格式化系统实现，确保：
- ✅ 国际化支持
- ✅ 统一的用户体验
- ✅ 易于维护和测试
- ✅ 类型安全

## 📚 使用方法

### 1. 推荐方式：使用 `useFormatters()` Composable（支持国际化）

```vue
<template>
  <div>
    <!-- 时间格式化 -->
    <p>发布时间：{{ formatTime(moment.created_at) }}</p>
    
    <!-- 距离格式化 -->
    <p>距离：{{ formatDistance(user.distance) }}</p>
    
    <!-- 数量格式化 -->
    <p>点赞数：{{ formatCount(moment.likes_count) }}</p>
    
    <!-- 日期格式化 -->
    <p>详细时间：{{ formatDate(moment.created_at, 'long') }}</p>
    
    <!-- 文件大小格式化 -->
    <p>文件大小：{{ formatFileSize(file.size) }}</p>
  </div>
</template>

<script setup lang="ts">
// 导入格式化 composable
const { formatTime, formatDistance, formatCount, formatDate, formatFileSize } = useFormatters()

// 你的其他逻辑...
</script>
```

### 2. 后备方式：直接导入工具函数（不支持国际化）

```vue
<script setup lang="ts">
import { formatTime, formatDistance } from '~/utils/formatters'

// 仅在无法使用 composable 的场景下使用
// 例如：在 Pinia store 或其他非 Vue 组件中
</script>
```

## 🌍 国际化支持

格式化系统完全支持国际化，会根据用户的语言设置自动切换：

### 中文 (zh-CN)
- `formatTime('2024-01-01T10:00:00Z')` → "3小时前"
- `formatDistance(1500)` → "1.5公里"
- `formatCount(15000)` → "1.5万"

### 英文 (en-US) - 未来支持
- `formatTime('2024-01-01T10:00:00Z')` → "3 hours ago"
- `formatDistance(1500)` → "1.5 km"
- `formatCount(15000)` → "15k"

## 📖 API 参考

### `formatTime(dateString)`
格式化时间为相对时间显示

**参数：**
- `dateString`: `string | Date | null | undefined` - ISO 8601 时间字符串或 Date 对象

**返回：** `string` - 格式化后的相对时间

**示例：**
```typescript
formatTime('2024-01-01T10:00:00Z') // "3小时前"
formatTime(null) // "-"
formatTime(new Date()) // "刚刚"
```

### `formatDistance(distance)`
格式化距离为可读字符串

**参数：**
- `distance`: `number | null | undefined` - 距离（米）

**返回：** `string` - 格式化后的距离

**示例：**
```typescript
formatDistance(500) // "500米"
formatDistance(1500) // "1.5公里"
formatDistance(null) // "-"
```

### `formatCount(count)`
格式化数字为可读的数量字符串

**参数：**
- `count`: `number | null | undefined` - 数量

**返回：** `string` - 格式化后的数量

**示例：**
```typescript
formatCount(999) // "999"
formatCount(1500) // "1.5k"
formatCount(15000) // "1万"
formatCount(150000000) // "1.5亿"
```

### `formatDate(dateString, format)`
格式化日期为指定格式

**参数：**
- `dateString`: `string | Date | null | undefined` - 时间字符串或 Date 对象
- `format`: `'short' | 'long' | 'time'` - 格式类型（默认：'short'）

**返回：** `string` - 格式化后的日期

**示例：**
```typescript
formatDate('2024-01-01T10:30:00Z', 'short') // "1月1日 10:30"
formatDate('2024-01-01T10:30:00Z', 'long') // "2024年1月1日 10:30"
formatDate('2024-01-01T10:30:00Z', 'time') // "10:30"
```

### `formatFileSize(bytes)`
格式化文件大小

**参数：**
- `bytes`: `number | null | undefined` - 字节数

**返回：** `string` - 格式化后的文件大小

**示例：**
```typescript
formatFileSize(1024) // "1 KB"
formatFileSize(1048576) // "1 MB"
formatFileSize(null) // "0 B"
```

## ⚠️ 注意事项

### 1. 空值处理
所有格式化函数都内置了空值处理，无需在 Vue 组件中额外检查：

```vue
<!-- ✅ 正确 - 无需额外检查 -->
<p>{{ formatTime(user.last_active_at) }}</p>

<!-- ❌ 错误 - 不需要额外的空值检查 -->
<p>{{ user.last_active_at ? formatTime(user.last_active_at) : '-' }}</p>
```

### 2. 类型安全
所有函数都支持 TypeScript 类型检查，会在编译时发现类型错误。

### 3. 性能考虑
格式化函数都是纯函数，可以安全地在模板中使用，不会造成性能问题。

## 🚫 禁止事项

### 1. 禁止在 Vue 文件中实现格式化逻辑
```vue
<!-- ❌ 绝对禁止 -->
<script setup lang="ts">
const formatTime = (time: string) => {
  // 禁止在组件中实现格式化逻辑
}
</script>
```

### 2. 禁止硬编码格式化字符串
```vue
<!-- ❌ 禁止硬编码 -->
<p>{{ Math.floor(diffInSeconds / 60) }}分钟前</p>

<!-- ✅ 使用统一格式化 -->
<p>{{ formatTime(timestamp) }}</p>
```

### 3. 禁止重复实现相同功能
如果需要新的格式化功能，请在 `useFormatters.ts` 中添加，而不是在组件中实现。

## 🔧 扩展指南

如需添加新的格式化功能：

1. 在 `~/composables/useFormatters.ts` 中添加新函数
2. 在 `~/i18n/locales/zh-CN.json` 中添加相关翻译
3. 更新此文档
4. 添加单元测试

## 📝 示例项目

查看以下文件了解实际使用：
- `~/pages/moments/index.vue` - 时间格式化示例
- `~/pages/users/search.vue` - 距离格式化示例
- `~/components/UserCard.vue` - 综合格式化示例
