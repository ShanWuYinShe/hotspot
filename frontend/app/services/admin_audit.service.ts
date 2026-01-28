import { BaseService } from './base.service'
import { ADMIN_AUDIT } from '~/config/api'
import type {
  PaginatedData,
  AuditLogItem,
  AdminAuditListRequest,
  AdminAuditDetailRequest,
  AdminAuditExportRequest,
  AdminAuditExportData,
  AdminAuditExportStatusRequest
} from '~/types/api'

class AdminAuditService extends BaseService {
  async list(request: AdminAuditListRequest = {}) {
    return this.post<PaginatedData<AuditLogItem>>(ADMIN_AUDIT.LIST, request)
  }

  async detail(request: AdminAuditDetailRequest) {
    return this.post<AuditLogItem>(ADMIN_AUDIT.DETAIL, request)
  }

  async export(request: AdminAuditExportRequest = {}) {
    return this.post<AdminAuditExportData>(ADMIN_AUDIT.EXPORT, request)
  }

  async exportStatus(request: AdminAuditExportStatusRequest) {
    return this.post<AdminAuditExportData>(ADMIN_AUDIT.EXPORT_STATUS, request)
  }
}

export const adminAuditService = new AdminAuditService()
export function useAdminAuditService() { return adminAuditService }
