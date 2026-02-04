import { BaseService } from './base.service'
import { ADMIN_ANNOUNCEMENTS, ANNOUNCEMENTS } from '~/config/api'
import type {
  PaginatedData,
  SystemAnnouncementItem,
  PublicAnnouncementListRequest,
  AdminCreateAnnouncementRequest,
  AdminAnnouncementListRequest,
  AdminDeleteAnnouncementRequest
} from '~/types/api'

class AdminAnnouncementsService extends BaseService {
  // 公开 API - 获取系统公告列表
  async getPublicList(request: PublicAnnouncementListRequest = {}) {
    return this.post<PaginatedData<SystemAnnouncementItem>>(ANNOUNCEMENTS.LIST, request)
  }

  // 管理员 API
  async list(request: AdminAnnouncementListRequest = {}) {
    return this.post<PaginatedData<SystemAnnouncementItem>>(ADMIN_ANNOUNCEMENTS.LIST, request)
  }

  async create(request: AdminCreateAnnouncementRequest) {
    return this.post<SystemAnnouncementItem>(ADMIN_ANNOUNCEMENTS.CREATE, request)
  }

  async delete(request: AdminDeleteAnnouncementRequest) {
    return this.post<void>(ADMIN_ANNOUNCEMENTS.DELETE, request)
  }
}

export const adminAnnouncementsService = new AdminAnnouncementsService()
export function useAdminAnnouncementsService() { return adminAnnouncementsService }
