/**
 * 统一模态框管理 - 简洁而完整
 * 所有模态框功能都在一个文件中管理
 */

/**
 * 模态框选项接口
 */
export interface ModalOptions {
  title?: string
  message?: string
  confirmText?: string
  cancelText?: string
  confirmColor?: 'primary' | 'error' | 'warning'
}

export interface ConfirmOptions extends ModalOptions {
  onConfirm?: () => void | Promise<void>
  onCancel?: () => void
}

/**
 * 统一模态框管理
 */
export function useModal() {
  const overlay = useOverlay()

  // ===== 确认对话框 =====
  const showConfirm = async (options: ConfirmOptions): Promise<boolean> => {
    return new Promise(async (resolve) => {
      const ConfirmModal = await import('~/components/modals/ConfirmModal.vue')
      
      const modal = overlay.create(ConfirmModal)

      modal.open()
      
      // 监听确认和取消事件
      // 这里需要根据 ConfirmModal 组件的实际实现来调整
      // 暂时返回 false
      resolve(false)
    })
  }

  // ===== 认证相关模态框 =====
  const showLoginPrompt = async (): Promise<boolean> => {
    return new Promise(async (resolve) => {
      const LoginPromptModal = await import('~/components/modals/LoginPromptModal.vue')
      
      const modal = overlay.create(LoginPromptModal)

      modal.open()
      
      // 监听登录事件
      // 这里需要根据 LoginPromptModal 组件的实际实现来调整
      // 暂时返回 false
      resolve(false)
    })
  }

  const showWelcome = async (): Promise<void> => {
    return new Promise(async (resolve) => {
      const WelcomeModal = await import('~/components/modals/WelcomeModal.vue')
      
      const modal = overlay.create(WelcomeModal)

      modal.open()
      
      // 监听关闭事件
      // 这里需要根据 WelcomeModal 组件的实际实现来调整
      // 暂时直接 resolve
      resolve()
    })
  }

  const showTempUserUpgrade = async (): Promise<boolean> => {
    return new Promise(async (resolve) => {
      const GuestUserModal = await import('~/components/modals/GuestUserModal.vue')
      
      const modal = overlay.create(GuestUserModal)

      modal.open()
      
      // 监听升级事件
      // 这里需要根据 GuestUserModal 组件的实际实现来调整
      // 暂时返回 false
      resolve(false)
    })
  }

  const showNicknameEdit = async (initialNickname?: string): Promise<string | null> => {
    return new Promise(async (resolve) => {
      const NicknameEditModal = await import('~/components/modals/NicknameEditModal.vue')
      
      const modal = overlay.create(NicknameEditModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 NicknameEditModal 组件的实际实现来调整
      // 暂时返回初始昵称
      resolve(initialNickname || null)
    })
  }

  const showChangePassword = async (): Promise<boolean | null> => {
    return new Promise(async (resolve) => {
      const ChangePasswordModal = await import('~/components/modals/ChangePasswordModal.vue')
      
      const modal = overlay.create(ChangePasswordModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 ChangePasswordModal 组件的实际实现来调整
      // 暂时返回 null
      resolve(null)
    })
  }

  const showDeleteAccount = async (): Promise<boolean | null> => {
    return new Promise(async (resolve) => {
      const DeleteAccountModal = await import('~/components/modals/DeleteAccountModal.vue')
      
      const modal = overlay.create(DeleteAccountModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 DeleteAccountModal 组件的实际实现来调整
      // 暂时返回 null
      resolve(null)
    })
  }

  const showSwitchAccount = async (): Promise<boolean | null> => {
    return new Promise(async (resolve) => {
      const SwitchAccountModal = await import('~/components/modals/SwitchAccountModal.vue')
      
      const modal = overlay.create(SwitchAccountModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 SwitchAccountModal 组件的实际实现来调整
      // 暂时返回 null
      resolve(null)
    })
  }

  const showPinInput = async (options: ModalOptions = {}): Promise<string | null> => {
    return new Promise(async (resolve) => {
      const PinInputModal = await import('~/components/modals/PinInputModal.vue')
      
      const modal = overlay.create(PinInputModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 PinInputModal 组件的实际实现来调整
      // 暂时返回 null
      resolve(null)
    })
  }

  // ===== 内容相关模态框 =====
  const showCreateMoment = async (options?: {
    onSuccess?: (momentId: string) => void
  }): Promise<void> => {
    return new Promise(async (resolve) => {
      const CreateMomentModal = await import('~/components/modals/CreateMomentModal.vue')
      
      const modal = overlay.create(CreateMomentModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 CreateMomentModal 组件的实际实现来调整
      // 暂时直接 resolve
      resolve()
    })
  }

  const showCreateGroup = async (options?: {
    onSuccess?: (groupId: string) => void
  }): Promise<void> => {
    return new Promise(async (resolve) => {
      const CreateGroupModal = await import('~/components/modals/CreateGroupModal.vue')
      
      const modal = overlay.create(CreateGroupModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 CreateGroupModal 组件的实际实现来调整
      // 暂时直接 resolve
      resolve()
    })
  }

  const showReport = async (options: {
    targetType: string
    targetId: string
    onSuccess?: () => void
  }): Promise<void> => {
    return new Promise(async (resolve) => {
      const ReportModal = await import('~/components/modals/ReportModal.vue')
      
      const modal = overlay.create(ReportModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 ReportModal 组件的实际实现来调整
      // 暂时直接 resolve
      resolve()
    })
  }

  const showReportHistory = async (): Promise<void> => {
    return new Promise(async (resolve) => {
      const ReportHistoryModal = await import('~/components/modals/ReportHistoryModal.vue')
      
      const modal = overlay.create(ReportHistoryModal)

      modal.open()
      
      // 监听关闭事件
      // 这里需要根据 ReportHistoryModal 组件的实际实现来调整
      // 暂时直接 resolve
      resolve()
    })
  }

  // ===== 群组相关模态框 =====
  const showGroupSettings = async (groupId: string, onSuccess?: () => void): Promise<void> => {
    return new Promise(async (resolve) => {
      const GroupSettingsModal = await import('~/components/modals/GroupSettingsModal.vue')
      
      const modal = overlay.create(GroupSettingsModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 GroupSettingsModal 组件的实际实现来调整
      // 暂时直接 resolve
      resolve()
    })
  }

  const showInviteMembers = async (groupId: string, onSuccess?: () => void): Promise<void> => {
    return new Promise(async (resolve) => {
      const InviteMembersModal = await import('~/components/modals/InviteMembersModal.vue')
      
      const modal = overlay.create(InviteMembersModal)

      modal.open()
      
      // 暂时直接 resolve
      resolve()
    })
  }

  const showTransferOwnership = async (groupId: string, members: any[], onSuccess?: (newOwnerId: string) => void): Promise<void> => {
    return new Promise(async (resolve) => {
      const TransferOwnershipModal = await import('~/components/modals/TransferOwnershipModal.vue')
      
      const modal = overlay.create(TransferOwnershipModal)

      modal.open()
      
      // 暂时直接 resolve
      resolve()
    })
  }

  // ===== 好友相关模态框 =====
  const showFriendRequests = async (): Promise<void> => {
    return new Promise(async (resolve) => {
      const FriendRequestsModal = await import('~/components/modals/FriendRequestsModal.vue')
      
      const modal = overlay.create(FriendRequestsModal)

      modal.open()
      
      // 监听关闭事件
      // 这里需要根据 FriendRequestsModal 组件的实际实现来调整
      // 暂时直接 resolve
      resolve()
    })
  }

  const showBlockedUsers = async (): Promise<void> => {
    return new Promise(async (resolve) => {
      const BlockedUsersModal = await import('~/components/modals/BlockedUsersModal.vue')
      
      const modal = overlay.create(BlockedUsersModal)

      modal.open()
      
      // 监听关闭事件
      // 这里需要根据 BlockedUsersModal 组件的实际实现来调整
      // 暂时直接 resolve
      resolve()
    })
  }

  const showFriendRemark = async (friendId: string, currentRemark?: string, onSuccess?: (newRemark: string) => void): Promise<string | null> => {
    return new Promise(async (resolve) => {
      const FriendRemarkModal = await import('~/components/modals/FriendRemarkModal.vue')
      
      const modal = overlay.create(FriendRemarkModal)

      modal.open()
      
      // 监听成功事件
      // 这里需要根据 FriendRemarkModal 组件的实际实现来调整
      // 暂时返回当前备注
      resolve(currentRemark || null)
    })
  }

  // ===== 命令面板 =====
  const commandPaletteOpen = ref(false)
  
  const showCommandPalette = async (options?: {
    onCreateMoment?: () => void
  }): Promise<void> => {
    commandPaletteOpen.value = true
  }

  // ===== Slideover =====
  const showEntityPreview = async (type: 'user' | 'group' | 'moment', id: string, onClose?: () => void): Promise<void> => {
    return new Promise(async (resolve) => {
      const EntityPreview = await import('~/components/layout/EntityPreview.vue')
      
      const modal = overlay.create(EntityPreview)

      modal.open()
      
      // 监听关闭事件
      // 这里需要根据 EntityPreview 组件的实际实现来调整
      // 暂时直接 resolve
      resolve()
    })
  }

  return {
    // 基础功能
    showConfirm,
    
    // 认证相关
    showLoginPrompt,
    showWelcome,
    showTempUserUpgrade,
    showNicknameEdit,
    showChangePassword,
    showDeleteAccount,
    showSwitchAccount,
    showPinInput,
    
    // 内容相关
    showCreateMoment,
    showCreateGroup,
    showReport,
    showReportHistory,
    
    // 群组相关
    showGroupSettings,
    showInviteMembers,
    showTransferOwnership,
    
    // 好友相关
    showFriendRequests,
    showBlockedUsers,
    showFriendRemark,
    
    // 命令面板
    showCommandPalette,
    commandPaletteOpen,
    
    // Slideover
    showEntityPreview,
    
    // 兼容旧的命名
    showConfirmModal: showConfirm,
    showLoginPromptModal: showLoginPrompt,
    showWelcomeModal: showWelcome,
    showTempUserModal: showTempUserUpgrade,
    showNicknameEditModal: showNicknameEdit,
    showChangePasswordModal: showChangePassword,
    showDeleteAccountModal: showDeleteAccount,
    showSwitchAccountModal: showSwitchAccount,
    showPinInputModal: showPinInput,
    showCreateMomentModal: showCreateMoment,
    showCreateGroupModal: showCreateGroup,
    showReportModal: showReport,
    showReportHistoryModal: showReportHistory,
    showGroupSettingsModal: showGroupSettings,
    showInviteMembersModal: showInviteMembers,
    showTransferOwnershipModal: showTransferOwnership,
    showFriendRequestsModal: showFriendRequests,
    showBlockedUsersModal: showBlockedUsers,
    showFriendRemarkModal: showFriendRemark,
    showDetailSlideover: showEntityPreview
  }
}
