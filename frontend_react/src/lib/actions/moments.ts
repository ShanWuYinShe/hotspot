'use server';

/**
 * Moments Server Actions
 * 处理动态相关的服务端操作
 */

import { revalidateTag } from 'next/cache';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type { CreateMomentRequest, UpdateMomentRequest } from '@/types/api';

/**
 * 点赞动态
 */
export async function likeMoment(momentId: string) {
  const response = await serverApi.post(ENDPOINTS.MOMENTS.LIKE(momentId), { moment_id: momentId }, {});
  if (isServerSuccess(response)) {
    revalidateTag(`moment-${momentId}`, {});
    revalidateTag('feed', {});
  }
  return response;
}

/**
 * 取消点赞动态
 */
export async function unlikeMoment(momentId: string) {
  const response = await serverApi.post(ENDPOINTS.MOMENTS.UNLIKE(momentId), { moment_id: momentId }, {});
  if (isServerSuccess(response)) {
    revalidateTag(`moment-${momentId}`, {});
    revalidateTag('feed', {});
  }
  return response;
}

/**
 * 收藏动态
 */
export async function favoriteMoment(momentId: string) {
  const response = await serverApi.post(ENDPOINTS.MOMENTS.FAVORITE(momentId), { moment_id: momentId }, {});
  if (isServerSuccess(response)) {
    revalidateTag(`moment-${momentId}`, {});
    revalidateTag('favorites', {});
  }
  return response;
}

/**
 * 取消收藏动态
 */
export async function unfavoriteMoment(momentId: string) {
  const response = await serverApi.post(ENDPOINTS.MOMENTS.UNFAVORITE(momentId), { moment_id: momentId }, {});
  if (isServerSuccess(response)) {
    revalidateTag(`moment-${momentId}`, {});
    revalidateTag('favorites', {});
  }
  return response;
}

/**
 * 删除动态
 */
export async function deleteMoment(momentId: string) {
  const response = await serverApi.post(ENDPOINTS.MOMENTS.DELETE(momentId), { moment_id: momentId }, {});
  if (isServerSuccess(response)) {
    revalidateTag('feed', {});
    revalidateTag('moments', {});
  }
  return response;
}

/**
 * 创建动态
 */
export async function createMoment(data: CreateMomentRequest) {
  const response = await serverApi.post(ENDPOINTS.MOMENTS.CREATE, data, {});
  if (isServerSuccess(response)) {
    revalidateTag('feed', {});
    revalidateTag('moments', {});
  }
  return response;
}

/**
 * 更新动态
 */
export async function updateMoment(momentId: string, data: Omit<UpdateMomentRequest, 'moment_id'>) {
  const response = await serverApi.post(
    ENDPOINTS.MOMENTS.UPDATE(momentId),
    { moment_id: momentId, ...data },
    {}
  );
  if (isServerSuccess(response)) {
    revalidateTag(`moment-${momentId}`, {});
    revalidateTag('moments', {});
  }
  return response;
}
