# WebSocket 客户端实现

严格遵循 `@spec/api_spec_websocket.mdc` 规范。

## 文件结构

```
websocket/
├── index.ts              - 模块入口
├── types.ts              - TypeScript 类型定义（与规范完全对应）
├── store.svelte.ts       - 连接管理（鉴权、心跳、重连）
├── handlers.ts           - 事件处理器 + 订阅管理
└── README.md             - 本文档
```

## 快速开始

### 1. 初始化 WebSocket（在根 Layout 中）

```typescript
import { initializeWebSocket } from '$lib/stores/websocket-lifecycle.svelte';

onMount(() => {
  initializeWebSocket(); // 自动监听登录状态，登录后自动连接
});
```

### 2. 在聊天页面中使用

```typescript
import { websocketStore, subscriptionManager, eventHandlers } from '$lib/websocket';

onMount(() => {
  // 订阅对方用户状态
  subscriptionManager.subscribe('user', [targetUserId]);
  
  // 监听私聊消息
  const unsubscribe = eventHandlers.onPrivateMessage((payload) => {
    if (payload.target_user_id === targetUserId) {
      // 添加到消息列表
      messages = [...messages, convertToMessageItem(payload)];
    }
  });
  
  // 组件卸载时清理
  onDestroy(() => {
    unsubscribe();
    subscriptionManager.unsubscribe('user', [targetUserId]);
  });
});
```

### 3. 发送正在输入状态

```typescript
function handleInput() {
  websocketStore.send({
    type: 'PrivateTyping',
    payload: {
      target_user_id: targetUserId,
      is_typing: true
    }
  });
}
```

### 4. 监听通知事件

```typescript
import { notificationStore } from '$lib/stores/notification.svelte';

// notificationStore 自动监听所有通知事件
// 访问未读通知
console.log(notificationStore.unreadCount);
console.log(notificationStore.unreadNotifications);
```

## 核心特性

### 自动重连

- 指数退避：1s, 2s, 4s, 8s, 16s, 30s（最大）
- 最大重连次数：5 次
- 重连成功后自动恢复订阅

### 心跳机制

- Ping 间隔：30 秒
- Pong 超时：10 秒
- 服务端超时：90 秒

### 订阅管理

- 支持增量订阅（`subscribe`）
- 支持取消订阅（`unsubscribe`）
- 支持替换式订阅（`setSubscriptions`）
- 重连后自动恢复订阅

### 事件处理

- 类型安全的事件监听
- 支持多个监听器
- 自动清理机制

## API 参考

### websocketStore

#### 方法

- `connect(url?)` - 连接 WebSocket
- `disconnect()` - 断开连接
- `send(event)` - 发送事件
- `on(eventType, handler)` - 监听事件
- `off(eventType, handler?)` - 移除监听

#### 状态

- `connectionState` - 连接状态
- `connectionId` - 连接 ID
- `lastError` - 最后的错误信息

### subscriptionManager

#### 方法

- `subscribe(type, ids)` - 订阅（增量）
- `unsubscribe(type, ids)` - 取消订阅
- `setSubscriptions(type, ids)` - 设置订阅（替换）
- `getSubscriptions(type)` - 获取当前订阅
- `isSubscribed(type, id)` - 检查是否已订阅

### eventHandlers

#### 方法

- `onPrivateMessage(handler)` - 监听私聊消息
- `onGroupMessage(handler)` - 监听群聊消息
- `onNotification(handler)` - 监听通知
- `onUserOnline(handler)` - 监听用户上线
- `onUserOffline(handler)` - 监听用户离线

## 注意事项

### 1. 消息发送流程

```typescript
// ❌ 错误：本地立即添加消息
const response = await sendMessage(text);
if (isSuccess(response)) {
  messages = [...messages, localMessage]; // 不符合规范
}

// ✅ 正确：等待 WebSocket 推送
const response = await sendMessage(text);
if (isSuccess(response)) {
  // 只检查发送是否成功，不添加本地消息
  // 等待 WebSocket 推送 PrivateMessageNew 事件
}
```

### 2. 订阅管理最佳实践

```typescript
// ✅ 推荐：在组件中管理订阅
onMount(() => {
  subscriptionManager.subscribe('group', [groupId]);
  
  onDestroy(() => {
    subscriptionManager.unsubscribe('group', [groupId]);
  });
});

// ❌ 不推荐：全局订阅所有（会导致性能问题）
subscriptionManager.subscribe('user', allUserIds); // 太多了
```

### 3. 错误处理

```typescript
// 监听连接错误
websocketStore.on('Error', (payload: any) => {
  console.error('WebSocket 错误:', payload);
  
  if (payload.code === 20001) {
    // 鉴权失败，重新登录
    authStore.logout();
  }
});
```

## 规范符合性

| 规范要求 | 实现状态 |
|---------|---------|
| 5秒鉴权超时 | ✅ 已实现 |
| 30秒心跳间隔 | ✅ 已实现 |
| 10秒 Pong 超时 | ✅ 已实现 |
| 指数退避重连 | ✅ 已实现 |
| 最大重连 5 次 | ✅ 已实现 |
| 重连后恢复订阅 | ✅ 已实现 |
| 所有事件类型 | ✅ 已实现 |
| 订阅管理 | ✅ 已实现 |
| 正在输入状态 | ✅ 已实现 |
