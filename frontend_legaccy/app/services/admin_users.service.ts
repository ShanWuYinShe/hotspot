import { BaseService } from './base.service'
import { ADMIN_USERS } from '~/config/api'
import type {
  PaginatedData,
  AdminUserSearchRequest,
  AdminUserItem,
  AdminUserDetailRequest,
  AdminUserDetailData,
  AdminBanUserRequest,
  AdminUnbanUserRequest
} from '~/types/api'

class AdminUsersService extends BaseService {
  async search(request: AdminUserSearchRequest = {}) {
    return this.post<PaginatedData<AdminUserItem>>(ADMIN_USERS.SEARCH, request)
  }

  async detail(request: AdminUserDetailRequest) {
    return this.post<AdminUserDetailData>(ADMIN_USERS.DETAIL, request)
  }

  async ban(request: AdminBanUserRequest) {
    return this.post<void>(ADMIN_USERS.BAN, request)
  }

  async unban(request: AdminUnbanUserRequest) {
    return this.post<void>(ADMIN_USERS.UNBAN, request)
  }
}

export const adminUsersService = new AdminUsersService()
export function useAdminUsersService() { return adminUsersService }
