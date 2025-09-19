/**
 * 地图页面加载状态管理
 */
export const useMapLoadingStates = () => {
  const isLoading = ref(false)
  const loadingUsers = ref(false)
  const loadingGroups = ref(false)
  const loadingMoments = ref(false)

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

  return {
    isLoading,
    loadingUsers,
    loadingGroups,
    loadingMoments,
    setTabLoading
  }
}