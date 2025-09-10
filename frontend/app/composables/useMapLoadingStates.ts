/**
 * 地图页面加载状态管理
 * 统一管理各种加载状态，避免重复代码
 */
export const useMapLoadingStates = () => {
  // 主要加载状态
  const isLoading = ref(false)
  
  // 数据加载状态
  const loadingUsers = ref(false)
  const loadingMoreUsers = ref(false)
  const loadingGroups = ref(false)
  const loadingMoreGroups = ref(false)
  const loadingMoments = ref(false)
  const loadingMoreMoments = ref(false)

  // 计算属性：当前标签页的加载状态
  const getCurrentTabLoading = (currentTab: Ref<'users' | 'groups' | 'moments'>) => {
    return computed(() => {
      switch (currentTab.value) {
        case 'users':
          return loadingUsers.value
        case 'groups':
          return loadingGroups.value
        case 'moments':
          return loadingMoments.value
        default:
          return false
      }
    })
  }

  // 设置加载状态的辅助函数
  const setTabLoading = (tab: 'users' | 'groups' | 'moments', loading: boolean) => {
    switch (tab) {
      case 'users':
        loadingUsers.value = loading
        break
      case 'groups':
        loadingGroups.value = loading
        break
      case 'moments':
        loadingMoments.value = loading
        break
    }
  }

  const setTabLoadingMore = (tab: 'users' | 'groups' | 'moments', loading: boolean) => {
    switch (tab) {
      case 'users':
        loadingMoreUsers.value = loading
        break
      case 'groups':
        loadingMoreGroups.value = loading
        break
      case 'moments':
        loadingMoreMoments.value = loading
        break
    }
  }

  return {
    // 状态
    isLoading,
    loadingUsers,
    loadingMoreUsers,
    loadingGroups,
    loadingMoreGroups,
    loadingMoments,
    loadingMoreMoments,
    
    // 计算属性工厂
    getCurrentTabLoading,
    
    // 辅助函数
    setTabLoading,
    setTabLoadingMore
  }
}
