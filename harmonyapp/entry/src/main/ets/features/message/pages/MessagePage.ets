/**
 * 消息列表页面
 * 使用 HarmonyOS 原生能力 + 系统颜色资源（支持深色模式）
 * LazyForEach + RDB 缓存 + swipeAction
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_LAST_MESSAGE_SYNC_TIME } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames } from '../../../constants/AppConstants';
import { PrivateChatItem, GroupItem, MyProfile, MyGroupsRequest, DeletePrivateChatRequest, LeaveGroupRequest } from '../../../core/types/api';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, SystemIcon } from '../../../components/SystemIcon';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { MessageCacheService, ChatCacheItem } from '../../../core/database/MessageCacheService';
import { VibrationUtil } from '../../../core/utils/VibrationUtil';
import { scanBarcode, scanCore } from '@kit.ScanKit';
import { common } from '@kit.AbilityKit';
import pasteboard from '@ohos.pasteboard';

const DOMAIN_NUMBER: number = 0xFF08;
const TAG: string = 'MessagePage';

interface ChatItem {
  id: string;
  type: 'private' | 'group';
  name: string;
  avatarUrl?: string;
  lastMessage?: string;
  lastTime: string;
  unreadCount: number;
  data: PrivateChatItem | GroupItem;
}

@Component
export struct MessagePage {
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_LAST_MESSAGE_SYNC_TIME) @Watch('onMessageSync') lastMessageSyncTime: number = 0;

  private chatListDataSource: BasicDataSource<ChatItem> = new BasicDataSource([]);
  private messageCacheService = MessageCacheService.getInstance();

  @State isLoading: boolean = false;
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

  async aboutToAppear() {
    await this.loadData();
  }

  onMessageSync() {
    if (this.lastMessageSyncTime > 0) {
      hilog.info(DOMAIN_NUMBER, TAG, '收到消息同步通知，刷新列表');
      this.loadData(false);
    }
  }

  private async loadData(forceRefresh: boolean = false) {
    if (!forceRefresh) {
      await this.loadFromCache();
    }

    if (this.chatListDataSource.totalCount() === 0) {
      this.isLoading = true;
    }
    this.errorMessage = '';

    try {
      const privateChatsPromise = this.loadPrivateChats();
      const groupChatsPromise = this.loadGroupChats();
      const results = await Promise.allSettled([privateChatsPromise, groupChatsPromise]);
      const privateChatsResult = results[0];
      const groupChatsResult = results[1];

      let allChats: ChatItem[] = [];

      if (privateChatsResult.status === 'fulfilled' && privateChatsResult.value) {
        allChats.push(...privateChatsResult.value);
      }
      if (groupChatsResult.status === 'fulfilled' && groupChatsResult.value) {
        allChats.push(...groupChatsResult.value);
      }

      allChats.sort((a, b) => {
        const timeA = new Date(a.lastTime).getTime();
        const timeB = new Date(b.lastTime).getTime();
        return timeB - timeA;
      });

      this.chatListDataSource.setData(allChats);
      this.hasData = allChats.length > 0;

      if (forceRefresh) {
        VibrationUtil.startSuccess();
      }

      this.updateCache(allChats);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载消息失败: ${JSON.stringify(error)}`);
      if (this.chatListDataSource.totalCount() === 0) {
        this.errorMessage = '加载消息失败，请检查网络';
        this.hasData = false;
      }
      if (forceRefresh) {
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
        const chatItem: ChatItem = {
          id: item.id,
          type: item.type,
          name: item.name,
          avatarUrl: item.avatarUrl,
          lastMessage: item.lastMessage,
          lastTime: item.lastTime,
          unreadCount: item.unreadCount,
          data: JSON.parse(item.data)
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
      if (response.code === 0 && response.data) {
        this.privateHasMore = response.data.pagination?.has_more ?? false;
        this.privateNextCursor = response.data.pagination?.next_cursor ?? '';
        return response.data.items
          .filter((item: PrivateChatItem) => item.other_user?.user_id)
          .map((item: PrivateChatItem): ChatItem => {
            const chatItem: ChatItem = {
              id: item.other_user.user_id,
              type: 'private',
              name: item.other_user.nickname ?? '',
              avatarUrl: item.other_user.avatar_url,
              lastMessage: this.formatLastMessage(item.last_message),
              lastTime: item.last_activity_at,
              unreadCount: item.unread_count,
              data: item
            };
            return chatItem;
          });
      }
      return [];
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载私聊失败: ${error}`);
      return [];
    }
  }

  private async loadGroupChats(cursor?: string): Promise<ChatItem[]> {
    try {
      const request: MyGroupsRequest = { pagination: { limit: 50, cursor: cursor } };
      const response = await ApiClient.getJoinedGroups(request);
      if (response.code === 0 && response.data) {
        this.groupHasMore = response.data.pagination?.has_more ?? false;
        this.groupNextCursor = response.data.pagination?.next_cursor ?? '';
        return response.data.items
          .filter((item: GroupItem) => item.group?.group_id)
          .map((item: GroupItem): ChatItem => {
            const chatItem: ChatItem = {
              id: item.group.group_id,
              type: 'group',
              name: item.group.name ?? '',
              avatarUrl: item.group.avatar_url,
              lastMessage: '点击进入群聊',
              lastTime: item.last_activity_at || item.created_at || '',
              unreadCount: item.my_state?.unread_count || 0,
              data: item
            };
            return chatItem;
          });
      }
      return [];
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载群聊失败: ${error}`);
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
          newChats.push(...result.value);
        }
      }
      if (newChats.length > 0) {
        newChats.sort((a, b) => new Date(b.lastTime).getTime() - new Date(a.lastTime).getTime());
        const existingItems = this.getAllDataFromSource();
        const mergedItems = existingItems.concat(newChats);
        this.chatListDataSource.setData(mergedItems);
        this.updateCache(mergedItems);
      }
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `加载更多消息失败: ${error}`);
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

  private formatTime(timeStr: string): string {
    const time = new Date(timeStr);
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

  private async enterChat(item: ChatItem) {
    try {
      const routerInstance = this.getUIContext().getRouter();
      const targetUserId = item.type === 'private' ? (item.data as PrivateChatItem).other_user?.user_id ?? item.id : item.id;
      await routerInstance.pushUrl({
        url: `features/chat/pages/${PageNames.CHAT}`,
        params: {
          chatId: targetUserId,
          chatType: item.type === 'private' ? 'private_chat' : 'group_chat',
          title: item.name,
          targetId: item.type === 'private' ? targetUserId : undefined
        }
      });
    } catch (error) {
      SystemToast.showError('跳转失败', this.getUIContext());
    }
  }

  private async deleteChat(item: ChatItem): Promise<void> {
    try {
      VibrationUtil.startShort();
      if (item.type === 'private') {
        const privateData = item.data as PrivateChatItem;
        const userId = privateData.other_user?.user_id ?? item.id;
        const request: DeletePrivateChatRequest = { target_user_id: userId };
        await ApiClient.deletePrivateChat(request);
      } else {
        const groupData = item.data as GroupItem;
        const groupId = groupData.group?.group_id ?? item.id;
        const request: LeaveGroupRequest = { group_id: groupId };
        await ApiClient.leaveGroup(request);
      }

      const index = this.chatListDataSource.getDataIndex(item);
      if (index >= 0) {
        this.chatListDataSource.deleteData(index);
      }
      this.updateCache(this.getAllDataFromSource());
      SystemToast.show('操作成功', this.getUIContext());
    } catch (error) {
      SystemToast.showError('操作失败', this.getUIContext());
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
      // 标题栏
      Row() {
        Text('消息')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Bold)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .padding({ left: DesignConstants.SPACING.LG })

        Row({ space: DesignConstants.SPACING.LG }) {
          // 搜索
          Text() {
            SymbolSpan($r(AppIcon.SEARCH))
              .fontSize(24)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
          .onClick(() => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.pushUrl({ url: `features/search/pages/${PageNames.SEARCH}` });
          })

          // 更多操作
          Text() {
            SymbolSpan($r(AppIcon.ADD))
              .fontSize(24)
              .fontColor([$r('sys.color.ohos_id_color_text_primary')])
          }
          .bindMenu([
            {
              value: '发起群聊',
              action: () => {
                const routerInstance = this.getUIContext().getRouter();
                routerInstance.pushUrl({ url: `features/group/pages/${PageNames.CREATE_GROUP}` });
              }
            },
            {
              value: '添加朋友',
              action: () => {
                const routerInstance = this.getUIContext().getRouter();
                routerInstance.pushUrl({ url: `features/search/pages/${PageNames.SEARCH}` });
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
                } catch (error) {
                  hilog.error(DOMAIN_NUMBER, TAG, `扫描启动失败: ${JSON.stringify(error)}`);
                  const errorObj = error as Record<string, Object>;
                  const errorCode = errorObj.code as number | undefined;
                  if (errorCode !== 1001) {
                    SystemToast.showError('扫描启动失败', this.getUIContext());
                  }
                }
              }
            }
          ])
        }
        .padding({ right: DesignConstants.SPACING.LG })
      }
      .width('100%')
      .height(56)
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      // 内容区域
      if (this.isLoading && !this.hasData) {
        SystemLoadingView({ message: '收取信件中...' })
      } else if (!this.hasData && this.errorMessage) {
        Refresh({ refreshing: $$this.isRefreshing, builder: this.RefreshBuilder() }) {
          this.buildEmptyState(AppIcon.WARNING, this.errorMessage, '重试', () => { this.loadData(true); })
        }
        .onRefreshing(() => { this.loadData(true); })
      } else if (!this.hasData) {
        Refresh({ refreshing: $$this.isRefreshing, builder: this.RefreshBuilder() }) {
          this.buildEmptyState(AppIcon.MESSAGE, '暂无消息', undefined, undefined)
        }
        .onRefreshing(() => { this.loadData(true); })
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: this.RefreshBuilder() }) {
          List() {
            LazyForEach(this.chatListDataSource, (item: ChatItem) => {
              ListItem() {
                this.buildChatItem(item)
              }
              .swipeAction({ end: this.SwipeActionEnd(item) })
            }, (item: ChatItem) => item.id + item.type)
            if (this.isLoadingMore) {
              ListItem() {
                Row({ space: DesignConstants.SPACING.SM }) {
                  LoadingProgress().width(20).height(20)
                  Text('加载更多...')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                }
                .width('100%')
                .justifyContent(FlexAlign.Center)
                .padding(DesignConstants.SPACING.MD)
              }
            }
          }
          .width('100%')
          .layoutWeight(1)
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .divider({ strokeWidth: 0.5, color: $r('sys.color.ohos_id_color_list_separator'), startMargin: 78, endMargin: 0 })
          .cachedCount(10)
          .onReachEnd(() => { this.loadMore(); })
        }
        .onRefreshing(() => { this.loadData(true); })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  private buildEmptyState(icon: AppIcon, message: string, actionLabel: string | undefined, action: (() => void) | undefined) {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text() {
        SymbolSpan($r(icon))
          .fontSize(48)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }

      Text(message)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)

      if (actionLabel && action) {
        Button(actionLabel)
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .height(40)
          .onClick(() => { action(); })
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.XXL)
  }

  @Builder
  RefreshBuilder() {
    Row() {
      LoadingProgress().width(24).height(24)
      Text('正在刷新...')
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .margin({ left: DesignConstants.SPACING.SM })
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.SM)
  }

  @Builder
  SwipeActionEnd(item: ChatItem) {
    Row() {
      Button({ type: ButtonType.Normal, stateEffect: true }) {
        Column() {
          Text('删除')
            .fontColor(Color.White)
            .fontSize(DesignConstants.FONT.SIZE.LG)
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      }
      .backgroundColor('#FF3B30')
      .width(80)
      .height('100%')
      .onClick(() => this.deleteChat(item))
    }
    .height('100%')
    .justifyContent(FlexAlign.End)
  }

  @Builder
  buildChatItem(item: ChatItem) {
    Row() {
      // 头像 + 未读角标
      Stack({ alignContent: Alignment.TopEnd }) {
        SystemAvatar({
          avatarUrl: item.avatarUrl,
          name: item.name,
          avatarSize: 50,
          shape: item.type === 'group' ? 'squircle' : 'circle'
        })

        if (item.unreadCount > 0) {
          Text(item.unreadCount > 99 ? '99+' : item.unreadCount.toString())
            .fontSize(10)
            .fontColor(Color.White)
            .backgroundColor('#FF3B30')
            .borderRadius(10)
            .padding({ left: 4, right: 4 })
            .height(16)
            .constraintSize({ minWidth: 16 })
            .textAlign(TextAlign.Center)
            .offset({ x: 4, y: -4 })
        }
      }
      .margin({ right: DesignConstants.SPACING.MD })

      // 内容
      Column({ space: 4 }) {
        Row() {
          Text(item.name)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Medium)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(this.formatTime(item.lastTime))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .width('100%')

        Text(item.lastMessage)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      }
      .layoutWeight(1)
    }
    .width('100%')
    .height(72)
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .onClick(() => this.enterChat(item))
  }
}
