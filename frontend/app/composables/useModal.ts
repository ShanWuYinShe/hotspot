/**
 * 统一模态框管理
 * 使用 useOverlay 控制所有模态框的显示（Nuxt UI v4）
 */

import type { GroupMemberItem } from '~/types/api'

/** 举报目标类型 */
export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group'

/** 确认对话框选项 */
export interface ConfirmOptions {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  confirmColor?: 'primary' | 'error' | 'warning' | 'success'
}

/**
 * 统一模态框管理
 */
export function useModal() {
  const overlay = useOverlay()

  // ===== 确认对话框 =====
  const showConfirm = async (options: ConfirmOptions): Promise<boolean> => {
    const { default: ConfirmModal } = await import('~/components/modals/ConfirmModal.vue')
    const modal = overlay.create(ConfirmModal)
    const instance = modal.open(options)
    const result = await instance.result
    return result === true
  }

  // ===== 全局搜索 =====
  const showGlobalSearch = async (): Promise<void> => {
    const { default: GlobalSearchModal } = await import('~/components/modals/GlobalSearchModal.vue')
    const modal = overlay.create(GlobalSearchModal)
    const instance = modal.open()
    await instance.result
  }

  // ===== 认证相关 =====
  const showLoginPrompt = async (): Promise<void> => {
    const { default: LoginPromptModal } = await import('~/components/modals/LoginPromptModal.vue')
    const modal = overlay.create(LoginPromptModal)
    const instance = modal.open()
    await instance.result
  }

  const showWelcome = async (): Promise<void> => {
    const { default: WelcomeModal } = await import('~/components/modals/WelcomeModal.vue')
    const modal = overlay.create(WelcomeModal)
    const instance = modal.open()
    await instance.result
  }

  const showTempUserUpgrade = async (): Promise<void> => {
    const { default: GuestUserModal } = await import('~/components/modals/GuestUserModal.vue')
    const modal = overlay.create(GuestUserModal)
    const instance = modal.open()
    await instance.result
  }

  const showChangePassword = async (): Promise<void> => {
    const { default: ChangePasswordModal } = await import('~/components/modals/ChangePasswordModal.vue')
    const modal = overlay.create(ChangePasswordModal)
    const instance = modal.open()
    await instance.result
  }

  const showDeleteAccount = async (): Promise<void> => {
    const { default: DeleteAccountModal } = await import('~/components/modals/DeleteAccountModal.vue')
    const modal = overlay.create(DeleteAccountModal)
    const instance = modal.open()
    await instance.result
  }

  const showSwitchAccount = async (): Promise<void> => {
    const { default: SwitchAccountModal } = await import('~/components/modals/SwitchAccountModal.vue')
    const modal = overlay.create(SwitchAccountModal)
    const instance = modal.open()
    await instance.result
  }

  const showPinInput = async (isSetup?: boolean): Promise<string | null> => {
    const { default: PinInputModal } = await import('~/components/modals/PinInputModal.vue')
    const modal = overlay.create(PinInputModal)
    const instance = modal.open({ isSetup })
    const result = await instance.result
    return typeof result === 'string' ? result : null
  }

  // ===== 内容创建 =====
  const showCreateMoment = async (): Promise<string | null> => {
    const { default: CreateMomentModal } = await import('~/components/modals/CreateMomentModal.vue')
    const modal = overlay.create(CreateMomentModal)
    const instance = modal.open()
    const result = await instance.result
    return typeof result === 'string' ? result : null
  }

  const showCreateGroup = async (): Promise<string | null> => {
    const { default: CreateGroupModal } = await import('~/components/modals/CreateGroupModal.vue')
    const modal = overlay.create(CreateGroupModal)
    const instance = modal.open()
    const result = await instance.result
    return typeof result === 'string' ? result : null
  }

  // ===== 举报 =====
  const showReport = async (targetType: ReportTargetType, targetId: string): Promise<void> => {
    const { default: ReportModal } = await import('~/components/modals/ReportModal.vue')
    const modal = overlay.create(ReportModal)
    const instance = modal.open({ targetType, targetId })
    await instance.result
  }

  const showReportHistory = async (): Promise<void> => {
    const { default: ReportHistoryModal } = await import('~/components/modals/ReportHistoryModal.vue')
    const modal = overlay.create(ReportHistoryModal)
    const instance = modal.open()
    await instance.result
  }

  // ===== 群组管理 =====
  const showGroupSettings = async (groupId: string): Promise<void> => {
    const { default: GroupSettingsModal } = await import('~/components/modals/GroupSettingsModal.vue')
    const modal = overlay.create(GroupSettingsModal)
    const instance = modal.open({ groupId })
    await instance.result
  }

  const showInviteMembers = async (groupId: string): Promise<void> => {
    const { default: InviteMembersModal } = await import('~/components/modals/InviteMembersModal.vue')
    const modal = overlay.create(InviteMembersModal)
    const instance = modal.open({ groupId })
    await instance.result
  }

  const showTransferOwnership = async (groupId: string, members: GroupMemberItem[]): Promise<string | null> => {
    const { default: TransferOwnershipModal } = await import('~/components/modals/TransferOwnershipModal.vue')
    const modal = overlay.create(TransferOwnershipModal)
    const instance = modal.open({ groupId, members })
    const result = await instance.result
    return typeof result === 'string' ? result : null
  }

  const showGroupPassword = async (groupName?: string): Promise<string | null> => {
    const { default: GroupPasswordModal } = await import('~/components/modals/GroupPasswordModal.vue')
    const modal = overlay.create(GroupPasswordModal)
    const instance = modal.open({ groupName })
    const result = await instance.result
    return typeof result === 'string' ? result : null
  }

  // ===== 好友管理 =====
  const showBlockedUsers = async (): Promise<void> => {
    const { default: BlockedUsersModal } = await import('~/components/modals/BlockedUsersModal.vue')
    const modal = overlay.create(BlockedUsersModal)
    const instance = modal.open()
    await instance.result
  }

  // ===== 位置选择 =====
  const showLocationPicker = async (onSelect: (location: { latitude: number; longitude: number; locationName: string }) => void): Promise<void> => {
    const { default: LocationPickerModal } = await import('~/components/modals/LocationPickerModal.vue')
    const modal = overlay.create(LocationPickerModal)
    const instance = modal.open({ onSelect })
    await instance.result
  }

  return {
    // 基础
    showConfirm,
    showGlobalSearch,
    
    // 认证
    showLoginPrompt,
    showWelcome,
    showTempUserUpgrade,
    showChangePassword,
    showDeleteAccount,
    showSwitchAccount,
    showPinInput,
    
    // 内容
    showCreateMoment,
    showCreateGroup,
    showReport,
    showReportHistory,
    
    // 群组
    showGroupSettings,
    showInviteMembers,
    showTransferOwnership,
    showGroupPassword,
    
    // 好友
    showBlockedUsers,
    
    // 位置
    showLocationPicker
  }
}
