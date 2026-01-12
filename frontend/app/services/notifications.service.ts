import { BaseService } from './base.service'
import { NOTIFICATIONS } from '~/config/api'
import type {
  PaginatedData,
  GetNotificationsRequest,
  NotificationItem,
  GetUnreadNotificationCountRequest,
  UnreadNotificationCountData,
  MarkNotificationsReadRequest
} from '~/types/api'

class NotificationsService extends BaseService {
  async getNotifications(request: GetNotificationsRequest = {}) {
    return this.post<PaginatedData<NotificationItem>>(NOTIFICATIONS.LIST, request)
  }

  async getUnreadCount(request: GetUnreadNotificationCountRequest = {}) {
    return this.post<UnreadNotificationCountData>(NOTIFICATIONS.UNREAD_COUNT, request)
  }

  async markAsRead(request: MarkNotificationsReadRequest = {}) {
    return this.post<void>(NOTIFICATIONS.MARK_READ, request)
  }
}

export const notificationsService = new NotificationsService()
export function useNotificationsService() { return notificationsService }
