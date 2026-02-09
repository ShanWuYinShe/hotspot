/**
 * 群组详情页面
 * 重构：使用 HarmonyOS 原生 List 组件
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { MyProfileData } from '../../../core/types/api';
import { ApiClient } from '../../../core/api/ApiClient';
import { PageNames } from '../../../constants/AppConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { IconButton, AppIcon, IconSize } from '../../../components/SystemIcon';
import { DesignConstants } from '../../../constants/DesignConstants';
import { 
  GroupSearchItem, 
  GroupMemberItem
} from '../../../core/types/api';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DistributedSyncService, SyncData } from '../../../core/services/DistributedSyncService';
import { SmartNotificationService } from '../../../core/services/SmartNotificationService';
import { AccessibilityManager } from '../../../core/accessibility/AccessibilityManager';
import { AuthAccountManager } from '../../../core/state/AuthAccountManager';

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

@Entry
@Component
struct GroupDetailPage {
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfileData | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  
  private stateManager = GlobalStateManager.getInstance();
  private distributedSyncService = DistributedSyncService.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  @State groupInfo: GroupSearchItem | null = null;
  @State members: GroupMemberItem[] = [];
  @State isLoading: boolean = true;
  @State isLoadingMembers: boolean = false;
  @State errorMessage: string = '';
  @State showMemberList: boolean = false;
  @State isJoining: boolean = false;
  @State isLeaving: boolean = false;
  @State joinPassword: string = '';
  private groupId: string = '';

  @State isMember: boolean = false;

  private ensureLoggedIn(actionName: string): boolean {
    if (this.currentUser) return true;

    SystemToast.showWarning(`请先登录后再${actionName}`, this.getUIContext());
    try {
      const routerInstance = this.getUIContext().getRouter();
      routerInstance.pushUrl({
        url: `features/auth/pages/${PageNames.LOGIN}`,
        params: { reason: 'action_required', action: actionName }
      });
    } catch (error) {
      // ignore
    }
    return false;
  }

  async aboutToAppear() {
    try {
      const routerInstance = this.getUIContext().getRouter();
      const params = routerInstance.getParams() as GroupDetailParams;
      if (params && params.groupId) {
        this.groupId = params.groupId;
        await this.loadGroupDetail();
      } else {
        this.errorMessage = '缺少群组ID参数';
      }
    } catch (error) {
      this.errorMessage = '初始化失败';
    }
  }

  private async loadGroupDetail(): Promise<void> {
    this.isLoading = true;
    this.errorMessage = '';

    try {
      const lat = this.currentLocation?.latitude || 0;
      const lng = this.currentLocation?.longitude || 0;

      const response = await ApiClient.getGroupDetail({
        group_id: this.groupId,
        latitude: lat,
        longitude: lng
      });

      if (response.code === 0 && response.data) {
        this.groupInfo = response.data;
        await this.syncGroupData();
        await this.accessibilityManager.announceText(`已加载群组：${this.groupInfo.name}`);
      } else {
        this.errorMessage = response.message || '加载群组详情失败';
      }
    } catch (error) {
      this.errorMessage = '加载群组详情失败';
    } finally {
      this.isLoading = false;
    }
  }

  private async loadGroupMembers(): Promise<void> {
    if (!this.groupInfo) return;

    this.isLoadingMembers = true;
    try {
      const response = await ApiClient.getGroupMembers({
        group_id: this.groupId,
        limit: 100
      });

      if (response.code === 0 && response.data) {
        this.members = response.data.items;
        this.showMemberList = true;
      } else {
        SystemToast.showError(response.message || '加载成员列表失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('加载成员列表失败', this.getUIContext());
    } finally {
      this.isLoadingMembers = false;
    }
  }

  private async joinGroup(): Promise<void> {
    if (!this.groupInfo) return;
    if (!this.ensureLoggedIn('加入群组')) return;

    if (this.groupInfo.has_password && !this.joinPassword.trim()) {
      SystemToast.showWarning('请输入群组密码', this.getUIContext());
      return;
    }

    this.isJoining = true;
    try {
      const response = await ApiClient.joinGroup({
        group_id: this.groupId,
        password: this.groupInfo.has_password ? this.joinPassword.trim() : undefined
      });

      if (response.code === 0) {
        SystemToast.showSuccess('加入群组成功', this.getUIContext());
        this.isMember = true;
        await this.loadGroupDetail();
      } else if (response.code === 1305) {
        SystemToast.show('您已是该群组成员', this.getUIContext());
        this.isMember = true;
        await this.enterChat();
      } else {
        SystemToast.showError(response.message || '加入群组失败', this.getUIContext());
      }
    } catch (error) {
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
      const response = await ApiClient.leaveGroup({
        group_id: this.groupId
      });

      if (response.code === 0) {
        SystemToast.showSuccess('已离开群组', this.getUIContext());
        this.isMember = false;
        const routerInstance = this.getUIContext().getRouter();
        routerInstance.back();
      } else {
        SystemToast.showError(response.message || '离开群组失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('离开群组失败', this.getUIContext());
    } finally {
      this.isLeaving = false;
    }
  }

  private async enterChat(): Promise<void> {
    if (!this.groupInfo) return;
    if (!this.ensureLoggedIn('进入群聊')) return;
    try {
      const routerInstance = this.getUIContext().getRouter();
      await routerInstance.pushUrl({
        url: `features/chat/pages/${PageNames.CHAT}`,
        params: {
          chatId: this.groupId,
          chatType: 'group_chat',
          title: this.groupInfo.name
        }
      });
    } catch (error) {
      SystemToast.showError('跳转聊天页面失败', this.getUIContext());
    }
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
      case 'admin': return DesignConstants.COLOR.BRAND_PRIMARY;
      case 'member': return DesignConstants.COLOR.TEXT_SECONDARY_LIGHT;
      default: return DesignConstants.COLOR.TEXT_SECONDARY_LIGHT;
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
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `同步群组数据失败: ${JSON.stringify(error)}`);
    }
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text('群组详情')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .height(56)
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

      if (this.isLoading) {
        SystemLoadingView({ message: '加载群组信息中...' }).layoutWeight(1)
      } else if (this.errorMessage) {
        Column() {
          Text('😕').fontSize(48).margin({ bottom: DesignConstants.SPACING.MD })
          Text(this.errorMessage)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .textAlign(TextAlign.Center)
            .margin({ bottom: DesignConstants.SPACING.XL })
          Button('重试')
            .type(ButtonType.Capsule)
            .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
            .fontColor(Color.White)
            .onClick(() => this.loadGroupDetail())
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .padding(DesignConstants.SPACING.XL)
      } else if (this.groupInfo) {
        // 使用 List 布局
        List({ space: 12 }) {
          // 1. 基本信息卡片
          ListItem() {
            Column() {
              // 头像和名称
              Column() {
                SystemAvatar({
                  src: this.groupInfo?.avatar_url,
                  name: this.groupInfo?.name,
                  avatarSize: 80
                })
                Text(this.groupInfo?.name || '')
                  .fontSize(DesignConstants.FONT.SIZE.XL)
                  .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
                  .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                  .margin({ top: DesignConstants.SPACING.MD })
                
                if (this.groupInfo?.description) {
                  Text(this.groupInfo?.description)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
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
                  Text(this.groupInfo?.members_count.toString())
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
                    .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                  Text('成员')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
                    .margin({ top: 4 })
                }
                .layoutWeight(1)

                Column() {
                  Text(this.groupInfo?.location?.distance ? `${Math.round(this.groupInfo.location.distance)}m` : '--')
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
                    .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                  Text('距离')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
                    .margin({ top: 4 })
                }
                .layoutWeight(1)
              }
              .width('100%')
              .padding({ bottom: DesignConstants.SPACING.XL })
              .justifyContent(FlexAlign.SpaceAround)
            }
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: 16, right: 16 })

          // 2. 位置信息
          ListItem() {
             Column() {
                Row() {
                  Image($r('sys.media.ohos_ic_public_location'))
                    .width(20)
                    .height(20)
                    .fillColor(DesignConstants.COLOR.BRAND_PRIMARY)
                  Text('位置信息')
                    .fontSize(DesignConstants.FONT.SIZE.LG)
                    .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                    .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                    .margin({ left: DesignConstants.SPACING.SM })
                }
                .width('100%')
                .margin({ bottom: DesignConstants.SPACING.MD })

                Text(this.groupInfo?.location?.location_name || '未知位置')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
                  .width('100%')

                if (this.groupInfo?.location) {
                  Text(`${this.groupInfo.location.latitude.toFixed(6)}, ${this.groupInfo.location.longitude.toFixed(6)}`)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
                    .width('100%')
                    .margin({ top: 4 })
                }
             }
             .padding(DesignConstants.SPACING.LG)
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: 16, right: 16 })

          // 3. 成员列表入口
          ListItem() {
            Row() {
              Column() {
                 Text('群组成员')
                  .fontSize(DesignConstants.FONT.SIZE.LG)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
                 Text(`${this.groupInfo?.members_count} 名成员`)
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
                  .margin({ top: 4 })
              }
              .layoutWeight(1)
              .alignItems(HorizontalAlign.Start)

              Text('查看全部 >')
                .fontSize(DesignConstants.FONT.SIZE.MD)
                .fontColor(DesignConstants.COLOR.BRAND_PRIMARY)
                .onClick(() => this.loadGroupMembers())
            }
            .width('100%')
            .padding(DesignConstants.SPACING.LG)
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: 16, right: 16 })

          // 4. 操作按钮
          ListItem() {
             Column() {
                if (this.isMember) {
                  Button('进入聊天')
                    .type(ButtonType.Capsule)
                    .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
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
                      .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
                      .borderRadius(DesignConstants.RADIUS.MD)
                      .margin({ bottom: DesignConstants.SPACING.MD })
                  }

                  Button(this.groupInfo?.has_password ? '加入群组（需要密码）' : '加入群组')
                    .type(ButtonType.Capsule)
                    .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
                    .fontColor(Color.White)
                    .width('100%')
                    .height(DesignConstants.COMPONENT.BUTTON_HEIGHT)
                    .enabled(!this.isJoining)
                    .onClick(() => this.joinGroup())
                }
             }
             .padding(DesignConstants.SPACING.LG)
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: 16, right: 16, bottom: 24 })

        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
    .bindSheet($$this.showMemberList, this.buildMemberListSheet(), {
      height: '70%',
      // 允许拖拽/点击外部关闭时，不显示右上角关闭按钮（避免冗余）
      showClose: false,
      dragBar: true,
      backgroundColor: DesignConstants.COLOR.SECONDARY_BG_LIGHT
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
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)

        Text(`${this.members.length} 人`)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
      }
      .width('100%')
      .padding(DesignConstants.SPACING.LG)

      // 成员列表
      if (this.isLoadingMembers) {
        SystemLoadingView({ message: '加载成员列表中...' }).layoutWeight(1)
      } else if (this.members.length > 0) {
        List() {
          ForEach(this.members, (member: GroupMemberItem) => {
            ListItem() {
              this.buildMemberItem(member)
            }
          })
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })
        .scrollBar(BarState.Auto)
      } else {
        Column() {
          Text('暂无成员信息')
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder
  buildMemberItem(member: GroupMemberItem) {
    Row() {
      SystemAvatar({
        src: member.user.avatar_url,
        name: member.user.nickname,
        avatarSize: 40,
        userInfo: member.user
      })

      Column() {
        Text(member.user.nickname)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row() {
          Text(this.getRoleDisplayText(member.role))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getRoleColor(member.role))
            .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
            .padding({ left: 6, right: 6, top: 2, bottom: 2 })
            .borderRadius(4)

          if (member.joined_at) {
            Text(`加入于 ${new Date(member.joined_at).toLocaleDateString()}`)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
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
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
    .borderRadius(DesignConstants.RADIUS.MD)
    .margin({ bottom: DesignConstants.SPACING.SM })
  }
}
