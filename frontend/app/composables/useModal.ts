/**
 * 统一模态框管理
 * 使用 useOverlay 控制所有模态框的显示（Nuxt UI v4）
 */

import type { Component } from 'vue'
import type { GroupMemberItem, MomentSearchItem } from '~/types/api'

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

  function openModal<TResult = void>(importer: () => Promise<{ default: Component }>): Promise<TResult>
  function openModal<TOptions, TResult = void>(
    importer: () => Promise<{ default: Component }>,
    options: TOptions
  ): Promise<TResult>
  async function openModal<TOptions, TResult = void>(
    importer: () => Promise<{ default: Component }>,
    options?: TOptions
  ): Promise<TResult> {
    const { default: Component } = await importer()
    const modal = overlay.create(Component)
    const instance = options === undefined ? modal.open() : modal.open(options)
    return await instance.result as TResult
  }

  // ===== 确认对话框 =====
  const showConfirm = async (options: ConfirmOptions): Promise<boolean> => {
    const result = await openModal<ConfirmOptions, unknown>(
      () => import('~/components/modals/ConfirmModal.vue'),
      options
    )
    return result === true
  }

  // ===== 全局搜索 =====
  const showGlobalSearch = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/GlobalSearchModal.vue'))
  }

  // ===== 认证相关 =====
  const showLoginPrompt = async (options?: { actionName?: string; redirectPath?: string }): Promise<void> => {
    if (options) {
      await openModal<{ actionName?: string; redirectPath?: string }>(
        () => import('~/components/modals/LoginPromptModal.vue'),
        options
      )
    } else {
      await openModal(() => import('~/components/modals/LoginPromptModal.vue'))
    }
  }

  const showWelcome = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/WelcomeModal.vue'))
  }

  const showGuestUserUpgrade = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/GuestUserModal.vue'))
  }

  const showChangePassword = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/ChangePasswordModal.vue'))
  }

  const showResetPassword = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/ResetPasswordModal.vue'))
  }

  const showDeleteAccount = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/DeleteAccountModal.vue'))
  }

  const showEditProfile = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/EditProfileModal.vue'))
  }

  const showSwitchAccount = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/SwitchAccountModal.vue'))
  }

  const showPinInput = async (isSetup?: boolean): Promise<string | null> => {
    const result = await openModal<{ isSetup?: boolean }, unknown>(
      () => import('~/components/modals/PinInputModal.vue'),
      { isSetup }
    )
    return typeof result === 'string' ? result : null
  }

  // ===== 内容创建 =====
  const showCreateMoment = async (): Promise<string | null> => {
    const result = await openModal<unknown>(() => import('~/components/modals/CreateMomentModal.vue'))
    return typeof result === 'string' ? result : null
  }

  const showEditMoment = async (moment: Pick<MomentSearchItem, 'id' | 'description' | 'tags' | 'images' | 'geo'>): Promise<string | null> => {
    const result = await openModal<{ mode: 'edit'; moment: Pick<MomentSearchItem, 'id' | 'description' | 'tags' | 'images' | 'geo'> }, unknown>(
      () => import('~/components/modals/CreateMomentModal.vue'),
      { mode: 'edit', moment }
    )
    return typeof result === 'string' ? result : null
  }

  const showCreateGroup = async (): Promise<string | null> => {
    const result = await openModal<unknown>(() => import('~/components/modals/CreateGroupModal.vue'))
    return typeof result === 'string' ? result : null
  }

  // ===== 举报 =====
  const showReport = async (targetType: ReportTargetType, targetId: string): Promise<void> => {
    await openModal<{ targetType: ReportTargetType; targetId: string }>(
      () => import('~/components/modals/ReportModal.vue'),
      { targetType, targetId }
    )
  }

  const showReportHistory = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/ReportHistoryModal.vue'))
  }

  // ===== 群组管理 =====
  const showGroupSettings = async (groupId: string): Promise<void> => {
    await openModal<{ groupId: string }>(() => import('~/components/modals/GroupSettingsModal.vue'), { groupId })
  }

  const showInviteMembers = async (groupId: string): Promise<void> => {
    await openModal<{ groupId: string }>(() => import('~/components/modals/InviteMembersModal.vue'), { groupId })
  }

  const showTransferOwnership = async (groupId: string, members: GroupMemberItem[]): Promise<string | null> => {
    const result = await openModal<{ groupId: string; members: GroupMemberItem[] }, unknown>(
      () => import('~/components/modals/TransferOwnershipModal.vue'),
      { groupId, members }
    )
    return typeof result === 'string' ? result : null
  }

  const showGroupPassword = async (groupName?: string): Promise<string | null> => {
    const result = await openModal<{ groupName?: string }, unknown>(
      () => import('~/components/modals/GroupPasswordModal.vue'),
      { groupName }
    )
    return typeof result === 'string' ? result : null
  }

  // ===== 好友管理 =====
  const showBlockedUsers = async (): Promise<void> => {
    await openModal(() => import('~/components/modals/BlockedUsersModal.vue'))
  }

  // ===== 位置选择 =====
  const showLocationPicker = async (onSelect: (location: { latitude: number; longitude: number; locationName: string }) => void): Promise<void> => {
    await openModal<{ onSelect: (location: { latitude: number; longitude: number; locationName: string }) => void }>(
      () => import('~/components/modals/LocationPickerModal.vue'),
      { onSelect }
    )
  }

  return {
    // 基础
    showConfirm,
    showGlobalSearch,
    
    // 认证
    showLoginPrompt,
    showWelcome,
    showGuestUserUpgrade,
    showChangePassword,
    showResetPassword,
    showDeleteAccount,
    showEditProfile,
    showSwitchAccount,
    showPinInput,
    
    // 内容
    showCreateMoment,
    showEditMoment,
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
