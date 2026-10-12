/**
 * 消息列表页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 * 会话列表：LazyForEach 虚拟化（过滤后的视图绑 filteredChatDataSource）
 * ＋ MessageCacheService（RDB）本地缓存首屏兜底；行操作走长按上下文菜单
 * （置顶/删除），不使用 swipeAction。
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_LAST_MESSAGE_SYNC_TIME } from '../../../core/state/GlobalStateManager';
import { formatCardTime } from '../../../core/utils/FormatUtil';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { Pressable } from '../../../components/Pressable';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { PageNames } from '../../../constants/AppConstants';
import {
  PrivateChatItem,
  GroupItem,
  AccountInfo,
  FetchJoinedGroupsRequest,
  ExitGroupRequest,
  DeletePrivateChatRequest
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { Skeleton } from '../../../components/states/Skeleton';
import { SkeletonShimmer } from '../../../components/states/SkeletonShimmer';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemDialog } from '../../../components/SystemDialog';
import { SystemInput } from '../../../components/SystemInput';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { useMasterDetail } from '../../../core/responsive/Breakpoint';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { MessageCacheService, ChatCacheItem } from '../../../core/database/MessageCacheService';
import { VibrationUtil } from '../../../core/utils/VibrationUtil';
import { scanBarcode, scanCore } from '@kit.ScanKit';
import { common } from '@kit.AbilityKit';
import pasteboard from '@ohos.pasteboard';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { InteractionStateStore } from '../../../core/state/InteractionStateStore';
import { TabsPillGroup, TabItem } from '../../../components/TabsPillGroup';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'MessagePage';

interface ChatItem {
  id: string;
  type: 'private' | 'group';
  name: string;
  avatarUrl?: string;
  lastMessage?: string;
  lastTime: string;
  lastTimestamp: number;
  isPinned: boolean;
  unreadCount: number;
  data: PrivateChatItem | GroupItem;
}

@Preview
@Component
export struct MessagePage {
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;
  @StorageLink(KEY_LAST_MESSAGE_SYNC_TIME) @Watch('onMessageSync') lastMessageSyncTime: number = 0;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';

  private chatListDataSource: BasicDataSource<ChatItem> = new BasicDataSource([]);
  private messageCacheService = MessageCacheService.getInstance();
  private isDestroyed: boolean = false;
  private loadVersion: number = 0;

  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State errorMessage: string = '';
  // 桌面宽屏（≥960）master-detail：当前选中的会话 key，用于高亮列表项
  @State selectedChatKey: string = '';
  @State hasData: boolean = false;
  // 分页状态：私聊
  private privateHasMore: boolean = false;
  private privateNextCursor: string = '';
  // 分页状态：群聊
  private groupHasMore: boolean = false;
  private groupNextCursor: string = '';
  @State isLoadingMore: boolean = false;
  @State deletingChatKeys: string[] = [];
  @State showDeleteChatConfirm: boolean = false;
  @State pendingDeleteChat: ChatItem | null = null;
  @State pressedChatKey: string = '';
  @State searchKeyword: string = '';
  @State chatItems: ChatItem[] = [];
  @State filteredChatItems: ChatItem[] = [];
  /** 过滤后的视图数据源：LazyForEach 绑定它做虚拟化（原实现用 ForEach 全量渲染） */
  private filteredChatDataSource: BasicDataSource<ChatItem> = new BasicDataSource<ChatItem>([]);

  private getOnlineChats(): ChatItem[] {
    const online: ChatItem[] = [];
    for (let i = 0; i < this.chatListDataSource.totalCount(); i++) {
      const item = this.chatListDataSource.getData(i);
      if (item.type === 'private') {
        const pci = item.data as PrivateChatItem;
        if (pci.presence_status === 'online') {
          online.push(item);
        }
      }
    }
    return online;
  }

  private getTotalUnread(): number {
    let count = 0;
    for (let i = 0; i < this.chatListDataSource.totalCount(); i++) {
      count += this.chatListDataSource.getData(i).unreadCount;
    }
    return count;
  }

  // 会话过滤 Tab（skill hotspot-harmony-ui 契约：列表头部「全部 / 未读 / 已读」
  // TabsPillGroup，对齐前端 TabBar 形态；0=全部 1=未读 2=已读）
  @State chatFilterIndex: number = 0;

  private applyFilter(): void {
    const keyword = this.searchKeyword.trim().toLowerCase();
    this.filteredChatItems = this.chatItems.filter((item: ChatItem) => {
      if (this.chatFilterIndex === 1 && item.unreadCount <= 0) return false;
      if (this.chatFilterIndex === 2 && item.unreadCount > 0) return false;
      if (keyword !== '' && item.name.toLowerCase().indexOf(keyword) < 0) return false;
      return true;
    });
    // 同步到 LazyForEach 数据源（虚拟化渲染的唯一数据入口）
    this.filteredChatDataSource.setData(this.filteredChatItems);
  }

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    // 先从缓存加载（避免白屏），再强制调接口刷新
    await this.loadFromCache();
    await this.loadData(true);
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  onMessageSync() {
    if (this.lastMessageSyncTime > 0) {
      hilog.info(DOMAIN_NUMBER, TAG, '收到消息同步通知，刷新列表');
      this.loadData(false, true);
    }
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private async loadData(forceRefresh: boolean = false, refresh: boolean = false) {
    if (!forceRefresh) {
      await this.loadFromCache();
    }

    // 请求版本号：每次进入递增，避免较慢的旧请求解析后覆盖较新的结果
    const capturedVersion = ++this.loadVersion;
    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';

    try {
      const results = await LoadingUtil.withMinimumTime(() =>
        Promise.allSettled([this.loadPrivateChats(), this.loadGroupChats()])
      );
      if (this.isDestroyed) return;
      // 旧请求晚于新请求返回：丢弃结果，避免覆盖最新数据
      if (capturedVersion !== this.loadVersion) return;
      const privateChatsResult = results[0];
      const groupChatsResult = results[1];

      let allChats: ChatItem[] = [];

      if (privateChatsResult.status === 'fulfilled' && privateChatsResult.value) {
        allChats = allChats.concat(privateChatsResult.value);
      }
      if (groupChatsResult.status === 'fulfilled' && groupChatsResult.value) {
        allChats = allChats.concat(groupChatsResult.value);
      }

      allChats = this.mergeAndSortChats([], allChats);

      this.chatListDataSource.setData(allChats);
      this.chatItems = allChats;
      this.applyFilter();
      this.hasData = allChats.length > 0;

      // 仅用户主动下拉刷新时触发振动反馈
      if (this.isRefreshing) {
        VibrationUtil.startSuccess();
      }

      this.updateCache(allChats);
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载消息失败');
      // 仅当当前请求仍为最新时才上报错误/清空数据
      if (capturedVersion === this.loadVersion) {
        if (this.chatListDataSource.totalCount() === 0) {
          this.errorMessage = '加载消息失败，请检查网络';
          this.hasData = false;
        }
        if (this.isRefreshing) {
          VibrationUtil.startFail();
        }
      }
    } finally {
      // 仅当前最新请求可重置加载态，否则会把新请求的 loading 提前置 false
      if (capturedVersion === this.loadVersion) {
        this.isLoading = false;
        this.isRefreshing = false;
      }
    }
  }

  private async loadFromCache() {
    const cachedItems = await this.messageCacheService.getCachedChatList();
    if (cachedItems.length > 0) {
      const chats = cachedItems.map(item => {
        const chatType = item.type as 'private' | 'group';
        const parsedData = JSON.parse(item.data) as PrivateChatItem | GroupItem;
        const chatItem: ChatItem = {
          id: item.id,
          type: chatType,
          name: item.name,
          avatarUrl: item.avatarUrl,
          lastMessage: item.lastMessage,
          lastTime: item.lastTime,
          lastTimestamp: this.parseTimestamp(item.lastTime),
          isPinned: this.extractPinnedFromData(chatType, parsedData),
          unreadCount: item.unreadCount,
          data: parsedData
        };
        return chatItem;
      });
      this.chatListDataSource.setData(chats);
      this.chatItems = chats;
      this.applyFilter();
      this.hasData = true;
      hilog.info(DOMAIN_NUMBER, TAG, `Loaded ${chats.length} chats from cache`);
    }
  }

  private async updateCache(chats: ChatItem[]): Promise<void> {
    const cacheItems: ChatCacheItem[] = chats.map(item => {
      const cacheItem: ChatCacheItem = {
        id: item.id,
        type: item.type,
        name: item.name,
        avatarUrl: item.avatarUrl,
        lastMessage: item.lastMessage,
        lastTime: item.lastTime,
        unreadCount: item.unreadCount,
        data: JSON.stringify(item.data)
      };
      return cacheItem;
    });
    await this.messageCacheService.cacheChatList(cacheItems);
  }

  private async loadPrivateChats(cursor?: string): Promise<ChatItem[]> {
    try {
      const response = await ApiClient.getPrivateChats({ pagination: { limit: 50, cursor: cursor } });
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.privateHasMore = !!(response.data?.next_cursor);
        this.privateNextCursor = response.data?.next_cursor ?? '';
        return items
          .filter((item: PrivateChatItem) => item.other_user?.user_id)
          .map((item: PrivateChatItem): ChatItem => {
            const chatItem: ChatItem = {
              id: item.other_user.user_id,
              type: 'private',
              name: item.other_user.nickname ?? '',
              avatarUrl: item.other_user.avatar_url,
              lastMessage: this.formatLastMessage(item.last_message),
              lastTime: item.last_message?.sent_at ?? item.last_activity_at,
              lastTimestamp: this.parseTimestamp(item.last_message?.sent_at ?? item.last_activity_at),
              isPinned: item.is_pinned ?? false,
              unreadCount: item.unread_count,
              data: item
            };
            return chatItem;
          });
      }
      return [];
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载私聊失败');
      return [];
    }
  }

  private async loadGroupChats(cursor?: string): Promise<ChatItem[]> {
    try {
      const request: FetchJoinedGroupsRequest = { pagination: { limit: 50, cursor: cursor } };
      const response = await ApiClient.getJoinedGroups(request);
      if (response.code === 0) {
        const items = response.data?.items ?? [];
        this.groupHasMore = !!(response.data?.next_cursor);
        this.groupNextCursor = response.data?.next_cursor ?? '';
        return items
          .filter((item: GroupItem) => item.group?.group_id)
          .map((item: GroupItem): ChatItem => {
            const chatItem: ChatItem = {
              id: item.group.group_id,
              type: 'group',
              name: item.group.name ?? '',
              avatarUrl: item.group.avatar_url,
              lastMessage: this.formatLastMessage(item.last_message),
              lastTime: item.last_message?.sent_at ?? item.last_activity_at ?? item.created_at ?? '',
              lastTimestamp: this.parseTimestamp(item.last_message?.sent_at ?? item.last_activity_at ?? item.created_at ?? ''),
              isPinned: item.viewer_state?.is_pinned ?? false,
              unreadCount: item.viewer_state?.unread_count || 0,
              data: item
            };
            return chatItem;
          });
      }
      return [];
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载群聊失败');
      return [];
    }
  }

  private async loadMore(): Promise<void> {
    if (this.isLoadingMore) return;
    if (!this.privateHasMore && !this.groupHasMore) return;
    this.isLoadingMore = true;
    try {
      const promises: Promise<ChatItem[]>[] = [];
      if (this.privateHasMore) {
        promises.push(this.loadPrivateChats(this.privateNextCursor));
      }
      if (this.groupHasMore) {
        promises.push(this.loadGroupChats(this.groupNextCursor));
      }
      const results = await Promise.allSettled(promises);
      let newChats: ChatItem[] = [];
      for (const result of results) {
        if (result.status === 'fulfilled' && result.value) {
          newChats = newChats.concat(result.value);
        }
      }
      if (newChats.length > 0) {
        const existingItems = this.getAllDataFromSource();
        const mergedItems = this.mergeAndSortChats(existingItems, newChats);
        this.chatListDataSource.setData(mergedItems);
        this.chatItems = mergedItems;
        this.applyFilter();
        this.updateCache(mergedItems);
      }
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载更多消息失败');
    } finally {
      this.isLoadingMore = false;
    }
  }

  private formatLastMessage(message: Object | null | undefined): string {
    if (!message) return '';
    const msgRecord = message as Record<string, Object>;
    const msgType = String(msgRecord.type || '');
    if (msgType === 'text') {
      const content = msgRecord.content;
      return content ? String(content) : '';
    }
    if (msgType === 'image') return '[图片]';
    if (msgType === 'video') return '[视频]';
    if (msgType === 'audio') return '[语音]';
    if (msgType === 'file') return '[文件]';
    return '[消息]';
  }

  private extractPinnedFromData(type: 'private' | 'group', data: PrivateChatItem | GroupItem): boolean {
    if (type === 'private') {
      const privateData = data as PrivateChatItem;
      return privateData.is_pinned ?? false;
    }
    const groupData = data as GroupItem;
    return groupData.viewer_state?.is_pinned ?? false;
  }

  private parseTimestamp(value: string): number {
    const ts = new Date(value).getTime();
    return Number.isFinite(ts) ? ts : 0;
  }

  private buildChatKey(item: ChatItem): string {
    return `${item.type}:${item.id}`;
  }

  private mergeAndSortChats(existing: ChatItem[], incoming: ChatItem[]): ChatItem[] {
    const merged = new Map<string, ChatItem>();
    const upsert = (chat: ChatItem): void => {
      const key = this.buildChatKey(chat);
      const previous = merged.get(key);
      if (!previous || this.compareChatItems(chat, previous) < 0) {
        merged.set(key, chat);
      }
    };
    existing.forEach(upsert);
    incoming.forEach(upsert);
    const items = Array.from(merged.values());
    items.sort((a, b) => this.compareChatItems(a, b));
    return items;
  }

  private compareChatItems(a: ChatItem, b: ChatItem): number {
    // 1. 置顶优先
    if (a.isPinned !== b.isPinned) {
      return a.isPinned ? -1 : 1;
    }
    // 2. 按最新消息时间倒序（已读/未读不参与排序，避免列表位置跳动；与 Web 端一致）
    const timeDiff = b.lastTimestamp - a.lastTimestamp;
    if (timeDiff !== 0) {
      return timeDiff;
    }
    if (a.type !== b.type) {
      return a.type === 'private' ? -1 : 1;
    }
    // id 是 int64 字符串，禁止 Number() 数值运算（>2^53 精度丢失），统一字符串比较
    return a.id.localeCompare(b.id);
  }

  private formatTime(timeStr: string): string {
    // 统一收敛 FormatUtil（跨年分支此前走 toLocaleDateString 输出美式日期）。
    return formatCardTime(timeStr);
  }

  private enterChat(item: ChatItem) {
    // 桌面宽屏（≥960）时记录选中项用于高亮，但依然 push 独立 ChatPage 保持聊天功能完整
    this.selectedChatKey = this.getChatKey(item);
    // 对齐前端：点击未读会话后徽章立即清零（列表按时间排序，位置不跳动）
    if (item.unreadCount > 0) {
      const updated: ChatItem[] = [];
      for (const chat of this.chatItems) {
        if (chat.id === item.id && chat.type === item.type) {
          updated.push({
            id: chat.id,
            type: chat.type,
            name: chat.name,
            avatarUrl: chat.avatarUrl,
            lastMessage: chat.lastMessage,
            lastTime: chat.lastTime,
            lastTimestamp: chat.lastTimestamp,
            isPinned: chat.isPinned,
            unreadCount: 0,
            data: chat.data
          });
        } else {
          updated.push(chat);
        }
      }
      this.chatItems = updated;
      this.chatListDataSource.setData(updated);
      this.applyFilter();
    }
    const targetUserId = item.type === 'private' ? (item.data as PrivateChatItem).other_user?.user_id ?? item.id : item.id;
    const chatParams = new ChatRouteParam();
    chatParams.chatId = targetUserId;
    chatParams.chatType = item.type === 'private' ? 'private_chat' : 'group_chat';
    chatParams.title = item.name;
    if (item.type === 'private') {
      chatParams.targetId = targetUserId;
      chatParams.presenceText = this.isChatOnline(item) ? '在线' : '离线';
    }
    AppRouter.push(PageNames.CHAT, chatParams);
  }

  private getChatKey(item: ChatItem): string {
    return `${item.type}:${item.id}`;
  }

  private isChatDeleting(item: ChatItem): boolean {
    return this.deletingChatKeys.indexOf(this.getChatKey(item)) >= 0;
  }

  private isChatSelected(item: ChatItem): boolean {
    return useMasterDetail(this.currentBreakpoint) && this.selectedChatKey === this.getChatKey(item);
  }

  private getChatItemBackground(item: ChatItem): ResourceColor {
    // 选中/未选中按压反馈不同（四态契约）：选中按压 = accent 底再加深一档；
    // 未选中按压 = surfaceSubtle（原实现两态按压同色，选中按压与常态无差）
    if (this.isChatSelected(item)) {
      return this.pressedChatKey === this.getChatKey(item) ?
        this.getCurrentThemePalette().buttonSecondaryBackground :
        this.getCurrentThemePalette().accentMuted;
    }
    if (this.pressedChatKey === this.getChatKey(item)) {
      return this.getCurrentThemePalette().surfaceSubtle;
    }
    if (item.isPinned) {
      return this.getCurrentThemePalette().surfaceSubtle;
    }
    return DesignConstants.COLOR.TRANSPARENT;
  }

  private setChatDeleting(item: ChatItem, pending: boolean): void {
    const key = this.getChatKey(item);
    if (pending) {
      if (this.deletingChatKeys.indexOf(key) < 0) {
        this.deletingChatKeys = this.deletingChatKeys.concat([key]);
      }
      return;
    }
    this.deletingChatKeys = this.deletingChatKeys.filter((existingKey: string) => existingKey !== key);
  }

  private async deleteChat(item: ChatItem): Promise<void> {
    if (this.isChatDeleting(item)) {
      return;
    }
    this.setChatDeleting(item, true);
    try {
      VibrationUtil.startShort();
      if (item.type === 'private') {
        const privateData = item.data as PrivateChatItem;
        const targetUserId = privateData.other_user?.user_id ?? item.id;
        const request: DeletePrivateChatRequest = { target_user_id: targetUserId };
        await ApiClient.deletePrivateChat(request);
      } else {
        const groupData = item.data as GroupItem;
        const groupId = groupData.group?.group_id ?? item.id;
        const request: ExitGroupRequest = { group_id: groupId };
        await ApiClient.leaveGroup(request);
        InteractionStateStore.getInstance().updateGroupMembership(groupId, { is_member: false });
      }

      const index = this.chatListDataSource.getDataIndex(item);
      if (index >= 0) {
        this.chatListDataSource.deleteData(index);
      }
      this.chatItems = this.getAllDataFromSource();
      this.applyFilter();
      this.updateCache(this.getAllDataFromSource());
      SystemToast.show('操作成功', this.getUIContext());
    } catch {
      SystemToast.showError('操作失败', this.getUIContext());
    } finally {
      this.setChatDeleting(item, false);
    }
  }

  private getAllDataFromSource(): ChatItem[] {
    const items: ChatItem[] = [];
    for (let i = 0; i < this.chatListDataSource.totalCount(); i++) {
      items.push(this.chatListDataSource.getData(i));
    }
    return items;
  }

  private withPinned(chat: ChatItem, pinned: boolean): ChatItem {
    return {
      id: chat.id,
      type: chat.type,
      name: chat.name,
      avatarUrl: chat.avatarUrl,
      lastMessage: chat.lastMessage,
      lastTime: chat.lastTime,
      lastTimestamp: chat.lastTimestamp,
      isPinned: pinned,
      unreadCount: chat.unreadCount,
      data: chat.data
    };
  }

  /** 置顶/取消置顶：乐观更新并重排，失败回滚（对齐前端 togglePin） */
  private async togglePin(item: ChatItem): Promise<void> {
    const next = !item.isPinned;
    const apply = (pinned: boolean): void => {
      this.chatItems = this.chatItems.map((chat: ChatItem) =>
        chat.id === item.id && chat.type === item.type ? this.withPinned(chat, pinned) : chat
      );
      const merged = this.mergeAndSortChats([], this.chatItems);
      this.chatItems = merged;
      this.chatListDataSource.setData(merged);
      this.applyFilter();
    };

    apply(next);
    try {
      if (item.type === 'private') {
        const targetUserId = (item.data as PrivateChatItem).other_user?.user_id ?? item.id;
        await ApiClient.updatePrivateChatSettings({ target_user_id: targetUserId, is_pinned: next });
      } else {
        await ApiClient.updateGroupMySettings({ group_id: item.id, is_pinned: next });
      }
      this.updateCache(this.chatItems);
    } catch {
      apply(!next);
      SystemToast.showError('操作失败', this.getUIContext());
    }
  }

  build() {
  Stack() {
  PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint}) {
    Column() {
      // 粘性头部 —— 对齐前端移动端 PageHeader titleSize=2xl：未读计数与「+」在标题右侧同排
      SystemPageHeader({
        title: '消息',
        titleSize: DesignConstants.FONT.SIZE.XXL,
        titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
        backgroundVariant: 'transparent'
      }) {
        // trailing: 未读计数 + 添加(菜单)（对齐前端移动端 trailing：N 条未读 / 全部已读 + Plus ghost）；
        // 无会话时状态文本整体隐藏（空列表上「全部已读」无意义）
        Row({ space: DesignConstants.SPACING.MD2 }) {
          if (this.chatItems.length > 0) {
            if (this.getTotalUnread() > 0) {
              Text(`${this.getTotalUnread()} 条未读`)
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getCurrentThemePalette().textSecondary)
            } else {
              Text('全部已读')
                .fontSize(DesignConstants.FONT.SIZE.XS)
                .fontColor(this.getCurrentThemePalette().textSecondary)
            }
          }

          Row() {
            LucideIcon({
              icon: AppIcon.ADD,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
              paletteRole: 'textSecondary'
            })
          }
          .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
          .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
          .justifyContent(FlexAlign.Center)
          .borderRadius(DesignConstants.RADIUS.FULL)
          // 对齐前端 surface-flat 圆钮（size-9 浅底圆形 + 次色 Plus）
          .backgroundColor(this.getCurrentThemePalette().surfaceSubtle)
          .responseRegion({
            x: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.LAYOUT.ICON_BUTTON_SIZE) / 2,
            y: -(DesignConstants.LAYOUT.MIN_HIT_TARGET - DesignConstants.LAYOUT.ICON_BUTTON_SIZE) / 2,
            width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
            height: DesignConstants.LAYOUT.MIN_HIT_TARGET
          })
          .bindMenu([
            {
              value: '发起群聊',
              action: () => {
                AppRouter.push(PageNames.CREATE_GROUP);
              }
            },
            {
              value: '添加朋友',
              action: () => {
                AppRouter.push(PageNames.SEARCH);
              }
            },
            {
              value: '扫一扫',
              action: async () => {
                try {
                  if (canIUse('SystemCapability.Multimedia.Scan.ScanBarcode')) {
                    const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
                    const scanTypes: scanCore.ScanType[] = [scanCore.ScanType.ALL];
                    const options: scanBarcode.ScanOptions = {
                      scanTypes: scanTypes,
                      enableMultiMode: true,
                      enableAlbum: true
                    };
                    const result = await scanBarcode.startScanForResult(context, options);
                    if (result.originalValue) {
                      const scanContent: string = result.originalValue;
                      const systemPasteboard = pasteboard.getSystemPasteboard();
                      const pasteData = pasteboard.createData(pasteboard.MIMETYPE_TEXT_PLAIN, scanContent);
                      await systemPasteboard.setData(pasteData);
                      SystemToast.show(`扫码内容已复制: ${scanContent}`, this.getUIContext());
                    }
                  } else {
                    SystemToast.show('当前设备不支持扫码', this.getUIContext());
                  }
                } catch {
                  hilog.error(DOMAIN_NUMBER, TAG, '扫描启动失败');
                  SystemToast.showError('扫描启动失败', this.getUIContext());
                }
              }
            }
          ])
        }
      }

      // 搜索栏（对齐前端 SearchInput：前缀图标 + 实时过滤）
      SystemInput({
        value: this.searchKeyword,
        placeholder: '搜索会话…',
        prefixIcon: AppIcon.SEARCH,
        showPrefixIcon: true,
        inputHeight: DesignConstants.COMPONENT.INPUT_HEIGHT,
        onChange: (value: string) => {
          this.searchKeyword = value;
          this.applyFilter();
        }
      })
      .margin({ bottom: DesignConstants.SPACING.SM })

      // 会话状态过滤（skill hotspot-harmony-ui 契约：头部「全部 / 未读 / 已读」，
      // 对齐前端 TabBar 胶囊形态；未读 Tab 带角标计数，客户端过滤不重新请求）
      TabsPillGroup({
        tabs: [
          { label: '全部' },
          { label: '未读', count: this.getTotalUnread() },
          { label: '已读' }
        ] as TabItem[],
        activeIndex: this.chatFilterIndex,
        actions: {
          onChange: (index: number) => {
            if (this.chatFilterIndex !== index) {
              this.chatFilterIndex = index;
              this.applyFilter();
            }
          }
        }
      })
      .padding({ bottom: DesignConstants.SPACING.SM })

      // 宽屏（master-detail）提示：当前点击会话仍 push 独立页（保持聊天功能完整）
      if (useMasterDetail(this.currentBreakpoint)) {
        Row({ space: DesignConstants.SPACING.SM }) {
          LucideIcon({
            icon: AppIcon.INFO,
            iconSize: DesignConstants.FONT.SIZE.SM,
            paletteRole: 'textSecondary'
          })
          Text('宽屏模式：点击左侧会话将在新页面打开聊天窗口')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .layoutWeight(1)
        }
        .width('100%')
        .padding({
          top: DesignConstants.SPACING.XS,
          bottom: DesignConstants.SPACING.SM
        })
      }

      // 内容区域
      if (this.isLoading) {
        // 骨架屏会话列表（对齐前端消息列表加载态；trailing lambda 组件包 Column 承载 layoutWeight）
        Column() {
          SkeletonShimmer({ active: true }) {
            Column() {
              ForEach([0, 1, 2, 3, 4], (i: number) => {
                Row({ space: DesignConstants.SPACING.MD }) {
                  Skeleton({ blockWidth: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL, blockHeight: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL, cornerRadius: DesignConstants.RADIUS.FULL })
                  Column({ space: DesignConstants.SPACING.SM }) {
                    Skeleton({ blockWidth: '55%', blockHeight: DesignConstants.FONT.SIZE.MD, cornerRadius: DesignConstants.RADIUS.SM })
                    Skeleton({ blockWidth: '35%', blockHeight: DesignConstants.FONT.SIZE.XS, cornerRadius: DesignConstants.RADIUS.SM })
                  }
                  .alignItems(HorizontalAlign.Start)
                  .layoutWeight(1)
                }
                .width('100%')
                .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
              }, (i: number) => `conv-skeleton-${i}`)
            }
            .width('100%')
          }
        }
        .width('100%')
        .layoutWeight(1)
      } else if (!this.hasData && this.errorMessage) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          EmptyState({ icon: AppIcon.WARNING, message: this.errorMessage, actionLabel: '重试', onAction: () => { this.loadData(true); } })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadData(true, true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
        )
      } else if (!this.hasData) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          // 空会话对齐前端：裸气泡图标 + muted「没有找到会话」（无卡片容器）
          Column() {
            Text('最近')
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textSecondary)
              .width('100%')
              .padding({ top: DesignConstants.SPACING.SM })
            EmptyState({ icon: AppIcon.MESSAGE, message: '没有找到会话', bare: true })
              .margin({ top: DesignConstants.SPACING.XXL })
          }
          .width('100%')
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadData(true, true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
        )
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          List() {
            // 过滤结果为空（对齐前端：无匹配会话空态）
            if (this.filteredChatItems.length === 0) {
              ListItem() {
                EmptyState({ icon: AppIcon.SEARCH, message: '没有找到会话', bare: true })
                  .margin({ top: DesignConstants.SPACING.XXL })
              }
            }

            // 在线头像条
            if (this.filteredChatItems.length > 0 && this.getOnlineChats().length > 0) {
              ListItem() {
                this.buildOnlineAvatarsStrip()
              }
            }

            if (this.filteredChatItems.length > 0) {
              // "最近" 标签
              ListItem() {
                Text('最近')
                  .fontSize(DesignConstants.FONT.SIZE.XS2)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .textCase(TextCase.UpperCase)
                  .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
                  .padding({
                    top: DesignConstants.SPACING.SM,
                    bottom: DesignConstants.SPACING.MD2
                  })
              }
            }

            // 会话列表（对齐前端 ListContainer 嵌入式：无外框卡片，仅行间分割线）
            // LazyForEach 按可视区虚拟化；行间分割线交给 ListItemGroup.divider
            // 承担（原 ForEach 内手写的分割线 Row 已删除）。ListItemGroup 只包会话行，
            // 避免 List 级 divider 在空态/在线条/「最近」标签之间误画线。
            if (this.filteredChatItems.length > 0) {
              ListItemGroup() {
                LazyForEach(this.filteredChatDataSource, (item: ChatItem) => {
                  ListItem() {
                    this.buildChatItem(item)
                  }
                  .padding({ top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.XS })
                }, (item: ChatItem) => item.id + item.type)
              }
              .width('100%')
              // 扁平会话列表（无卡片外框）：不留自身横向外边距/行内横向内边距，
              // 否则行内容会比页头多缩进（留白只由 PageShellContainer 提供）
              .divider({
                strokeWidth: DesignConstants.BORDER_WIDTH.SUBPIXEL,
                color: this.getSurfaceBorder(),
                startMargin: 0,
                endMargin: 0
              })
            }

            if (this.isLoadingMore) {
              ListItem() {
                LoadingMoreIndicator()
                  .margin({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
              }
            }
          }
          .width('100%')
          .padding({
            bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
          })
          .scrollBar(BarState.Off)
          .edgeEffect(EdgeEffect.Spring)
          // 与 ChatPage 会话/消息列表同档预加载
          .cachedCount(8)
          .onReachEnd(() => { this.loadMore(); })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadData(true, true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
        )
      }
    }
    .width('100%')
    .height('100%')
  }

  // 删除会话 / 退出群组二次确认
  SystemDialog({
    visible: this.showDeleteChatConfirm,
    title: this.pendingDeleteChat?.type === 'group' ? '退出群组' : '删除会话',
    description: this.pendingDeleteChat?.type === 'group' ?
      '退出后将不再接收该群组的消息' : '删除后该会话将从消息列表移除',
    confirmText: this.pendingDeleteChat?.type === 'group' ? '退出群组' : '删除',
    confirmVariant: 'danger',
    onCancel: () => {
      this.showDeleteChatConfirm = false;
      this.pendingDeleteChat = null;
    },
    onConfirm: () => {
      const target = this.pendingDeleteChat;
      this.showDeleteChatConfirm = false;
      this.pendingDeleteChat = null;
      if (target) {
        this.deleteChat(target);
      }
    }
  })
  }
  }

  @Builder
  buildOnlineAvatarsStrip() {
    // 对齐前端 OnlineStrip（lg）：eyebrow「在线」+ 52px 头像（2px accent 描边）+ 在线点 + 昵称
    Column() {
      Text('在线')
        .fontSize(DesignConstants.FONT.SIZE.XS2)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .textCase(TextCase.UpperCase)
        .letterSpacing(DesignConstants.FONT.LETTER_SPACING.EYEBROW)
        .padding({ bottom: DesignConstants.SPACING.MD2 })

      Scroll() {
        Row({ space: DesignConstants.SPACING.MD }) {
          ForEach(this.getOnlineChats(), (chat: ChatItem) => {
            Pressable({ onPress: () => {
              this.enterChat(chat);
            } }) {
              Column({ space: DesignConstants.SPACING.SM2 }) {
                // 48px 头像 + accent 描边环；在线点由 SystemAvatar 内建（isOnline）
                SystemAvatar({
                  avatarUrl: chat.avatarUrl,
                  name: chat.name,
                  avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM,
                  shape: 'circle',
                  isOnline: true
                })
                .border({ width: DesignConstants.BORDER_WIDTH.STRONG, color: this.getCurrentThemePalette().accentBackground })
                .borderRadius(DesignConstants.RADIUS.FULL)

                Text(chat.name)
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .maxLines(1)
                  .textOverflow({ overflow: TextOverflow.Ellipsis })
                  .constraintSize({ maxWidth: DesignConstants.LAYOUT.NICKNAME_MAX_WIDTH })
              }
              .alignItems(HorizontalAlign.Center)
            }
          }, (chat: ChatItem) => chat.id + chat.type)
        }

      }
      .scrollable(ScrollDirection.Horizontal)
      .scrollBar(BarState.Off)
      .constraintSize({ maxHeight: DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM + DesignConstants.SPACING.MD3 })
    }
    .width('100%')
    .padding({ top: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.MD })
  }

  // 私聊对端在线（群聊无 presence 语义，恒 false；对齐前端 ChatListItem online 点）
  private isChatOnline(item: ChatItem): boolean {
    if (item.type !== 'private') return false;
    return (item.data as PrivateChatItem).presence_status === 'online';
  }

  @Builder
  buildChatItem(item: ChatItem) {
    Row({ space: DesignConstants.SPACING.MD }) {
      // 头像 44 + hairline 描边 + 在线角标（对齐前端 ChatListItem 契约）
      SystemAvatar({
        avatarUrl: item.avatarUrl,
        name: item.name,
        avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL_PLUS,
        shape: item.type === 'group' ? 'squircle' : 'circle',
        isOnline: this.isChatOnline(item)
      })
      .border({ width: DesignConstants.BORDER_WIDTH.REGULAR, color: this.getCurrentThemePalette().borderColor })
      .borderRadius(DesignConstants.RADIUS.FULL)

      Column({ space: DesignConstants.SPACING.HALF }) {
        Row() {
          Text(item.name)
            .fontSize(DesignConstants.FONT.SIZE.SM) // 12 对齐前端 text-body
            .fontWeight(DesignConstants.FONT.WEIGHT.REGULAR)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .layoutWeight(1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(this.formatTime(item.lastTime))
            .fontSize(DesignConstants.FONT.SIZE.XS) // 11 对齐前端 text-caption
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
        .width('100%')

        Row() {
          Text(item.lastMessage || '')
            .fontSize(DesignConstants.FONT.SIZE.XS) // 11 对齐前端 text-xs 预览
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .layoutWeight(1)

          if (item.isPinned) {
            LucideIcon({
              icon: AppIcon.PIN_FILL,
              iconSize: 12,
              paletteRole: 'accent'
            })
              .margin({ left: DesignConstants.SPACING.XS })
          }
        }
        .width('100%')
      }
      .layoutWeight(1)

      // 行尾未读徽章（对齐前端 ChatListItem 22cbd552 契约：inline-flex + h-5 + leading-none
      // + tabular-nums，双轴居中，无光晕——前端徽标无 accent glow）
      Stack({ alignContent: Alignment.Center }) {
        if (item.unreadCount > 0) {
          Text(item.unreadCount > 99 ? '99+' : `${item.unreadCount}`)
            .fontSize(DesignConstants.FONT.SIZE.XS) // 11 对齐前端 text-xs
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getCurrentThemePalette().accentText)
            .minFontScale(0.5)
            .maxLines(1)
            .lineHeight(DesignConstants.FONT.SIZE.XS) // leading-none：行高锁字号防下沉
        }
      }
      .width(DesignConstants.LAYOUT.UNREAD_BADGE_SIZE)
      .height(DesignConstants.LAYOUT.UNREAD_BADGE_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(item.unreadCount > 0 ? this.getCurrentThemePalette().accentBackground : DesignConstants.COLOR.TRANSPARENT)
    }
    .width('100%')
    // 行内 padding（对齐前端 chat-list 10px/16px）；圆角只在按压/选中态出现，
    // 常态背景由 getChatItemBackground 返回透明（对齐前端 ChatListBody 的
    // 「无外框 + divide-y 平铺列表」），不再每行自带一块圆角底色
    .padding({
      top: DesignConstants.SPACING.MD,
      bottom: DesignConstants.SPACING.MD
    })
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
    .backgroundColor(this.getChatItemBackground(item))
    .opacity(this.isChatDeleting(item) ? DesignConstants.OPACITY.LOADING : 1)
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedChatKey = this.getChatKey(item);
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedChatKey = '';
      }
    })
    .onClick(() => {
      if (this.isChatDeleting(item)) {
        return;
      }
      this.enterChat(item);
    })
    // 长按弹上下文菜单（原 bindMenu 点击即弹，与行 onClick「点一下同时进聊天+弹菜单」冲突）
    .bindContextMenu(this.buildChatRowMenu(item), ResponseType.LongPress)
  }

  @Builder
  private buildChatRowMenu(item: ChatItem) {
    Menu() {
      MenuItem({
        content: item.isPinned ? '取消置顶' : '置顶'
      })
        .onClick(() => { this.togglePin(item); })
      MenuItem({
        content: item.type === 'group' ? '退出群组' : '删除'
      })
        .onClick(() => {
          this.pendingDeleteChat = item;
          this.showDeleteChatConfirm = true;
        })
    }
  }
}