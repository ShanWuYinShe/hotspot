/**
 * 群组详情页面
 * 显示群组信息、成员列表、群组管理功能
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { PageNames } from '../constants/AppConstants';
import { 
  SearchGroupByIdRequest, 
  SearchGroupByIdResponse, 
  GetMembersResponse 
} from '../common/types/BackendApiTypes';
import { SystemAvatar } from '../components/SystemAvatar';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { SystemToast } from '../components/SystemToast';
import { IconButton, AppIcon, IconSize } from '../components/SystemIcon';

@Entry
@Component
struct GroupDetailPage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State group: SearchGroupByIdResponse | null = null;
  @State members: GetMembersResponse[] = [];
  @State isLoading: boolean = true;
  @State isJoining: boolean = false;
  @State groupPassword: string = '';
  @State showPasswordDialog: boolean = false;
  private groupId: string = '';

  async aboutToAppear() {
    // 订阅状态变化
    this.stateManager.subscribe('group_detail_page', (state: GlobalState) => {
      this.globalState = state;
    });

    // 获取路由参数
    try {
      const routerInstance = this.getUIContext().getRouter();
      const params = routerInstance.getParams() as Record<string, Object>;
      if (params && typeof params.groupId === 'string') {
        this.groupId = params.groupId;
        await this.loadGroupDetail();
      }
    } catch (error) {
      SystemToast.showError('获取群组信息失败');
    }
  }

  aboutToDisappear() {
    this.stateManager.unsubscribe('group_detail_page');
  }

  /**
   * 加载群组详情
   */
  private async loadGroupDetail(): Promise<void> {
    if (!this.groupId) return;

    this.isLoading = true;
    try {
      // 获取群组详情
      const request: SearchGroupByIdRequest = {
        group_id: this.groupId
      };
      const groupResponse = await ApiClient.getGroupById(request);
      if (groupResponse.code === 0 && groupResponse.data) {
        const groupData = groupResponse.data;
        this.group = {
          group_id: groupData.group_id,
          name: groupData.name,
          description: groupData.description,
          owner_id: groupData.owner_id,
          creator_name: groupData.creator_name,
          created_at: groupData.created_at,
          last_active_at: groupData.last_active_at,
          latitude: groupData.latitude,
          longitude: groupData.longitude,
          member_count: groupData.member_count,
          distance: groupData.distance,
          location_name: groupData.location_name,
          is_password_required: groupData.is_password_required,
          is_member: groupData.is_member,
          user_role: groupData.user_role || 'member',
          avatar_url: groupData.avatar_url
        };
      }

      // 获取群组成员
      const membersResponse = await ApiClient.getGroupMembers({
        group_id: this.groupId,
        limit: 100
      });
      if (membersResponse.code === 0 && membersResponse.data) {
        this.members = membersResponse.data.items;
      }
    } catch (error) {
      SystemToast.showError('加载群组信息失败');
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 加入群组
   */
  private async joinGroup(): Promise<void> {
    if (!this.group || !this.globalState.currentUser) {
      SystemToast.show('请先登录');
      return;
    }

    // 如果需要密码且未输入，显示密码对话框
    if (this.group.is_password_required && !this.groupPassword) {
      this.showPasswordDialog = true;
      return;
    }

    this.isJoining = true;
    try {
      const response = await ApiClient.joinGroup({
        group_id: this.group.group_id,
        password: this.groupPassword || undefined
      });

      if (response.code === 0) {
        SystemToast.showSuccess('加入群组成功');
        await this.loadGroupDetail(); // 重新加载群组信息
        this.showPasswordDialog = false;
        this.groupPassword = '';
      } else {
        SystemToast.showError(response.msg || '加入群组失败');
      }
    } catch (error) {
      SystemToast.showError('加入群组失败');
    } finally {
      this.isJoining = false;
    }
  }

  /**
   * 离开群组
   */
  private async leaveGroup(): Promise<void> {
    if (!this.group || !this.globalState.currentUser) return;

    try {
      const response = await ApiClient.leaveGroup({ group_id: this.group.group_id });
      if (response.code === 0) {
        SystemToast.showSuccess('已离开群组');
        // 返回上一页
        const routerInstance = this.getUIContext().getRouter();
        routerInstance.back();
      } else {
        SystemToast.showError(response.msg || '离开群组失败');
      }
    } catch (error) {
      SystemToast.showError('离开群组失败');
    }
  }

  /**
   * 进入群聊
   */
  private async enterGroupChat(): Promise<void> {
    if (!this.group) return;

    try {
      const routerInstance = this.getUIContext().getRouter();
      await routerInstance.pushUrl({
        url: 'pages/ChatPage',
        params: {
          groupId: this.group.group_id,
          groupName: this.group.name,
          groupType: 'public_chat'
        }
      });
    } catch (error) {
      SystemToast.showError('进入群聊失败');
    }
  }

  /**
   * 构建密码输入对话框
   */
  @Builder
  buildPasswordDialog() {
    if (this.showPasswordDialog) {
      Column() {
        Text('输入群组密码')
          .fontSize(18)
          .fontWeight(FontWeight.Medium)
          .margin({ bottom: 20 })

        TextInput({ placeholder: '请输入群组密码' })
          .type(InputType.Password)
          .onChange((value: string) => {
            this.groupPassword = value;
          })
          .margin({ bottom: 20 })

        Row() {
          Button('取消')
            .type(ButtonType.Normal)
            .backgroundColor($r('sys.color.ohos_id_color_background'))
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .onClick(() => {
              this.showPasswordDialog = false;
              this.groupPassword = '';
            })
            .layoutWeight(1)

          Button('确认')
            .type(ButtonType.Capsule)
            .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
            .fontColor(Color.White)
            .enabled(!this.isJoining)
            .onClick((): Promise<void> => this.joinGroup())
            .layoutWeight(1)
            .margin({ left: 12 })
        }
        .width('100%')
      }
      .width('80%')
      .padding(24)
      .backgroundColor(Color.White)
      .borderRadius(12)
      .shadow({ radius: 20, color: 'rgba(0,0,0,0.1)' })
    }
  }

  /**
   * 构建成员列表项
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
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })

        Text(`角色: ${member.role}`)
          .fontSize(12)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .margin({ top: 2 })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
      .margin({ left: 12 })

      // 显示角色标识
      if (this.group && member.user_id === this.group.owner_id) {
        Text('群主')
          .fontSize(10)
          .fontColor(Color.White)
          .backgroundColor($r('sys.color.ohos_id_color_warning'))
          .padding({ left: 6, right: 6, top: 2, bottom: 2 })
          .borderRadius(4)
      }
    }
    .width('100%')
    .padding(12)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(8)
    .margin({ bottom: 8 })
  }

  build() {
    Stack() {
      Column() {
        // 标题栏
        Row() {
          IconButton({
            icon: AppIcon.BACK,
            size: IconSize.LARGE,
            backgroundColor: Color.Transparent,
            onClick: () => {
              const routerInstance = this.getUIContext().getRouter();
              routerInstance.back();
            }
          })

          Text('群组详情')
            .fontSize(18)
            .fontWeight(FontWeight.Medium)
            .layoutWeight(1)
            .textAlign(TextAlign.Center)

          // 占位符保持居中
          Row().width(40).height(40)
        }
        .width('100%')
        .height(56)
        .padding({ left: 16, right: 16 })
        .backgroundColor($r('sys.color.ohos_id_color_background'))

        if (this.isLoading) {
          SystemLoadingView({ message: '加载中...' })
            .layoutWeight(1)
        } else if (this.group) {
          Scroll() {
            Column() {
              // 群组信息卡片
              Column() {
                SystemAvatar({
                  avatarUrl: this.group.avatar_url,
                  nickname: this.group.name,
                  avatarSize: 80
                })
                  .margin({ bottom: 16 })

                Text(this.group.name)
                  .fontSize(20)
                  .fontWeight(FontWeight.Medium)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .margin({ bottom: 8 })

                if (this.group.description) {
                  Text(this.group.description)
                    .fontSize(14)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .textAlign(TextAlign.Center)
                    .margin({ bottom: 16 })
                }

                Row() {
                  Text(`成员: ${this.members.length}`)
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))

                  Text(`位置: ${this.group.location_name}`)
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .margin({ left: 16 })
                }
              }
              .width('100%')
              .padding(24)
              .backgroundColor($r('sys.color.ohos_id_color_background'))
              .borderRadius(12)
              .margin({ bottom: 16 })

              // 操作按钮
              if (this.group.is_member) {
                Row() {
                  Button('进入群聊')
                    .type(ButtonType.Capsule)
                    .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontColor(Color.White)
                    .onClick(() => this.enterGroupChat())
                    .layoutWeight(1)

                  Button('离开群组')
                    .type(ButtonType.Normal)
                    .backgroundColor($r('sys.color.ohos_id_color_background'))
                    .fontColor($r('sys.color.ohos_id_color_warning'))
                    .onClick(() => this.leaveGroup())
                    .layoutWeight(1)
                    .margin({ left: 12 })
                }
                .width('100%')
                .margin({ bottom: 16 })
              } else {
                Button('加入群组')
                  .type(ButtonType.Capsule)
                  .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
                  .fontColor(Color.White)
                  .width('100%')
                  .enabled(!this.isJoining)
                  .onClick(() => this.joinGroup())
                  .margin({ bottom: 16 })
              }

              // 成员列表
              if (this.members.length > 0) {
                Column() {
                  Text(`群成员 (${this.members.length})`)
                    .fontSize(16)
                    .fontWeight(FontWeight.Medium)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    .width('100%')
                    .margin({ bottom: 12 })

                  ForEach(this.members, (member: GetMembersResponse) => {
                    this.buildMemberItem(member)
                  })
                }
                .width('100%')
                .alignItems(HorizontalAlign.Start)
              }
            }
            .width('100%')
            .padding(16)
          }
          .layoutWeight(1)
          .scrollBar(BarState.Auto)
        } else {
          Column() {
            Text('群组不存在')
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

      // 密码输入对话框
      if (this.showPasswordDialog) {
        Column() {
          this.buildPasswordDialog()
        }
        .width('100%')
        .height('100%')
        .backgroundColor('rgba(0,0,0,0.5)')
        .justifyContent(FlexAlign.Center)
        .alignItems(HorizontalAlign.Center)
        .onClick(() => {
          this.showPasswordDialog = false;
          this.groupPassword = '';
        })
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }
}
