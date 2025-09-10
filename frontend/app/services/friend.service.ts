/**
 * 好友服务 - Composition API 风格
 * 
 * ⚠️ 注意：根据 hotspot.md 3.3.1，好友系统标记为"预留"功能
 * 后端暂未实现具体API端点，此服务暂时返回模拟数据或错误提示
 */

// import { FRIEND_ENDPOINTS } from "~/config/api"; // 暂未实现
import type { PaginatedResponse, BasicSuccess, User } from "~/types";

export const useFriendService = () => {
  // const api = useApi(); // 暂未使用

  // 获取好友列表（暂未实现）
  const getFriendsList = async (cursor?: number): Promise<PaginatedResponse<User>> => {
    throw new Error('好友功能暂未实现，请等待后端API上线');
    // return await api.post(`${FRIEND_ENDPOINTS.BASE}${FRIEND_ENDPOINTS.LIST}`, {
    //   cursor,
    //   limit: 20
    // }, { auth: true });
  };

  // 发送好友请求（暂未实现）
  const sendFriendRequest = async (userId: string): Promise<BasicSuccess> => {
    throw new Error('好友功能暂未实现，请等待后端API上线');
  };

  // 接受好友请求（暂未实现）
  const acceptFriendRequest = async (userId: string): Promise<BasicSuccess> => {
    throw new Error('好友功能暂未实现，请等待后端API上线');
  };

  // 拒绝好友请求（暂未实现）
  const rejectFriendRequest = async (userId: string): Promise<BasicSuccess> => {
    throw new Error('好友功能暂未实现，请等待后端API上线');
  };

  // 删除好友（暂未实现）
  const removeFriend = async (userId: string): Promise<BasicSuccess> => {
    throw new Error('好友功能暂未实现，请等待后端API上线');
  };

  // 获取好友请求列表（暂未实现）
  const getFriendRequests = async (type: 'received' | 'sent' = 'received', cursor?: string): Promise<PaginatedResponse<User>> => {
    throw new Error('好友功能暂未实现，请等待后端API上线');
  };

  // 获取待处理好友请求数量（暂未实现）
  const getPendingRequestsCount = async (): Promise<{ count: number }> => {
    // 返回默认值，避免UI错误
    return { count: 0 };
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
