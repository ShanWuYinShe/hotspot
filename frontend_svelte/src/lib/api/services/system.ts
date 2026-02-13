/**
 * 系统服务
 */

import { post } from '../client';
import { SYSTEM } from '../endpoints';
import type { ClientConfig, HealthStatus, SystemAnnouncementItem, SystemMeta } from '$lib/types/api';

export const systemService = {
	/**
	 * 健康检查
	 */
	health() {
		return post<HealthStatus>(SYSTEM.HEALTH, undefined, {
			skipAuth: true
		});
	},

	/**
	 * 获取系统元信息
	 */
	getMeta() {
		return post<SystemMeta>(SYSTEM.GET_META, undefined, { skipAuth: true });
	},

	/**
	 * 获取客户端配置
	 */
	getClientConfig() {
		return post<ClientConfig>(SYSTEM.GET_CLIENT_CONFIG, undefined, { skipAuth: true });
	},

	/**
	 * 获取系统公告列表（非分页，返回条数由服务端 ResponseDefaults 控制）
	 */
	getAnnouncements() {
		return post<{ items: SystemAnnouncementItem[] }>(SYSTEM.LIST_ANNOUNCEMENTS, {});
	}
};
