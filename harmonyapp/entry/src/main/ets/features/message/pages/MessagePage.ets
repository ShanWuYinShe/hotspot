/**
 * 消息列表页面
 * 重构：使用 LazyForEach 和 BasicDataSource 提升性能
 * 新增：滑动操作（删除）
 * 优化：使用 RDB 缓存实现离线浏览
 * 优化：使用原生菜单和震动反馈
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_LAST_MESSAGE_SYNC_TIME } from '../../../core/state/GlobalStateManager';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames } from '../../../constants/AppConstants';
import { PrivateChatListItem, JoinedGroupItem, MyProfileData, MyGroupsRequest, DeletePrivateChatRequest, LeaveGroupRequest } from '../../../core/types/api';
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
  data: PrivateChatListItem | JoinedGroupItem;
}

@Component
export struct MessagePage {
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfileData | null = null;
  @StorageLink(KEY_LAST_MESSAGE_SYNC_TIME) @Watch('onMessageSync') lastMessageSyncTime: number = 0;
  
  // 使用自定义 DataSource
  private chatListDataSource: BasicDataSource<ChatItem> = new BasicDataSource([]);
  private messageCacheService = MessageCacheService.getInstance();
  
  @State isLoading: boolean = false;
  @State isRefreshing: boolean = false;
  @State errorMessage: string = '';
  @State hasData: boolean = false;

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
    // 1. 先尝试加载缓存
    if (!forceRefresh) {
        await this.loadFromCache();
    }

    // 2. 发起网络请求
    if (this.chatListDataSource.totalCount() === 0) {
        this.isLoading = true;
    }
    this.errorMessage = '';

    try {
      // 并行加载私聊和群聊
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

      // 排序
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

      // 3. 异步更新缓存
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

  private async loadPrivateChats(): Promise<ChatItem[]> {
    try {
      const response = await ApiClient.getPrivateChats({ limit: 50 });
      if (response.code === 0 && response.data) {
        return response.data.items.map((item: PrivateChatListItem): ChatItem => {
          const chatItem: ChatItem = {
            id: item.other_user.user_id.toString(),
            type: 'private',
            name: item.other_user.nickname,
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

  private async loadGroupChats(): Promise<ChatItem[]> {
    try {
      const request: MyGroupsRequest = { limit: 50 };
      const response = await ApiClient.getJoinedGroups(request);
      if (response.code === 0 && response.data) {
        return response.data.items.map((item: JoinedGroupItem): ChatItem => {
          const chatItem: ChatItem = {
            id: item.group_id,
            type: 'group',
            name: item.name,
            avatarUrl: item.avatar_url,
            lastMessage: '点击进入群聊',
            lastTime: item.created_at || '',
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
    // 今天
    if (time.toDateString() === now.toDateString()) {
      return `${time.getHours().toString().padStart(2, '0')}:${time.getMinutes().toString().padStart(2, '0')}`;
    }
    // 昨天
    const yesterday = new Date(now);
    yesterday.setDate(yesterday.getDate() - 1);
    if (time.toDateString() === yesterday.toDateString()) {
      return '昨天';
    }
    // 今年
    if (time.getFullYear() === now.getFullYear()) {
      return `${(time.getMonth() + 1).toString().padStart(2, '0')}-${time.getDate().toString().padStart(2, '0')}`;
    }
    return time.toLocaleDateString();
  }

  private async enterChat(item: ChatItem) {
    try {
      const routerInstance = this.getUIContext().getRouter();
      await routerInstance.pushUrl({
        url: `features/chat/pages/${PageNames.CHAT}`,
        params: {
          chatId: item.type === 'private' ? (item.data as PrivateChatListItem).other_user.user_id.toString() : item.id, 
          chatType: item.type === 'private' ? 'private_chat' : 'group_chat',
          title: item.name,
          targetId: item.type === 'private' ? (item.data as PrivateChatListItem).other_user.user_id : undefined
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
        const privateData = item.data as PrivateChatListItem;
        const request: DeletePrivateChatRequest = { target_user_id: privateData.other_user.user_id };
        await ApiClient.deletePrivateChat(request);
      } else {
        const groupData = item.data as JoinedGroupItem;
        const request: LeaveGroupRequest = { group_id: groupData.group_id };
        await ApiClient.leaveGroup(request);
      }
      
      // 更新列表
      const index = this.chatListDataSource.getDataIndex(item);
      if (index >= 0) {
        this.chatListDataSource.deleteData(index);
      }
      // 更新缓存
      const currentData = this.chatListDataSource.getData(0) ? [] : []; 
      this.updateCache(this.getAllDataFromSource());

      SystemToast.show('操作成功', this.getUIContext());
    } catch (error) {
      SystemToast.showError('操作失败', this.getUIContext());
    }
  }
  
  private getAllDataFromSource(): ChatItem[] {
      const items: ChatItem[] = [];
      for(let i = 0; i < this.chatListDataSource.totalCount(); i++) {
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
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)
          .padding({ left: 16 })

        Row() {
          SystemIcon({ 
            icon: AppIcon.SEARCH, 
            iconSize: 24, 
            iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
            clickable: true,
            onIconClick: () => {
              const routerInstance = this.getUIContext().getRouter();
              routerInstance.pushUrl({ url: `features/search/pages/${PageNames.SEARCH}` });
            }
          })
          
          Blank().width(16)

          SystemIcon({ 
            icon: AppIcon.ADD, 
            iconSize: 24, 
            iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
            clickable: true,
            onIconClick: () => {
              // 占位，实际由 bindMenu 接管
            }
          })
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
                // 集成 Scan Kit
                try {
                  // 检查是否支持扫码能力
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
                      // 简单处理：复制到剪贴板并提示
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
                  // 错误码处理
                  const errorObj = error as Record<string, Object>;
                  const errorCode = errorObj.code as number | undefined;
                  if (errorCode === 1001) {
                     // 用户取消
                  } else {
                     SystemToast.showError('扫描启动失败', this.getUIContext());
                  }
                }
              }
            }
          ])
        }
        .padding({ right: 16 })
      }
      .width('100%')
      .height(56)
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

      if (this.isLoading && !this.hasData) {
        SystemLoadingView({ message: '收取信件中...' })
      } else if (!this.hasData && this.errorMessage) {
        Refresh({ refreshing: $$this.isRefreshing, builder: this.RefreshBuilder() }) {
          Column({ space: 16 }) {
            Text('⚠️')
               .fontSize(48)
            Text(this.errorMessage)
              .fontSize(16)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            Button('重试').onClick(() => this.loadData(true))
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
        }
        .onRefreshing(() => {
            this.loadData(true);
        })
      } else if (!this.hasData) {
        Refresh({ refreshing: $$this.isRefreshing, builder: this.RefreshBuilder() }) {
          Column({ space: 16 }) {
            Text('📭')
              .fontSize(48)
            Text('暂无消息')
              .fontSize(16)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
        }
        .onRefreshing(() => {
          this.loadData(true);
        })
      } else {
        Refresh({ refreshing: $$this.isRefreshing, builder: this.RefreshBuilder() }) {
          List() {
            LazyForEach(this.chatListDataSource, (item: ChatItem) => {
              ListItem() {
                this.buildChatItem(item)
              }
              .swipeAction({
                end: this.SwipeActionEnd(item)
              })
            }, (item: ChatItem) => item.id + item.type)
          }
          .width('100%')
          .layoutWeight(1)
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT) 
          .divider({ strokeWidth: 1, color: DesignConstants.COLOR.TERTIARY_BG_LIGHT, startMargin: 78, endMargin: 0 })
          .cachedCount(10)
        }
        .onRefreshing(() => {
          this.loadData(true);
        })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder
  RefreshBuilder() {
    Row() {
      LoadingProgress().width(24).height(24)
      Text('正在刷新...').fontSize(14).margin({left: 8}).fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(10)
  }

  @Builder
  SwipeActionEnd(item: ChatItem) {
    Row() {
      Button({ type: ButtonType.Normal, stateEffect: true }) {
        Column() {
          Text('删除')
            .fontColor(Color.White)
            .fontSize(16)
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
      }
      .backgroundColor(DesignConstants.COLOR.ERROR)
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
      // 头像
      Stack({ alignContent: Alignment.TopEnd }) {
        SystemAvatar({
          avatarUrl: item.avatarUrl,
          name: item.name,
          avatarSize: 50,
          shape: item.type === 'group' ? 'square' : 'circle'
        })

        if (item.unreadCount > 0) {
          Text(item.unreadCount > 99 ? '99+' : item.unreadCount.toString())
            .fontSize(10)
            .fontColor(Color.White)
            .backgroundColor(DesignConstants.COLOR.ERROR)
            .borderRadius(10)
            .padding({ left: 4, right: 4 })
            .height(16)
            .constraintSize({ minWidth: 16 })
            .textAlign(TextAlign.Center)
            .offset({ x: 4, y: -4 })
        }
      }
      .margin({ right: 12 })

      // 内容
      Column({ space: 4 }) {
        Row() {
          Text(item.name)
            .fontSize(16)
            .fontWeight(FontWeight.Medium)
            .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
            .layoutWeight(1)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(this.formatTime(item.lastTime))
            .fontSize(12)
            .fontColor(DesignConstants.COLOR.TEXT_TERTIARY_LIGHT)
        }
        .width('100%')

        Text(item.lastMessage)
          .fontSize(14)
          .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .width('100%')
      }
      .layoutWeight(1)
    }
    .width('100%')
    .height(72) // 固定高度
    .padding({ left: 16, right: 16 })
    .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
    .onClick(() => this.enterChat(item))
  }
}
