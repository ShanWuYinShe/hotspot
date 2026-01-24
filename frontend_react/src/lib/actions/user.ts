'use server';

/**
 * User Server Actions
 * 处理用户相关的服务端操作
 */

import { revalidateTag } from 'next/cache';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';

/**
 * 关注用户
 */
export async function followUser(userId: number) {
  const response = await serverApi.post(ENDPOINTS.USER.FOLLOW(userId));
  if (isServerSuccess(response)) {
    revalidateTag(`user-${userId}`);
    revalidateTag('following');
  }
  return response;
}

/**
 * 取消关注用户
 */
export async function unfollowUser(userId: number) {
  const response = await serverApi.post(ENDPOINTS.USER.UNFOLLOW(userId));
  if (isServerSuccess(response)) {
    revalidateTag(`user-${userId}`);
    revalidateTag('following');
  }
  return response;
}

/**
 * 更新用户资料
 */
export async function updateProfile(data: {
  nickname?: string;
  avatar_file_id?: string;
  bio?: string;
  gender?: 'male' | 'female' | 'other';
}) {
  const response = await serverApi.put(ENDPOINTS.USER.ME_PROFILE, data);
  if (isServerSuccess(response)) {
    revalidateTag('profile');
    revalidateTag('me');
  }
  return response;
}

/**
 * 更新用户设置
 */
export async function updateSettings(data: {
  privacy_level?: 'public' | 'friends' | 'private';
  location_visible?: boolean;
  online_visible?: boolean;
  last_online_visible?: boolean;
  friends_visible?: 'public' | 'friends' | 'private';
  following_visible?: 'public' | 'friends' | 'private';
  followers_visible?: 'public' | 'friends' | 'private';
  moments_visible?: 'public' | 'friends' | 'private';
  stats_visible?: 'public' | 'friends' | 'private';
  allow_follow?: boolean;
  allow_private_chat?: 'everyone' | 'friends' | 'nobody';
  searchable?: boolean;
  phone_searchable?: boolean;
  nickname_searchable?: boolean;
  location_searchable?: boolean;
}) {
  const response = await serverApi.put(ENDPOINTS.USER.ME_SETTINGS, data);
  if (isServerSuccess(response)) {
    revalidateTag('settings');
  }
  return response;
}
