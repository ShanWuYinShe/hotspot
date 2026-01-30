/**
 * 系统服务
 */

import { post, paginate } from '../client';
import { SYSTEM, GLOBAL_SETTINGS, ANNOUNCEMENTS } from '../endpoints';
import type { GlobalSettings, HealthStatus, SystemAnnouncementItem } from '$lib/types/api';

export const systemService = {
	/**
	 * 健康检查
	 */
	health() {
		return post<HealthStatus>(SYSTEM.HEALTH, undefined, {
			skipAuth: true
		});
	}
};

export const globalSettingsService = {
	/**
	 * 获取全局配置
	 */
	fetch() {
		return post<GlobalSettings>(GLOBAL_SETTINGS.FETCH, undefined, { skipAuth: true });
	}
};

export const announcementsService = {
	/**
	 * 获取系统公告列表
	 */
	getList(params?: { limit?: number; cursor?: string }) {
		return paginate<SystemAnnouncementItem>(ANNOUNCEMENTS.LIST, params);
	}
};
