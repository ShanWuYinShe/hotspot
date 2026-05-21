import { post, paginate, generateIdempotencyKey } from '../client';
import { REPORTS } from '../endpoints';
import type {
  ReportCreated,
  ReportItem,
  ReportReason,
  ReportStatus,
  ReportTargetType,
  PaginationParams
} from '@/lib/types/api';

export const reportsService = {
  create(data: {
    target_type: ReportTargetType;
    target_id: string;
    reason: ReportReason;
    description?: string;
  }) {
    return post<ReportCreated>(REPORTS.CREATE, data, { idempotencyKey: generateIdempotencyKey() });
  },

  getMyReports(params?: {
    status?: ReportStatus;
    target_type?: ReportTargetType;
    pagination?: PaginationParams;
  }) {
    return paginate<ReportItem>(REPORTS.ME_LIST, params);
  }
};
