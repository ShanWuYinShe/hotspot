/**
 * 系统服务
 */

import { post } from '../client';
import { SYSTEM } from '../endpoints';
import type {
  ApiVersionMeta,
  ClientConfig,
  HealthStatus,
  SystemAnnouncementItem
} from '$lib/types/api';

export const systemService = {
  /**
   * 健康检查
   */
  health() {
    return post<HealthStatus>(
      SYSTEM.HEALTH,
      {},
      {
        skipAuth: true
      }
    );
  },

  /**
   * 获取系统元信息
   */
  getMeta() {
    return post<ApiVersionMeta>(SYSTEM.GET_META, {}, { skipAuth: true });
  },

  /**
   * 获取客户端配置
   */
  getClientConfig() {
    return post<ClientConfig>(SYSTEM.GET_CLIENT_CONFIG, {}, { skipAuth: true });
  },

  /**
   * 获取系统公告列表（非分页，返回条数由服务端 ResponseDefaults 控制）
   * spec: 无认证
   */
  getAnnouncements() {
    return post<{ items: SystemAnnouncementItem[] }>(
      SYSTEM.LIST_ANNOUNCEMENTS,
      {},
      { skipAuth: true }
    );
  }
};
