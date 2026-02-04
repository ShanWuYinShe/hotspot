import { BaseService } from './base.service'
import { ADMIN_GROUPS } from '~/config/api'
import type {
  PaginatedData,
  AdminGroupSearchRequest,
  AdminGroupItem,
  AdminGroupDetailRequest,
  AdminGroupDetailData,
  AdminDismissGroupRequest
} from '~/types/api'

class AdminGroupsService extends BaseService {
  async search(request: AdminGroupSearchRequest = {}) {
    return this.post<PaginatedData<AdminGroupItem>>(ADMIN_GROUPS.SEARCH, request)
  }

  async detail(request: AdminGroupDetailRequest) {
    return this.post<AdminGroupDetailData>(ADMIN_GROUPS.DETAIL, request)
  }

  async dismiss(request: AdminDismissGroupRequest) {
    return this.post<void>(ADMIN_GROUPS.DISMISS, request)
  }
}

export const adminGroupsService = new AdminGroupsService()
export function useAdminGroupsService() { return adminGroupsService }
