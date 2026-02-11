/**
 * 举报服务
 */

import { post, paginate } from '../client';
import { REPORTS } from '../endpoints';
import type { ReportCreated, ReportItem, ReportTargetType, PaginationParams } from '$lib/types/api';

export const reportsService = {
	/**
	 * 创建举报
	 */
	create(data: {
		target_type: ReportTargetType;
		target_id: string;
		reason: string;
		description?: string;
	}) {
		return post<ReportCreated>(REPORTS.CREATE, data, { idempotencyKey: crypto.randomUUID() });
	},

	/**
	 * 获取我的举报记录
	 */
	getMyReports(params?: { pagination?: PaginationParams }) {
		return paginate<ReportItem>(REPORTS.ME_LIST, params);
	}
};
