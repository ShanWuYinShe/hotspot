/**
 * 群组详情页面
 * 显示群组信息、成员列表、管理功能等
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { PageNames } from '../constants/AppConstants';
import { SystemAvatar } from '../components/SystemAvatar';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { SystemToast } from '../components/SystemToast';
import { IconButton, AppIcon, IconSize } from '../components/SystemIcon';
import { 
  SearchGroupByIdResponse, 
  GetMembersResponse, 
  isTempUser,
  hasUserPermission 
} from '../common/types/BackendApiTypes';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DistributedSyncService, SyncData } from '../common/services/DistributedSyncService';
import { SmartNotificationService } from '../common/services/SmartNotificationService';
import { AccessibilityManager } from '../common/accessibility/AccessibilityManager';

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
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  private distributedSyncService = DistributedSyncService.getInstance();
  private smartNotificationService = SmartNotificationService.getInstance();
  private accessibilityManager = AccessibilityManager.getInstance();
  @State groupInfo: SearchGroupByIdResponse | null = null;
  @State members: GetMembersResponse[] = [];
  @State isLoading: boolean = true;
  @State isLoadingMembers: boolean = false;
  @State errorMessage: string = '';
  @State showMemberList: boolean = false;
  @State isJoining: boolean = false;
  @State isLeaving: boolean = false;
  private groupId: string = '';

  async aboutToAppear() {
    hilog.info(DOMAIN_NUMBER, TAG, '群组详情页面初始化');
    
    // 订阅状态变化
    this.stateManager.subscribe('group_detail_page', (state: GlobalState) => {
      this.globalState = state;
    });

    // 获取路由参数
    try {
      const routerInstance = this.getUIContext().getRouter();
      const params = routerInstance.getParams() as GroupDetailParams;
      if (params && params.groupId) {
        this.groupId = params.groupId;
        await this.loadGroupDetail();
      } else {
        this.errorMessage = '缺少群组ID参数';
        hilog.error(DOMAIN_NUMBER, TAG, '缺少群组ID参数');
      }
    } catch (error) {
      this.errorMessage = '初始化失败';
      hilog.error(DOMAIN_NUMBER, TAG, `初始化失败: ${JSON.stringify(error)}`);
    }
  }

  aboutToDisappear() {
    this.stateManager.unsubscribe('group_detail_page');
  }

  /**
   * 加载群组详情
   */
  private async loadGroupDetail(): Promise<void> {
    this.isLoading = true;
    this.errorMessage = '';

    try {
      const response = await ApiClient.searchGroupById({
        group_id: this.groupId
      });

      if (response.code === 0 && response.data) {
        this.groupInfo = response.data;
        hilog.info(DOMAIN_NUMBER, TAG, `群组详情加载成功: ${this.groupInfo.name}`);
        
        // 同步群组数据到分布式存储
        await this.syncGroupData();
        
        
        // 无障碍播报
        await this.accessibilityManager.announceText(`已加载群组：${this.groupInfo.name}`);
        
      } else {
        this.errorMessage = response.msg || '加载群组详情失败';
        hilog.error(DOMAIN_NUMBER, TAG, `加载群组详情失败: ${response.msg}`);
      }
    } catch (error) {
      this.errorMessage = '加载群组详情失败';
      hilog.error(DOMAIN_NUMBER, TAG, `加载群组详情异常: ${JSON.stringify(error)}`);
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 加载群组成员
   */
  private async loadGroupMembers(): Promise<void> {
    if (!this.groupInfo) return;

    this.isLoadingMembers = true;
    try {
      const response = await ApiClient.getMembers({
        group_id: this.groupId,
        limit: 100
      });

      if (response.code === 0 && response.data) {
        this.members = response.data.items;
        this.showMemberList = true;
        hilog.info(DOMAIN_NUMBER, TAG, `群组成员加载成功: ${this.members.length}个成员`);
      } else {
        SystemToast.showError(response.msg || '加载成员列表失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('加载成员列表失败', this.getUIContext());
      hilog.error(DOMAIN_NUMBER, TAG, `加载群组成员失败: ${JSON.stringify(error)}`);
    } finally {
      this.isLoadingMembers = false;
    }
  }

  /**
   * 加入群组
   */
  private async joinGroup(): Promise<void> {
    if (!this.groupInfo || !this.globalState.currentUser) return;

    // 检查临时用户权限
    if (isTempUser(this.globalState.currentUser.user_id) && 
        !hasUserPermission(this.globalState.currentUser.user_id, 'join_group')) {
      SystemToast.showWarning('临时用户功能受限，请升级为正式用户', this.getUIContext());
      return;
    }

    this.isJoining = true;
    try {
      const response = await ApiClient.joinGroup({
        group_id: this.groupId,
        password: this.groupInfo.is_password_required ? undefined : undefined // 如果需要密码，应该弹出输入框
      });

      if (response.code === 0) {
        SystemToast.showSuccess('加入群组成功', this.getUIContext());
        // 重新加载群组信息
        await this.loadGroupDetail();
      } else {
        SystemToast.showError(response.msg || '加入群组失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('加入群组失败', this.getUIContext());
      hilog.error(DOMAIN_NUMBER, TAG, `加入群组失败: ${JSON.stringify(error)}`);
    } finally {
      this.isJoining = false;
    }
  }

  /**
   * 离开群组
   */
  private async leaveGroup(): Promise<void> {
    if (!this.groupInfo || !this.globalState.currentUser) return;

    this.isLeaving = true;
    try {
      const response = await ApiClient.leaveGroup({
        group_id: this.groupId
      });

      if (response.code === 0) {
        SystemToast.showSuccess('已离开群组', this.getUIContext());
        // 返回上一页
        const routerInstance = this.getUIContext().getRouter();
        routerInstance.back();
      } else {
        SystemToast.showError(response.msg || '离开群组失败', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('离开群组失败', this.getUIContext());
      hilog.error(DOMAIN_NUMBER, TAG, `离开群组失败: ${JSON.stringify(error)}`);
    } finally {
      this.isLeaving = false;
    }
  }

  /**
   * 进入聊天页面
   */
  private async enterChat(): Promise<void> {
    if (!this.groupInfo) return;

    try {
      const routerInstance = this.getUIContext().getRouter();
      await routerInstance.pushUrl({
        url: `pages/${PageNames.CHAT}`,
        params: {
          groupId: this.groupId,
          groupName: this.groupInfo.name,
          groupType: 'public_chat'
        }
      });
    } catch (error) {
      SystemToast.showError('跳转聊天页面失败', this.getUIContext());
      hilog.error(DOMAIN_NUMBER, TAG, `跳转聊天页面失败: ${JSON.stringify(error)}`);
    }
  }

  /**
   * 获取用户角色显示文本
   */
  private getRoleDisplayText(role: string): string {
    switch (role) {
      case 'owner': return '群主';
      case 'admin': return '管理员';
      case 'member': return '成员';
      default: return '未知';
    }
  }

  /**
   * 获取用户角色颜色
   */
  private getRoleColor(role: string): ResourceColor {
    switch (role) {
      case 'owner': return $r('sys.color.ohos_id_color_warning');
      case 'admin': return $r('sys.color.ohos_id_color_emphasize');
      case 'member': return $r('sys.color.ohos_id_color_text_secondary');
      default: return $r('sys.color.ohos_id_color_text_secondary');
    }
  }

  /**
   * 构建成员项
   */
  @Builder
  buildMemberItem(member: GetMembersResponse) {
    Row() {
      SystemAvatar({
        avatarUrl: member.avatar_url,
        nickname: member.nickname,
        avatarSize: 40
      })

      Column() {
        Text(member.nickname)
          .fontSize(16)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Row() {
          Text(this.getRoleDisplayText(member.user_role))
            .fontSize(12)
            .fontColor(this.getRoleColor(member.user_role))
            .backgroundColor($r('sys.color.ohos_id_color_background_transparent'))
            .padding({ left: 6, right: 6, top: 2, bottom: 2 })
            .borderRadius(4)

          if (member.joined_at) {
            Text(`加入于 ${new Date(member.joined_at).toLocaleDateString()}`)
              .fontSize(12)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .margin({ left: 8 })
          }
        }
        .margin({ top: 4 })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
      .margin({ left: 12 })
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(8)
    .margin({ bottom: 8 })
  }

  build() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
          }
        })

        Text('群组详情')
          .fontSize(18)
          .fontWeight(FontWeight.Medium)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        IconButton({
          icon: AppIcon.MORE,
          iconSize: IconSize.LARGE,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            SystemToast.show('更多功能待开发', this.getUIContext());
          }
        })
      }
      .width('100%')
      .height(56)
      .padding({ left: 16, right: 16 })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      if (this.isLoading) {
        SystemLoadingView({ message: '加载群组信息中...' })
          .layoutWeight(1)
      } else if (this.errorMessage) {
        // 错误状态
        Column() {
          Text('😕')
            .fontSize(48)
            .margin({ bottom: 16 })

          Text(this.errorMessage)
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .textAlign(TextAlign.Center)
            .margin({ bottom: 24 })

          Button('重试')
            .onClick(() => this.loadGroupDetail())
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
        .padding(24)
      } else if (this.groupInfo) {
        Scroll() {
          Column() {
            // 群组基本信息
            Column() {
              // 群组头像和名称
              Column() {
                SystemAvatar({
                  avatarUrl: this.groupInfo.avatar_url,
                  nickname: this.groupInfo.name,
                  avatarSize: 80
                })

                Text(this.groupInfo.name)
                  .fontSize(20)
                  .fontWeight(FontWeight.Bold)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .margin({ top: 16 })

                if (this.groupInfo.description) {
                  Text(this.groupInfo.description)
                    .fontSize(14)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .textAlign(TextAlign.Center)
                    .maxLines(3)
                    .textOverflow({ overflow: TextOverflow.Ellipsis })
                    .margin({ top: 8 })
                }
              }
              .width('100%')
              .padding(24)

              // 群组统计信息
              Row() {
                Column() {
                  Text(this.groupInfo.member_count.toString())
                    .fontSize(18)
                    .fontWeight(FontWeight.Bold)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))

                  Text('成员')
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .margin({ top: 4 })
                }
                .layoutWeight(1)

                Column() {
                  Text(this.groupInfo.distance ? `${Math.round(this.groupInfo.distance)}m` : '--')
                    .fontSize(18)
                    .fontWeight(FontWeight.Bold)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))

                  Text('距离')
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .margin({ top: 4 })
                }
                .layoutWeight(1)

                Column() {
                  Text(new Date(this.groupInfo.created_at).toLocaleDateString())
                    .fontSize(18)
                    .fontWeight(FontWeight.Bold)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))

                  Text('创建时间')
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .margin({ top: 4 })
                }
                .layoutWeight(1)
              }
              .width('100%')
              .padding({ left: 24, right: 24, bottom: 24 })
              .justifyContent(FlexAlign.SpaceAround)
            }
            .width('100%')
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius(12)
            .margin({ bottom: 16 })

            // 位置信息
            Column() {
              Row() {
                Image($r('sys.media.ohos_ic_public_location'))
                  .width(20)
                  .height(20)
                  .fillColor($r('sys.color.ohos_id_color_emphasize'))

                Text('位置信息')
                  .fontSize(16)
                  .fontWeight(FontWeight.Medium)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .margin({ left: 8 })
              }
              .width('100%')
              .margin({ bottom: 12 })

              Text(this.groupInfo.location_name)
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .width('100%')

              Text(`${this.groupInfo.latitude.toFixed(6)}, ${this.groupInfo.longitude.toFixed(6)}`)
                .fontSize(12)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .width('100%')
                .margin({ top: 4 })
            }
            .width('100%')
            .padding(16)
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius(12)
            .margin({ bottom: 16 })

            // 成员列表
            Column() {
              Row() {
                Text('群组成员')
                  .fontSize(16)
                  .fontWeight(FontWeight.Medium)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .layoutWeight(1)

                Button('查看全部')
                  .type(ButtonType.Normal)
                  .backgroundColor(Color.Transparent)
                  .fontColor($r('sys.color.ohos_id_color_emphasize'))
                  .fontSize(14)
                  .onClick(() => this.loadGroupMembers())
              }
              .width('100%')
              .margin({ bottom: 12 })

              Text(`${this.groupInfo.member_count} 名成员`)
                .fontSize(14)
                .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                .width('100%')
            }
            .width('100%')
            .padding(16)
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius(12)
            .margin({ bottom: 16 })

            // 操作按钮
            Column() {
              if (this.groupInfo.is_member) {
                // 已加入群组的操作
                Button('进入聊天')
                  .type(ButtonType.Capsule)
                  .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                  .fontColor(Color.White)
                  .width('100%')
                  .height(44)
                  .onClick(() => this.enterChat())
                  .margin({ bottom: 12 })

                Button('离开群组')
                  .type(ButtonType.Capsule)
                  .backgroundColor('#00000000')
                  .fontColor($r('sys.color.ohos_id_color_warning'))
                  .border({
                    width: 1,
                    color: $r('sys.color.ohos_id_color_warning')
                  })
                  .width('100%')
                  .height(44)
                  .enabled(!this.isLeaving)
                  .onClick(() => this.leaveGroup())
              } else {
                // 未加入群组的操作
                Button(this.groupInfo.is_password_required ? '申请加入（需要密码）' : '加入群组')
                  .type(ButtonType.Capsule)
                  .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                  .fontColor(Color.White)
                  .width('100%')
                  .height(44)
                  .enabled(!this.isJoining)
                  .onClick(() => this.joinGroup())
              }
            }
            .width('100%')
            .padding(16)
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .borderRadius(12)
            .margin({ bottom: 32 })
          }
          .width('100%')
          .padding(16)
        }
        .layoutWeight(1)
        .scrollBar(BarState.Auto)
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    .bindSheet($$this.showMemberList, this.buildMemberListSheet(), {
      height: '70%',
      showClose: true,
      dragBar: true
    })
  }

  /**
   * 构建成员列表弹窗
   */
  @Builder
  buildMemberListSheet() {
    Column() {
      // 标题
      Row() {
        Text('群组成员')
          .fontSize(18)
          .fontWeight(FontWeight.Bold)
          .layoutWeight(1)

        Text(`${this.members.length} 人`)
          .fontSize(14)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      }
      .width('100%')
      .padding(16)

      // 成员列表
      if (this.isLoadingMembers) {
        SystemLoadingView({ message: '加载成员列表中...' })
          .layoutWeight(1)
      } else if (this.members.length > 0) {
        List() {
          ForEach(this.members, (member: GetMembersResponse) => {
            ListItem() {
              this.buildMemberItem(member)
            }
          })
        }
        .width('100%')
        .layoutWeight(1)
        .padding({ left: 16, right: 16 })
        .scrollBar(BarState.Auto)
      } else {
        Column() {
          Text('暂无成员信息')
            .fontSize(16)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      }
    }
    .width('100%')
    .height('100%')
  }

  /**
   * 同步群组数据到分布式存储
   */
  private async syncGroupData(): Promise<void> {
    if (!this.groupInfo || !this.globalState.currentUser?.user_id) return;

    try {
      const syncDataObject: GroupSyncData = {
        groupId: this.groupId,
        groupInfo: this.groupInfo,
        members: this.members,
        lastViewed: Date.now()
      };
      
      const syncData: SyncData = {
        userId: this.globalState.currentUser.user_id,
        dataType: 'groups',
        data: syncDataObject,
        timestamp: Date.now(),
        deviceId: 'current_device' // 这里应该获取实际的设备ID
      };
      
      await this.distributedSyncService.syncUserData(syncData);

      hilog.info(DOMAIN_NUMBER, TAG, `群组数据已同步到分布式存储: ${this.groupInfo.name}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `同步群组数据失败: ${JSON.stringify(error)}`);
    }
  }


  /**
   * 发送社交通知
   */
  private async sendSocialNotification(type: 'join' | 'leave', targetUserId?: string): Promise<void> {
    if (!this.globalState.currentUser?.nickname || !this.groupInfo) return;

    try {
      let message: string;
      if (type === 'join') {
        message = `${this.globalState.currentUser.nickname} 加入了群组 ${this.groupInfo.name}`;
      } else {
        message = `${this.globalState.currentUser.nickname} 离开了群组 ${this.groupInfo.name}`;
      }

      await this.smartNotificationService.createSocialNotification(
        type === 'join' ? 'group_invite' : 'follow',
        this.globalState.currentUser.nickname,
        message,
        this.groupId
      );

      hilog.info(DOMAIN_NUMBER, TAG, `社交通知已发送: ${type}`);
    } catch (error) {
      hilog.error(DOMAIN_NUMBER, TAG, `发送社交通知失败: ${JSON.stringify(error)}`);
    }
  }
}