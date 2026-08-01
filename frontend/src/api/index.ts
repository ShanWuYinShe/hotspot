export { Endpoints } from '@/api/endpoints';
export {
  setApiBaseUrl,
  login, register, guest, logout, checkAuthStatus,
  requestPasswordReset, resetPassword,
  getAccount, updateAccount, getSettings, updateSettings, updateLocation,
  follow, block, getProfile, getUserDetail, batchGet,
  listFriends, listFollowing, listFollowers,
  getFeedFollowing, getFeedNearby, getTrending, getNearbyUsers, getNearbyGroups,
  searchMoments, getMomentDetail, createMoment, listMyMoments, getFavoritePosts,
  searchGroups, getGroupDetail, createGroup, listMyGroups,
  searchUsers, searchSuggest,
  getUnreadCount, listNotifications, markRead,
  getClientConfig, bootstrap,
  createReport,
  getShare, createShare,
  getPrivateChats, getPrivateMessages, sendPrivateMessage,
  getGroupMessages, sendGroupMessage, markPrivateRead, markGroupRead,
  presignUpload, completeUpload,
  setReaction, listReactions,
  createComment,
  favoriteMoment,
  getGroupMembers, getGroupAnnouncements,
  listUserMoments, listUserReactedMoments,
  listComments,
  getMapCanvas,
} from '@/api/client';
// P2: useApi 以 createApiState 形式重建于 src/lib/useApi.svelte.ts
export type { ApiResponse, PaginatedData } from '@/api/client';
