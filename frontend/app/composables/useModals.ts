/**
 * 使用 useOverlay 管理所有 Modal 组件的 composable
 */
export const useModals = () => {
  const overlay = useOverlay()

  // 懒加载 Modal 组件
  const CreateModalContent = defineAsyncComponent(() => import('~/components/CreateModalContent.vue'))
  const LoginPromptModalContent = defineAsyncComponent(() => import('~/components/LoginPromptModalContent.vue'))
  const WelcomeModalContent = defineAsyncComponent(() => import('~/components/WelcomeModalContent.vue'))
  const CreateGroupModalContent = defineAsyncComponent(() => import('~/components/CreateGroupModalContent.vue'))
  const ConfirmModalContent = defineAsyncComponent(() => import('~/components/ConfirmModalContent.vue'))
  const NicknameEditModalContent = defineAsyncComponent(() => import('~/components/NicknameEditModalContent.vue'))
  const AvatarUploadModalContent = defineAsyncComponent(() => import('~/components/AvatarUploadModalContent.vue'))
  const MobileDrawerContent = defineAsyncComponent(() => import('~/components/MobileDrawerContent.vue'))
  const DetailSlideoverContent = defineAsyncComponent(() => import('~/components/DetailSlideoverContent.vue'))
  const TempUserModalContent = defineAsyncComponent(() => import('~/components/TempUserModalContent.vue'))
  const SwitchAccountModalContent = defineAsyncComponent(() => import('~/components/SwitchAccountModalContent.vue'))
  const SearchModalContent = defineAsyncComponent(() => import('~/components/SearchModalContent.vue'))
  const FriendRemarkModalContent = defineAsyncComponent(() => import('~/components/FriendRemarkModalContent.vue'))

  // 创建 Modal 实例
  const createModal = overlay.create(CreateModalContent)
  const loginPromptModal = overlay.create(LoginPromptModalContent)
  const welcomeModal = overlay.create(WelcomeModalContent)
  const createGroupModal = overlay.create(CreateGroupModalContent)
  const confirmModal = overlay.create(ConfirmModalContent)
  const nicknameEditModal = overlay.create(NicknameEditModalContent)
  const avatarUploadModal = overlay.create(AvatarUploadModalContent)
  const mobileDrawer = overlay.create(MobileDrawerContent)
  const detailSlideover = overlay.create(DetailSlideoverContent)
  const tempUserModal = overlay.create(TempUserModalContent)
  const switchAccountModal = overlay.create(SwitchAccountModalContent)
  const searchModal = overlay.create(SearchModalContent)
  const friendRemarkModal = overlay.create(FriendRemarkModalContent)

  // 打开创建内容 Modal
  const openCreateModal = () => {
    return createModal.open()
  }

  // 打开登录提示 Modal
  const openLoginPromptModal = () => {
    const instance = loginPromptModal.open()
    
    // 设置事件处理器
    loginPromptModal.patch({
      'onTemp-experience': async () => {
        // 处理临时体验 - 自动创建临时用户
        try {
          const authStore = useAuthStore()
          const toast = useSmartToast()
          
          if (import.meta.dev) {
            // console.log('用户选择临时体验，正在创建临时用户...')
          }
          
          await authStore.createTempUser()
          
          toast.success({
            title: '临时体验已开启',
            description: '您现在可以使用所有功能了！'
          })
          
          // 跳转到首页
          await navigateTo('/')
        } catch (error: any) {
          const toast = useSmartToast()
          toast.error({
            title: '临时体验开启失败',
            description: error?.message || '请稍后重试'
          })
        }
      },
      onLogin: () => {
        // 处理登录 - 跳转到认证页面
        navigateTo('/auth')
      },
      onSkip: () => {
        // 处理跳过
        if (import.meta.dev) {
          // console.log('用户选择跳过')
        }
      },
      'onNever-show': () => {
        // 处理不再显示
        if (import.meta.dev) {
          // console.log('用户选择不再显示')
        }
      }
    })
    
    return instance
  }

  // 打开欢迎 Modal
  const openWelcomeModal = () => {
    const instance = welcomeModal.open()
    
    // 设置事件处理器
    welcomeModal.patch({
      'onTemp-experience': async () => {
        // 处理临时体验 - 自动创建临时用户
        try {
          const authStore = useAuthStore()
          const toast = useSmartToast()
          
          if (import.meta.dev) {
            // console.log('用户选择临时体验，正在创建临时用户...')
          }
          
          await authStore.createTempUser()
          
          toast.success({
            title: '欢迎来到 HotSpot！',
            description: '临时体验已开启，开始探索吧！'
          })
          
          // 跳转到首页
          await navigateTo('/')
        } catch (error: any) {
          const toast = useSmartToast()
          toast.error({
            title: '临时体验开启失败',
            description: error?.message || '请稍后重试'
          })
        }
      },
      onRegister: () => {
        // 处理注册 - 跳转到认证页面
        navigateTo('/auth')
      },
      onLogin: () => {
        // 处理登录 - 跳转到认证页面
        navigateTo('/auth')
      },
      onSkip: () => {
        // 处理跳过
        if (import.meta.dev) {
          // console.log('用户选择跳过')
        }
      }
    })
    
    return instance
  }

  // 打开创建群组 Modal
  const openCreateGroupModal = (options?: {
    onSuccess?: (group: any) => void | Promise<void>;
    onCancel?: () => void;
  }) => {
    const instance = createGroupModal.open()
    
    // 设置事件处理器
    createGroupModal.patch({
      onSuccess: async (group: any) => {
        // 处理创建成功
        // console.log('群组创建成功:', group)
        if (options?.onSuccess) {
          await options.onSuccess(group)
        }
      },
      onCancel: () => {
        // 处理取消
        // console.log('用户取消创建群组')
        if (options?.onCancel) {
          options.onCancel()
        }
      }
    })
    
    return instance
  }

  // 打开确认 Modal
  const openConfirmModal = (options: {
    title?: string;
    message?: string;
    confirmText?: string;
    cancelText?: string;
    confirmColor?: 'primary' | 'error' | 'warning' | 'success';
    onConfirm?: () => void | Promise<void>;
    onCancel?: () => void;
  }) => {
    const instance = confirmModal.open({
      title: options.title,
      message: options.message,
      confirmText: options.confirmText,
      cancelText: options.cancelText,
      confirmColor: options.confirmColor
    })
    
    // 设置事件处理器
    confirmModal.patch({
      onConfirm: async () => {
        if (options.onConfirm) {
          await options.onConfirm()
        }
      },
      onCancel: () => {
        if (options.onCancel) {
          options.onCancel()
        }
      },
      onClose: () => {
        // Modal 关闭时的清理逻辑
        if (import.meta.dev) {
          // console.log('确认 Modal 已关闭')
        }
      }
    })
    
    return instance
  }

  // 打开昵称编辑 Modal
  const openNicknameEditModal = (options: {
    currentNickname?: string;
    onSuccess?: (nickname: string) => void | Promise<void>;
    onCancel?: () => void;
  }) => {
    const instance = nicknameEditModal.open({
      currentNickname: options.currentNickname
    })
    
    // 设置事件处理器
    nicknameEditModal.patch({
      onSuccess: async (nickname: string) => {
        if (options.onSuccess) {
          await options.onSuccess(nickname)
        }
      },
      onCancel: () => {
        if (options.onCancel) {
          options.onCancel()
        }
      }
    })
    
    return instance
  }

  // 打开头像上传 Modal
  const openAvatarUploadModal = (options: {
    user?: any;
    onUploaded?: (data: any) => void | Promise<void>;
  }) => {
    const instance = avatarUploadModal.open({
      user: options.user
    })
    
    // 设置事件处理器
    avatarUploadModal.patch({
      onUploaded: async (data: any) => {
        if (options.onUploaded) {
          await options.onUploaded(data)
        }
      }
    })
    
    return instance
  }

  // 打开移动端抽屉
  const openMobileDrawer = () => {
    return mobileDrawer.open()
  }

  // 打开详情侧边栏
  const openDetailSlideover = (options: {
    type?: 'user' | 'group' | 'moment';
    data?: any;
    side?: 'left' | 'right';
    onAction?: (action: string, data?: any) => void;
  }) => {
    const instance = detailSlideover.open({
      type: options.type,
      data: options.data,
      side: options.side
    })
    
    // 设置事件处理器
    detailSlideover.patch({
      onAction: (action: string, data?: any) => {
        if (options.onAction) {
          options.onAction(action, data)
        }
      }
    })
    
    return instance
  }

  // 打开临时用户创建 Modal
  const openTempUserModal = (options: {
    onCreateTemp?: (nickname?: string) => Promise<void>
  }) => {
    const instance = tempUserModal.open({
      onCreateTemp: options.onCreateTemp
    })
    return instance
  }

  // 打开账户切换确认 Modal
  const openSwitchAccountModal = (options: {
    account?: any
    onConfirm?: () => Promise<void>
  }) => {
    const instance = switchAccountModal.open({
      account: options.account,
      onConfirm: options.onConfirm
    })
    return instance
  }

  // 打开搜索 Modal
  const openSearchModal = (options?: {
    onCreateMoment?: () => void;
  }) => {
    const instance = searchModal.open()
    
    // 设置事件处理器
    searchModal.patch({
      'onCreate-moment': () => {
        // 处理创建动态事件
        if (options?.onCreateMoment) {
          options.onCreateMoment()
        }
      },
      onClose: () => {
        // 搜索 Modal 关闭时的处理逻辑
        if (import.meta.dev) {
          // console.log('搜索 Modal 已关闭')
        }
      }
    })
    
    return instance
  }

  // 打开好友备注编辑 Modal
  const openFriendRemarkModal = (options: {
    friend?: any
    onSave?: (remarkText: string) => Promise<void>
  }) => {
    const instance = friendRemarkModal.open({
      friend: options.friend,
      onSave: options.onSave
    })
    return instance
  }

  return {
    // Modal 实例
    createModal,
    loginPromptModal,
    welcomeModal,
    createGroupModal,
    confirmModal,
    nicknameEditModal,
    avatarUploadModal,
    mobileDrawer,
    detailSlideover,
    
    // 打开方法
    openCreateModal,
    openLoginPromptModal,
    openWelcomeModal,
    openCreateGroupModal,
    openConfirmModal,
    openNicknameEditModal,
    openAvatarUploadModal,
    openMobileDrawer,
    openDetailSlideover,
    openTempUserModal,
    openSwitchAccountModal,
    openSearchModal,
    openFriendRemarkModal
  }
}
