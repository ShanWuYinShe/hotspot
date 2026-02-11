/**
 * 群组详情页面
 * 重构：使用 HarmonyOS 原生 List 组件
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { MyProfile, GeoPoint } from '../../../core/types/api';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames } from '../../../constants/AppConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { IconButton, AppIcon, IconSize, SystemIcon } from '../../../components/SystemIcon';
import { SystemShareService } from '../../../core/services/SystemShareService';
import { common } from '@kit.AbilityKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import {
  GroupItem,
  GroupMemberItem
} from '../../../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DistributedSyncService, SyncData } from '../../../core/services/DistributedSyncService';
import { SmartNotificationService } from '../../../core/services/SmartNotificationService';
import { AccessibilityManager } from '../../../core/accessibility/AccessibilityManager';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { BasicDataSource } from '../../../core/utils/BasicDataSource';
import { NfcShareService } from '../../../core/services/NfcShareService';
import { AppRouter, LoginRouteParam, ChatRouteParam } from '../../../core/router/AppRouter';

interface GroupSyncData {
  groupId: string;
  groupInfo: Object;
  members: Object[];
  lastViewed: number;
}

const DOMAIN_NUMBER: number = 0xFF17;
const TAG: string = 'GroupDetailPage';

interface GroupDetailParams {
  groupId: string;
}

@Component
export struct GroupDetailPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  
  private stateManager = GlobalStateManager.getInstance();
  private distributedSyncService = DistributedSyncService.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  @State groupInfo: GroupItem | null = null;
  @State members: GroupMemberItem[] = [];
  private membersDataSource: BasicDataSource<GroupMemberItem> = new BasicDataSource<GroupMemberItem>([]);
  @State isLoading: boolean = true;
  @State isLoadingMembers: boolean = false;
  @State errorMessage: string = '';
  @State showMemberList: boolean = false;
  @State membersHasMore: boolean = false;
  @State membersNextCursor: string = '';
  @State membersLoadingMore: boolean = false;
  @State isJoining: boolean = false;
  @State isLeaving: boolean = false;
  @State joinPassword: string = '';
  private groupId: string = '';

  @State isMember: boolean = false;

  private ensureLoggedIn(actionName: string): boolean {
    if (this.currentUser) return true;

    SystemToast.showWarning(`请先登录后再${actionName}`, this.getUIContext());
    const loginParams = new LoginRouteParam();
    loginParams.reason = 'action_required';
    loginParams.action = actionName;
    AppRouter.push(PageNames.LOGIN, loginParams);
    return false;
  }

  private async loadGroupDetail(): Promise<void> {
    this.isLoading = true;
    this.errorMessage = '';

    try {
      await LoadingUtil.withMinimumTime(async () => {
        let referencePoint: GeoPoint = { latitude: 0, longitude: 0 };
        if (this.currentLocation) {
          referencePoint = { latitude: this.currentLocation.latitude, longitude: this.currentLocation.longitude };
        }
        const response = await ApiClient.getGroupDetail({
          group_id: this.groupId,
          reference_point: referencePoint
        });

        if (response.code === 0 && response.data) {
          this.groupInfo = response.data.group;
          this.isMember = this.groupInfo.my_state?.is_joined ?? false;
          await this.syncGroupData();
          await this.accessibilityManager.announceText(`已加载群组：${this.groupInfo.group?.name ?? ''}`);
        } else {
          this.errorMessage = response.message || '加载群组详情失败';
        }
      });
    } catch {
      this.errorMessage = '加载群组详情失败';
    } finally {
      this.isLoading = false;
    }
  }

  private async loadGroupMembers(): Promise<void> {
    if (!this.groupInfo) return;

    this.isLoadingMembers = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const response = await ApiClient.getGroupMembers({
          group_id: this.groupId,
          pagination: { limit: 100 }
        });

        if (response.code === 0 && response.data) {
          this.members = response.data.items;
          this.membersDataSource.setData(this.members);
          this.membersHasMore = response.data.pagination?.has_more ?? false;
          this.membersNextCursor = response.data.pagination?.next_cursor ?? '';
          this.showMemberList = true;
        } else {
          SystemToast.showError(response.message || '加载成员列表失败', this.getUIContext());
        }
      });
    } catch {
      SystemToast.showError('加载成员列表失败', this.getUIContext());
    } finally {
      this.isLoadingMembers = false;
    }
  }

  private async loadMoreMembers(): Promise<void> {
    if (this.membersLoadingMore || !this.membersHasMore) return;
    this.membersLoadingMore = true;
    try {
      const response = await ApiClient.getGroupMembers({
        group_id: this.groupId,
        pagination: { limit: 100, cursor: this.membersNextCursor }
      });
      if (response.code === 0 && response.data) {
        this.members = this.members.concat(response.data.items);
        this.membersDataSource.appendData(response.data.items);
        this.membersHasMore = response.data.pagination?.has_more ?? false;
        this.membersNextCursor = response.data.pagination?.next_cursor ?? '';
      }
    } catch {
      // 静默失败
    } finally {
      this.membersLoadingMore = false;
    }
  }

  private async joinGroup(): Promise<void> {
    if (!this.groupInfo) return;
    if (!this.ensureLoggedIn('加入群组')) return;

    if (this.groupInfo.has_password && !this.joinPassword.trim()) {
      SystemToast.showWarning('请输入群组密码', this.getUIContext());
      return;
    }

    const groupInfoRef = this.groupInfo;
    this.isJoining = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const response = await ApiClient.joinGroup({
          group_id: this.groupId,
          password: groupInfoRef.has_password ? this.joinPassword.trim() : undefined
        });

        if (response.code === 0 && response.data) {
          if (response.data.status === 'joined') {
            SystemToast.showSuccess('加入群组成功', this.getUIContext());
            this.isMember = true;
            await this.loadGroupDetail();
          } else if (response.data.status === 'pending_approval') {
            SystemToast.showSuccess('已提交申请，等待管理员审批', this.getUIContext());
          }
        } else if (response.code === 1305) {
          SystemToast.show('您已是该群组成员', this.getUIContext());
          this.isMember = true;
          await this.enterChat();
        } else {
          SystemToast.showError(response.message || '加入群组失败', this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch {
      SystemToast.showError('加入群组失败', this.getUIContext());
    } finally {
      this.isJoining = false;
    }
  }

  private async leaveGroup(): Promise<void> {
    if (!this.groupInfo) return;
    if (!this.ensureLoggedIn('离开群组')) return;

    this.isLeaving = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const response = await ApiClient.leaveGroup({
          group_id: this.groupId
        });

        if (response.code === 0) {
          SystemToast.showSuccess('已离开群组', this.getUIContext());
          this.isMember = false;
          AppRouter.pop();
        } else {
          SystemToast.showError(response.message || '离开群组失败', this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch {
      SystemToast.showError('离开群组失败', this.getUIContext());
    } finally {
      this.isLeaving = false;
    }
  }

  private async shareGroup(): Promise<void> {
    if (!this.groupInfo) return;
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const content = SystemShareService.buildGroupShareContent(
        this.groupInfo.group?.name ?? '',
        this.groupInfo.description,
        this.groupInfo.group?.group_id ?? this.groupId
      );
      await SystemShareService.getInstance().shareText(context, content);
    } catch {
      // User cancelled or share failed - non-fatal
    }
  }

  private async enterChat(): Promise<void> {
    if (!this.groupInfo) return;
    if (!this.ensureLoggedIn('进入群聊')) return;
    const chatParams = new ChatRouteParam();
    chatParams.chatId = this.groupId;
    chatParams.chatType = 'group_chat';
    chatParams.title = this.groupInfo.group?.name ?? '';
    AppRouter.push(PageNames.CHAT, chatParams);
  }

  private getRoleDisplayText(role: string): string {
    switch (role) {
      case 'owner': return '群主';
      case 'admin': return '管理员';
      case 'member': return '成员';
      default: return '未知';
    }
  }

  private getRoleColor(role: string): ResourceColor {
    switch (role) {
      case 'owner': return DesignConstants.COLOR.WARNING;
      case 'admin': return $r('sys.color.ohos_id_color_emphasize');
      case 'member': return $r('sys.color.ohos_id_color_text_secondary');
      default: return $r('sys.color.ohos_id_color_text_secondary');
    }
  }

  private async syncGroupData(): Promise<void> {
    if (!this.groupInfo) return;
    const userId = await AuthAccountManager.getCurrentUserId();
    if (!userId) return;
    try {
      const syncDataObject: GroupSyncData = {
        groupId: this.groupId,
        groupInfo: this.groupInfo,
        members: this.members,
        lastViewed: Date.now()
      };
      
      const syncData: SyncData = {
        userId: userId,
        dataType: 'groups',
        data: syncDataObject,
        timestamp: Date.now(),
        deviceId: 'current_device'
      };
      
      await this.distributedSyncService.syncUserData(syncData);
    } catch {
      hilog.error(DOMAIN_NUMBER, TAG, '同步群组数据失败');
    }
  }

  build() {
    NavDestination() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => {
            AppRouter.pop();
          }
        })

        Text('群组详情')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        SystemIcon({
          icon: AppIcon.SHARE,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          clickable: true,
          onIconClick: () => this.shareGroup()
        })
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      if (this.isLoading) {
        SystemLoadingView({ message: '加载群组信息中...' })
          .layoutWeight(1)
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.errorMessage) {
        Column() {
          Text('😕').fontSize(48).margin({ bottom: DesignConstants.SPACING.MD })
          Text(this.errorMessage)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
            .margin({ bottom: DesignConstants.SPACING.XL })
          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .onClick(() => this.loadGroupDetail())
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .padding(DesignConstants.SPACING.XL)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      } else if (this.groupInfo) {
        // 使用 List 布局
        List({ space: 12 }) {
          // 1. 基本信息卡片
          ListItem() {
            Column() {
              // 头像和名称
              Column() {
                SystemAvatar({
                  src: this.groupInfo?.group?.avatar_url ?? '',
                  name: this.groupInfo?.group?.name ?? '',
                  avatarSize: 80
                })
                Text(this.groupInfo?.group?.name ?? '')
                  .fontSize(DesignConstants.FONT.SIZE.XL)
                  .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .margin({ top: DesignConstants.SPACING.MD })

                if (this.groupInfo?.description) {
                  Text(this.groupInfo?.description)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .textAlign(TextAlign.Center)
                    .maxLines(3)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                    .margin({ top: DesignConstants.SPACING.SM })
                }
              }
              .width('100%')
              .padding({ top: DesignConstants.SPACING.XL, bottom: DesignConstants.SPACING.XL })

              // 统计信息
              Row() {
                Column() {
                  Text((this.groupInfo?.group?.members_count ?? 0).toString())
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  Text('成员')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .margin({ top: 4 })
                }
                .layoutWeight(1)

                Column() {
                  Text(this.groupInfo?.location?.distance ? `${Math.round(this.groupInfo.location.distance)}m` : '--')
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  Text('距离')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .margin({ top: 4 })
                }
                .layoutWeight(1)
              }
              .width('100%')
              .padding({ bottom: DesignConstants.SPACING.XL })
              .justifyContent(FlexAlign.SpaceAround)
            }
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

          // 2. 位置信息
          ListItem() {
             Column() {
                Row() {
                  Image($r('sys.media.ohos_ic_public_location'))
                    .width(20)
                    .height(20)
                    .fillColor($r('sys.color.ohos_id_color_emphasize'))
                  Text('位置信息')
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    .margin({ left: DesignConstants.SPACING.SM })
                }
                .width('100%')
                .margin({ bottom: DesignConstants.SPACING.MD })

                Text(this.groupInfo?.location?.location_name || '未知位置')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .width('100%')

                if (this.groupInfo?.location?.point) {
                  Text(`${this.groupInfo.location.point.latitude.toFixed(6)}, ${this.groupInfo.location.point.longitude.toFixed(6)}`)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .width('100%')
                    .margin({ top: 4 })
                }
             }
             .padding(DesignConstants.SPACING.LG)
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

          // 3. 成员列表入口
          ListItem() {
            Row() {
              Column() {
                 Text('群组成员')
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                 Text(`${this.groupInfo?.group?.members_count ?? 0} 名成员`)
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .margin({ top: 4 })
              }
              .layoutWeight(1)
              .alignItems(HorizontalAlign.Start)

              Text('查看全部 >')
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontColor($r('sys.color.ohos_id_color_emphasize'))
                .onClick(() => this.loadGroupMembers())
            }
            .width('100%')
            .padding(DesignConstants.SPACING.LG)
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

          // 4. 操作按钮
          ListItem() {
             Column() {
                if (this.isMember) {
                  Button('进入聊天')
                    .type(ButtonType.Capsule)
                    .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontColor(Color.White)
                    .width('100%')
                    .height(DesignConstants.COMPONENT.BUTTON_HEIGHT)
                    .onClick(() => this.enterChat())
                    .margin({ bottom: DesignConstants.SPACING.MD })

                  Button('离开群组')
                    .type(ButtonType.Capsule)
                    .backgroundColor(Color.Transparent)
                    .fontColor(DesignConstants.COLOR.ERROR)
                    .border({ width: 1, color: DesignConstants.COLOR.ERROR })
                    .width('100%')
                    .height(DesignConstants.COMPONENT.BUTTON_HEIGHT)
                    .enabled(!this.isLeaving)
                    .onClick(() => this.leaveGroup())
                } else {
                  if (this.groupInfo?.has_password) {
                    TextInput({ placeholder: '请输入群组密码', text: this.joinPassword })
                      .type(InputType.Password)
                      .onChange((value: string) => {
                        this.joinPassword = value;
                      })
                      .fontSize(16)
                      .padding(16)
                      .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
                      .borderRadius(DesignConstants.RADIUS.MD)
                      .margin({ bottom: DesignConstants.SPACING.MD })
                  }

                  Button(this.groupInfo?.has_password ? '加入群组（需要密码）' : '加入群组')
                    .type(ButtonType.Capsule)
                    .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontColor(Color.White)
                    .width('100%')
                    .height(DesignConstants.COMPONENT.BUTTON_HEIGHT)
                    .enabled(!this.isJoining)
                    .onClick(() => this.joinGroup())
                }
             }
             .padding(DesignConstants.SPACING.LG)
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, bottom: 24 })

        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
        .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const param = context.pathInfo.param as Record<string, Object> | undefined;
      if (param && param.groupId) {
        this.groupId = param.groupId as string;
        (async () => {
          await this.loadGroupDetail();
        })();
      } else {
        this.errorMessage = '缺少群组ID参数';
      }
    })
    .bindSheet($$this.showMemberList, this.buildMemberListSheet(), {
      height: '70%',
      // 允许拖拽/点击外部关闭时，不显示右上角关闭按钮（避免冗余）
      showClose: false,
      dragBar: true,
      backgroundColor: $r('sys.color.ohos_id_color_sub_background')
    })
  }

  @Builder
  buildMemberListSheet() {
    Column() {
      // 标题
      Row() {
        Text('群组成员')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)

        Text(`${this.members.length} 人`)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      // 成员列表
      if (this.isLoadingMembers) {
        SystemLoadingView({ message: '加载成员列表中...' }).layoutWeight(1)
      } else if (this.members.length > 0) {
        List() {
          LazyForEach(this.membersDataSource, (member: GroupMemberItem) => {
            ListItem() {
              this.buildMemberItem(member)
            }
          }, (member: GroupMemberItem) => member.user?.user_id ?? member.joined_at ?? '')
          if (this.membersLoadingMore) {
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
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
        .scrollBar(BarState.Auto)
        .onReachEnd(() => { this.loadMoreMembers(); })
      } else {
        Column() {
          Text('暂无成员信息')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  buildMemberItem(member: GroupMemberItem) {
    Row() {
      SystemAvatar({
        src: member.user?.avatar_url ?? '',
        name: member.user?.nickname ?? '',
        avatarSize: 40,
        userInfo: member.user
      })

      Column() {
        Text(member.user?.nickname ?? '')
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row() {
          Text(this.getRoleDisplayText(member.role))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getRoleColor(member.role))
            .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
            .padding({ left: DesignConstants.SPACING.SM, right: DesignConstants.SPACING.SM, top: 2, bottom: 2 })
            .borderRadius(DesignConstants.SPACING.XS)

          if (member.joined_at) {
            Text(`加入于 ${new Date(member.joined_at).toLocaleDateString()}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .margin({ left: 8 })
          }
        }
        .margin({ top: 4 })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
      .margin({ left: DesignConstants.SPACING.MD })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.MD)
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    .borderRadius(DesignConstants.RADIUS.MD)
    .margin({ bottom: DesignConstants.SPACING.SM })
  }
}

@Builder
export function GroupDetailPageBuilder() {
  GroupDetailPage()
}
