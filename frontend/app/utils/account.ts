import { useAdminStore } from '~/stores/admin.store'
import { useGroupsStore } from '~/stores/groups.store'
import { useMessagesStore } from '~/stores/messages.store'
import { useMomentsStore } from '~/stores/moments.store'
import { useSettingsStore } from '~/stores/settings.store'

/**
 * 多账号切换/锁定/退出时的“强隔离”清理：
 * - 清空所有会话相关运行时状态（消息、feed、权限等）
 * - 清理跨账号不应复用的持久化缓存（避免显示旧数据）
 *
 * 说明：这不是“安全防护”（无法防止同浏览器下的恶意脚本/DevTools 读取存储），
 * 但能保证产品层面“切换账号不会看到上一个账号的数据”。
 */
export function resetAppForAccountChange() {
  const messagesStore = useMessagesStore()
  const momentsStore = useMomentsStore()
  const groupsStore = useGroupsStore()
  const settingsStore = useSettingsStore()
  const adminStore = useAdminStore()

  // 运行时状态清空
  messagesStore.clearAll()
  settingsStore.resetUserScoped()
  adminStore.setAccess(false)

  // 用户级缓存清空（避免跨账号看到旧列表/草稿/点赞状态）
  groupsStore.clearUserCache()
  momentsStore.clearUserCache()

  // 兼容清理：旧版本 moments-store 可能仍残留
  if (import.meta.client) {
    try { localStorage.removeItem('moments-store') } catch { /* ignore */ }
  }
}

