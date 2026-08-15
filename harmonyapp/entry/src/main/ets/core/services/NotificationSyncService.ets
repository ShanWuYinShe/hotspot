import { ApiClient } from '../api/ApiClient';
import { NotifyItem, FetchNotifyRequest, NotifyMarkReadRequest, UnreadSummaryResult } from '../types/api';
import { DefaultConfig } from '../../constants/AppConstants';
import { GlobalStateManager, UnreadCount as GlobalUnreadCount } from '../state/GlobalStateManager';
import { PreferencesUtil } from '../storage/PreferencesUtil';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF36;
const TAG: string = 'NotificationSyncService';

type Listener = () => void;

export class NotificationSyncService {
  private static instance: NotificationSyncService | null = null;

  private notifies: NotifyItem[] = [];
  private nextCursor: string = '';
  private hasMore: boolean = false;
  private isLoading: boolean = false;
  private isLoadingMore: boolean = false;
  private lastError: string = '';
  private listeners: Listener[] = [];

  private globalState = GlobalStateManager.getInstance();

  static getInstance(): NotificationSyncService {
    if (!NotificationSyncService.instance) {
      NotificationSyncService.instance = new NotificationSyncService();
    }
    return NotificationSyncService.instance;
  }

  subscribe(listener: Listener): void {
    if (!this.listeners.includes(listener)) {
      this.listeners.push(listener);
    }
  }

  unsubscribe(listener: Listener): void {
    const index = this.listeners.indexOf(listener);
    if (index >= 0) {
      this.listeners.splice(index, 1);
    }
  }

  getNotifications(): NotifyItem[] {
    return [...this.notifies];
  }

  hasMoreData(): boolean {
    return this.hasMore;
  }

  getLastError(): string {
    return this.lastError;
  }

  isInitialLoading(): boolean {
    return this.isLoading;
  }

  isMoreLoading(): boolean {
    return this.isLoadingMore;
  }

  reset(): void {
    this.notifies = [];
    this.nextCursor = '';
    this.hasMore = false;
    this.lastError = '';
    this.notify();
  }

  async refresh(): Promise<void> {
    if (this.isLoading) {
      return;
    }

    this.isLoading = true;
    this.lastError = '';
    this.notify();

    try {
      const response = await ApiClient.listNotifies({
        pagination: { limit: DefaultConfig.PAGE_SIZE_LARGE }
      } as FetchNotifyRequest);

      if (response.code !== 0 || !response.data) {
        throw new Error(response.message || '加载通知失败');
      }

      this.notifies = response.data.items ?? [];
      this.nextCursor = response.data.next_cursor ?? '';
      this.hasMore = !!response.data.next_cursor;
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载通知失败';
      this.lastError = message;
      hilog.error(DOMAIN_NUMBER, TAG, message);
      throw new Error(message);
    } finally {
      this.isLoading = false;
      this.notify();
    }
  }

  async loadMore(): Promise<void> {
    if (this.isLoadingMore || !this.hasMore || !this.nextCursor) {
      return;
    }

    this.lastError = '';
    this.isLoadingMore = true;
    this.notify();

    try {
      const response = await ApiClient.listNotifies({
        pagination: {
          limit: DefaultConfig.PAGE_SIZE_LARGE,
          cursor: this.nextCursor
        }
      } as FetchNotifyRequest);

      if (response.code !== 0 || !response.data) {
        throw new Error(response.message || '加载通知失败');
      }

      const items = response.data.items ?? [];
      if (items.length > 0) {
        this.notifies = this.notifies.concat(items);
      }
      this.nextCursor = response.data.next_cursor ?? '';
      this.hasMore = !!response.data.next_cursor;
    } catch (error) {
      const message = error instanceof Error ? error.message : '加载更多通知失败';
      this.lastError = message;
      hilog.error(DOMAIN_NUMBER, TAG, message);
      throw new Error(message);
    } finally {
      this.isLoadingMore = false;
      this.notify();
    }
  }

  async markAsRead(ids: string[]): Promise<void> {
    if (ids.length === 0) {
      return;
    }

    const request: NotifyMarkReadRequest = { scope: 'ids', notify_ids: ids };
    const response = await ApiClient.markNotifiesRead(request);
    if (response.code !== 0) {
      throw new Error(response.message || '标记通知失败');
    }

    const idSet = new Set<string>(ids);
    this.notifies = this.notifies.map((item: NotifyItem): NotifyItem => {
      if (idSet.has(item.notify_id)) {
        const readItem: NotifyItem = {
          notify_id: item.notify_id,
          type: item.type,
          is_read: true,
          created_at: item.created_at,
          payload: item.payload
        };
        return readItem;
      }
      return item;
    });
    this.notify();
  }

  async markAllAsRead(): Promise<void> {
    const response = await ApiClient.markAllNotifiesRead();
    if (response.code !== 0) {
      throw new Error(response.message || '标记全部已读失败');
    }

    this.notifies = this.notifies.map((item: NotifyItem): NotifyItem => {
      const readItem: NotifyItem = {
        notify_id: item.notify_id,
        type: item.type,
        is_read: true,
        created_at: item.created_at,
        payload: item.payload
      };
      return readItem;
    });
    this.notify();
  }

  async syncUnreadCount(): Promise<void> {
    try {
      const response = await ApiClient.getUnreadCount();
      if (response.code !== 0 || !response.data) {
        throw new Error(response.message || '获取未读通知失败');
      }
      await this.updateGlobalUnreadState(response.data);
    } catch (error) {
      const message = error instanceof Error ? error.message : '同步未读通知失败';
      hilog.warn(DOMAIN_NUMBER, TAG, message);
    }
  }

  private async updateGlobalUnreadState(result: UnreadSummaryResult): Promise<void> {
    const globalCount: GlobalUnreadCount = {
      total_unread_count: result.total
    };
    this.globalState.updateUnreadCount(globalCount);
    await PreferencesUtil.setObject('unread_counts', globalCount);
  }

  private notify(): void {
    this.listeners.forEach((listener: Listener) => listener());
  }
}
