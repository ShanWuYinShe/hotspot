/**
 * 举报服务
 */

import { post, paginate } from '../client';
import { REPORTS } from '../endpoints';
import type { ReportCreated, ReportItem, ReportTargetType, ReportReason } from '$lib/types/api';

export const reportsService = {
	/**
	 * 创建举报
	 */
	create(data: {
		target_type: ReportTargetType;
		target_id: string;
		reason: ReportReason;
		description?: string;
	}) {
		return post<ReportCreated>(REPORTS.CREATE, data);
	},

	/**
	 * 获取我的举报记录
	 */
	getMyReports(params?: { limit?: number; cursor?: string }) {
		return paginate<ReportItem>(REPORTS.ME, params);
	}
};
