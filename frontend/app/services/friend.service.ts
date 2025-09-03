/**
 * 好友服务 - Composition API 版本
 * 处理好友关系相关的业务逻辑
 */

import { FRIEND_ENDPOINTS } from "~/config/api";

export const useFriendService = () => {
  const api = useApi();

  // 获取好友列表
  const getFriendsList = async (cursor?: number) => {
    return await api.post(`${FRIEND_ENDPOINTS.BASE}${FRIEND_ENDPOINTS.LIST}`, {
      cursor,
      limit: 20
    });
  };

  // 发送好友请求
  const sendFriendRequest = async (userId: number) => {
    return await api.post(`${FRIEND_ENDPOINTS.BASE}${FRIEND_ENDPOINTS.SEND_REQUEST}`, {
      user_id: userId
    });
  };

  // 接受好友请求
  const acceptFriendRequest = async (userId: number) => {
    return await api.post(`${FRIEND_ENDPOINTS.BASE}${FRIEND_ENDPOINTS.ACCEPT_REQUEST}`, {
      user_id: userId
    });
  };

  // 拒绝好友请求
  const rejectFriendRequest = async (userId: number) => {
    return await api.post(`${FRIEND_ENDPOINTS.BASE}${FRIEND_ENDPOINTS.REJECT_REQUEST}`, {
      user_id: userId
    });
  };

  // 删除好友
  const removeFriend = async (userId: number) => {
    return await api.post(`${FRIEND_ENDPOINTS.BASE}${FRIEND_ENDPOINTS.REMOVE_FRIEND}`, {
      user_id: userId
    });
  };

  // 获取好友请求列表
  const getFriendRequests = async (type: 'received' | 'sent' = 'received', cursor?: number) => {
    const endpoint = type === 'received' ? FRIEND_ENDPOINTS.REQUESTS_RECEIVED : FRIEND_ENDPOINTS.REQUESTS_SENT;
    return await api.post(`${FRIEND_ENDPOINTS.BASE}${endpoint}`, {
      cursor,
      limit: 20
    });
  };

  // 获取待处理好友请求数量
  const getPendingRequestsCount = async () => {
    return await api.post(`${FRIEND_ENDPOINTS.BASE}${FRIEND_ENDPOINTS.REQUESTS_PENDING_COUNT}`);
  };

  return {
    getFriendsList,
    sendFriendRequest,
    acceptFriendRequest,
    rejectFriendRequest,
    removeFriend,
    getFriendRequests,
    getPendingRequestsCount
  };
};
