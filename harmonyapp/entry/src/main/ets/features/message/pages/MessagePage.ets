/**
 * 消息列表页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 * LazyForEach + RDB 缓存 + swipeAction
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_LAST_MESSAGE_SYNC_TIME } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { AppRouter, ChatRouteParam } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { PageNames } from '../../../constants/AppConstants';
import {
  PrivateChatItem,
  GroupItem,
  MyProfile,
  MyGroupsRequest,
  LeaveGroupRequest,
  DeletePrivateChatRequest
} from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { EmptyState, LoadingMoreIndicator, RefreshIndicator } from '../../../components/EmptyState';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { MessageCacheService, ChatCacheItem } from '../../../core/database/MessageCacheService';
import { VibrationUtil } from '../../../core/utils/VibrationUtil';
import { scanBarcode, scanCore } from '@kit.ScanKit';
import { common } from '@kit.AbilityKit';
import pasteboard from '@ohos.pasteboard';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { InteractionStateStore } from '../../../core/state/InteractionStateStore';
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
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_LAST_MESSAGE_SYNC_TIME) @Watch('onMessageSync') lastMessageSyncTime: number = 0;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private chatListDataSource: BasicDataSource<ChatItem> = new BasicDataSource([]);
  private messageCacheService = MessageCacheService.getInstance();

  private getHeaderTopPadding(): number {
    return SafeAreaUtils.resolveTopPadding(this.safeAreaTop);
  }

  @State isLoading: boolean = true;
  @State isRefreshing: boolean = false;
  @State errorMessage: string = '';
  @State hasData: boolean = false;
  // 分页状态：私聊
  private privateHasMore: boolean = false;
  private privateNextCursor: string = '';
  // 分页状态：群聊
  private groupHasMore: boolean = false;
  private groupNextCursor: string = '';
  @State isLoadingMore: boolean = false;
  @State deletingChatKeys: string[] = [];
  @State pressedChatKey: string = '';

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
    // 先从缓存加载（避免白屏），再强制调接口刷新
    await this.loadFromCache();
    await this.loadData(true);
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

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private async loadData(forceRefresh: boolean = false, refresh: boolean = false) {
    if (!forceRefresh) {
      await this.loadFromCache();
    }

    if (!refresh) { this.isLoading = true; }
    this.errorMessage = '';

    try {
      const results = await LoadingUtil.withMinimumTime(() =>
        Promise.allSettled([this.loadPrivateChats(), this.loadGroupChats()])
      );
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
      this.hasData = allChats.length > 0;

      // 仅用户主动下拉刷新时触发振动反馈
      if (this.isRefreshing) {
        VibrationUtil.startSuccess();
      }

      this.updateCache(allChats);
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '加载消息失败');
      if (this.chatListDataSource.totalCount() === 0) {
        this.errorMessage = '加载消息失败，请检查网络';
        this.hasData = false;
      }
      if (this.isRefreshing) {
        VibrationUtil.startFail();
      }
    } finally {
      this.isLoading = false;
      this.isRefreshing = false;
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
              lastTime: item.last_activity_at,
              lastTimestamp: this.parseTimestamp(item.last_activity_at),
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
      const request: MyGroupsRequest = { pagination: { limit: 50, cursor: cursor } };
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
              lastMessage: '点击进入群聊',
              lastTime: item.last_activity_at || item.created_at || '',
              lastTimestamp: this.parseTimestamp(item.last_activity_at || item.created_at || ''),
              isPinned: item.my_state?.is_pinned ?? false,
              unreadCount: item.my_state?.unread_count || 0,
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
    return groupData.my_state?.is_pinned ?? false;
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
    if (a.isPinned !== b.isPinned) {
      return a.isPinned ? -1 : 1;
    }
    const timeDiff = b.lastTimestamp - a.lastTimestamp;
    if (timeDiff !== 0) {
      return timeDiff;
    }
    if (a.type !== b.type) {
      return a.type === 'private' ? -1 : 1;
    }
    const numberA = Number(a.id);
    const numberB = Number(b.id);
    if (Number.isFinite(numberA) && Number.isFinite(numberB) && numberA !== numberB) {
      return numberA - numberB;
    }
    return a.id.localeCompare(b.id);
  }

  private formatTime(timeStr: string): string {
    if (!timeStr) {
      return '';
    }
    const time = new Date(timeStr);
    if (!Number.isFinite(time.getTime())) {
      return '';
    }
    const now = new Date();
    if (time.toDateString() === now.toDateString()) {
      return `${time.getHours().toString().padStart(2, '0')}:${time.getMinutes().toString().padStart(2, '0')}`;
    }
    const yesterday = new Date(now);
    yesterday.setDate(yesterday.getDate() - 1);
    if (time.toDateString() === yesterday.toDateString()) {
      return '昨天';
    }
    if (time.getFullYear() === now.getFullYear()) {
      return `${(time.getMonth() + 1).toString().padStart(2, '0')}-${time.getDate().toString().padStart(2, '0')}`;
    }
    return time.toLocaleDateString();
  }

  private enterChat(item: ChatItem) {
    const targetUserId = item.type === 'private' ? (item.data as PrivateChatItem).other_user?.user_id ?? item.id : item.id;
    const chatParams = new ChatRouteParam();
    chatParams.chatId = targetUserId;
    chatParams.chatType = item.type === 'private' ? 'private_chat' : 'group_chat';
    chatParams.title = item.name;
    if (item.type === 'private') {
      chatParams.targetId = targetUserId;
    }
    AppRouter.push(PageNames.CHAT, chatParams);
  }

  private getChatKey(item: ChatItem): string {
    return `${item.type}:${item.id}`;
  }

  private isChatDeleting(item: ChatItem): boolean {
    return this.deletingChatKeys.indexOf(this.getChatKey(item)) >= 0;
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
        const request: LeaveGroupRequest = { group_id: groupId };
        await ApiClient.leaveGroup(request);
        InteractionStateStore.getInstance().updateGroupMembership(groupId, { is_member: false });
      }

      const index = this.chatListDataSource.getDataIndex(item);
      if (index >= 0) {
        this.chatListDataSource.deleteData(index);
      }
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

  build() {
    Column() {
      // 标题栏 - 扁平风格（白色背景）
      Row() {
        Text('消息')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)
          .padding({ left: DesignConstants.SPACING.LG })

        Row({ space: DesignConstants.SPACING.MD }) {
          // 搜索 - 触感木质拟物：轻微凸起
          Row() {
            Text() {
              SymbolSpan($r(AppIcon.SEARCH))
                .fontSize(22)
                .fontColor([this.getCurrentThemePalette().textPrimary])
            }
          }
          .width(40)
          .height(40)
          .justifyContent(FlexAlign.Center)
          .alignItems(VerticalAlign.Center)
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.MD)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: 1, color: this.getSurfaceBorder() })
          .onClick(() => {
            AppRouter.push(PageNames.SEARCH);
          })

          // 更多操作 - 触感木质拟物：轻微凸起
          Row() {
            Text() {
              SymbolSpan($r(AppIcon.ADD))
                .fontSize(22)
                .fontColor([this.getCurrentThemePalette().textPrimary])
            }
          }
          .width(40)
          .height(40)
          .justifyContent(FlexAlign.Center)
          .alignItems(VerticalAlign.Center)
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.RADIUS.MD)
          .shadow(this.getCurrentThemePalette().controlShadow)
          .border({ width: 1, color: this.getSurfaceBorder() })
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
        .padding({ right: DesignConstants.SPACING.LG })
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
      .padding({
        top: this.getHeaderTopPadding(),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor(Color.Transparent)
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      // 内容区域
      if (this.isLoading) {
        SystemLoadingView({ message: '收取信件中...' })
          .transition(
            TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
          )
      } else if (!this.hasData && this.errorMessage) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          EmptyState({ icon: AppIcon.WARNING, message: this.errorMessage, actionLabel: '重试', onAction: () => { this.loadData(true); } })
        }
        .onRefreshing(() => { this.loadData(true, true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      } else if (!this.hasData) {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          EmptyState({ icon: AppIcon.MESSAGE, message: '暂无消息' })
        }
        .onRefreshing(() => { this.loadData(true, true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: RefreshIndicator }) {
          List() {
            LazyForEach(this.chatListDataSource, (item: ChatItem) => {
              ListItem() {
                this.buildChatItem(item)
              }
              .swipeAction({ end: this.SwipeActionEnd(item) })
            }, (item: ChatItem) => item.id + item.type)
            if (this.isLoadingMore) {
              ListItem() {
                LoadingMoreIndicator()
              }
            }
          }
          .width('100%')
          .padding({
            bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, 0)
          })
          .backgroundColor(Color.Transparent)
          .cachedCount(10)
          .onReachEnd(() => { this.loadMore(); })
        }
        .layoutWeight(1)
        .onRefreshing(() => { this.loadData(true, true); })
        .transition(
          TransitionEffect.OPACITY.animation({ duration: 320, curve: Curve.FastOutSlowIn })
            .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: 320, curve: Curve.FastOutSlowIn }))
        )
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(this.getCurrentThemePalette().pageBackground)
  }

  @Builder
  SwipeActionEnd(item: ChatItem) {
    Row() {
          Button({ type: ButtonType.Normal, stateEffect: true }) {
            Column() {
              Text(this.isChatDeleting(item) ? '处理中...' : '删除')
            .fontColor(this.getCurrentThemePalette().accentText)
            .fontSize(DesignConstants.FONT.SIZE.LG)
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      }
      .backgroundColor(this.getCurrentThemePalette().accentBackground)
      .width(80)
      .height('100%')
      .borderRadius(DesignConstants.RADIUS.MD)
      .opacity(this.isChatDeleting(item) ? 0.62 : 1)
      .enabled(!this.isChatDeleting(item))
      .onClick(() => this.deleteChat(item))
    }
    .height('100%')
    .justifyContent(FlexAlign.End)
  }

  @Builder
  buildChatItem(item: ChatItem) {
    Column() {
      Row() {
        Badge({
          count: item.unreadCount,
          maxCount: 99,
          position: BadgePosition.RightTop,
          style: { badgeSize: 16, badgeColor: this.getCurrentThemePalette().accentBackground, fontSize: 10 }
        }) {
          SystemAvatar({
            avatarUrl: item.avatarUrl,
            name: item.name,
            avatarSize: 50,
            shape: item.type === 'group' ? 'squircle' : 'circle',
            inset: true
          })
        }
        .margin({ right: DesignConstants.SPACING.LG })

        Column({ space: 4 }) {
          Row() {
            Text(item.name)
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontWeight(FontWeight.Medium)
              .fontColor(this.getCurrentThemePalette().textPrimary)
              .layoutWeight(1)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })

            Text(this.formatTime(item.lastTime))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.getCurrentThemePalette().textSecondary)
          }
          .width('100%')

          Text(item.lastMessage)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .width('100%')
        }
        .layoutWeight(1)
      }
      .width('100%')
    }
    .constraintSize({ minHeight: 84 })
    .padding(22)
    .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: 8, bottom: 8 })
    .backgroundColor(this.pressedChatKey === this.getChatKey(item)
      ? this.getSurfaceSubtle()
      : this.getSurfaceBackground())
    .borderRadius(DesignConstants.RADIUS.XXL + 4)
    .shadow(this.getCurrentThemePalette().elevatedShadow)
    .border({ width: 1, color: this.getSurfaceBorder() })
    .opacity(this.isChatDeleting(item) ? 0.7 : 1)
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
  }
}
