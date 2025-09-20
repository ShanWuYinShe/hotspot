/**
 * 极简个人页面
 */

import { GlobalStateManager, GlobalState } from '../common/state/GlobalStateManager';
import { ApiClient } from '../common/api/ApiClient';
import { PreferencesUtil } from '../common/storage/PreferencesUtil';
import { HttpUtil } from '../common/network/HttpUtil';
import { WebSocketService } from '../common/websocket/WebSocketService';
import { PageNames } from '../constants/AppConstants';
import { SystemAvatar } from '../components/SystemAvatar';
import { SystemToast } from '../components/SystemToast';
import { SystemLoadingView } from '../components/SystemLoadingView';
import { UserResponse } from '../common/types/BackendApiTypes';

@Entry
@Component
struct ProfilePage {
  @State globalState: GlobalState = GlobalStateManager.getInstance().getState();
  private stateManager = GlobalStateManager.getInstance();
  @State isEditing: boolean = false;
  @State editNickname: string = '';
  @State editBio: string = '';
  @State isLoading: boolean = false;
  @State myMomentsCount: number = 0;
  @State myGroupsCount: number = 0;
  private webSocketService = WebSocketService.getInstance();

  async aboutToAppear() {
    // 订阅状态变化
    this.stateManager.subscribe('profile_page', (state: GlobalState) => {
      this.globalState = state;
    });

    // 初始化编辑表单
    if (this.globalState.currentUser) {
      this.editNickname = this.globalState.currentUser.nickname;
      this.editBio = this.globalState.currentUser.bio || '';
    }

    // 加载用户统计信息
    await this.loadUserStats();
  }

  aboutToDisappear() {
    // 取消订阅
    this.stateManager.unsubscribe('profile_page');
  }

  /**
   * 检查是否为临时用户
   * 根据后端文档，用户类型通过 user_id 的高8位编码
   */
  private isTemporaryUser(user: UserResponse | null): boolean {
    if (!user || !user.user_id) return false;
    
    try {
      // 将字符串 user_id 转换为数字，检查高8位
      const userId = BigInt(user.user_id);
      const userType = Number(userId >> 56n); // 获取高8位
      return userType === 1; // 假设1表示临时用户
    } catch (error) {
      return false;
    }
  }

  /**
   * 加载用户统计信息
   */
  private async loadUserStats(): Promise<void> {
    if (!this.globalState.isLoggedIn || !this.globalState.currentLocation) {
      return;
    }

    try {
      // 加载我的瞬间数量
      const momentsResponse = await ApiClient.searchMomentsByLocation({
        latitude: this.globalState.currentLocation.latitude,
        longitude: this.globalState.currentLocation.longitude,
        radius: 50000 // 大范围搜索
      });
      // 过滤出当前用户的瞬间
      if (momentsResponse.code === 0 && momentsResponse.data) {
        this.myMomentsCount = momentsResponse.data.items.filter(
          moment => moment.user_id === this.globalState.currentUser?.user_id
        ).length;
      }

      // 加载我的群组数量
      const groupsResponse = await ApiClient.searchGroupsByLocation({
        latitude: this.globalState.currentLocation.latitude,
        longitude: this.globalState.currentLocation.longitude,
        radius: 50000, // 大范围搜索
        limit: 100
      });
      // 过滤出我加入的群组
      if (groupsResponse.code === 0 && groupsResponse.data) {
        this.myGroupsCount = groupsResponse.data.items.filter(
          group => group.is_member
        ).length;
      }
    } catch (error) {
      console.error('加载用户统计失败:', error);
    }
  }

  /**
   * 开始编辑
   */
  private startEdit(): void {
    if (this.globalState.currentUser) {
      this.editNickname = this.globalState.currentUser.nickname;
      this.editBio = this.globalState.currentUser.bio || '';
      this.isEditing = true;
    }
  }

  /**
   * 取消编辑
   */
  private cancelEdit(): void {
    this.isEditing = false;
  }

  /**
   * 保存编辑
   */
  private async saveEdit(): Promise<void> {
    if (!this.editNickname.trim()) {
      SystemToast.showWarning('昵称不能为空');
      return;
    }

    this.isLoading = true;

    try {
      // 注意：这个API需要在后端实现
      // await ApiClient.updateUserProfile(this.editNickname, this.editBio);

      // 临时更新本地状态
      if (this.globalState.currentUser) {
        const updatedUser: UserResponse = {
          user_id: this.globalState.currentUser.user_id,
          nickname: this.editNickname,
          bio: this.editBio,
          avatar_url: this.globalState.currentUser.avatar_url,
          longitude: this.globalState.currentUser.longitude,
          latitude: this.globalState.currentUser.latitude,
          location_name: this.globalState.currentUser.location_name,
          created_at: this.globalState.currentUser.created_at,
          last_active_at: this.globalState.currentUser.last_active_at
        };
        this.stateManager.updateUser(updatedUser);
        await PreferencesUtil.setObject('user_info', updatedUser);
      }

      this.isEditing = false;
      SystemToast.showSuccess('资料更新成功');
    } catch (error) {
      SystemToast.showError('更新失败');
    } finally {
      this.isLoading = false;
    }
  }

  /**
   * 更新位置
   */
  private async updateLocation(): Promise<void> {
    if (!this.globalState.currentLocation) {
      SystemToast.show('请先获取位置信息');
      return;
    }

    try {
      await ApiClient.updateUserLocation({
        latitude: this.globalState.currentLocation.latitude,
        longitude: this.globalState.currentLocation.longitude,
        location_name: this.globalState.currentLocation.locationName || '南天门'
      });
      SystemToast.showSuccess('位置更新成功');
    } catch (error) {
      SystemToast.showError('位置更新失败');
    }
  }

  /**
   * 退出登录
   */
  private async logout(): Promise<void> {
    try {
      // 断开WebSocket连接
      this.webSocketService.disconnect();

      await PreferencesUtil.clear();
      await HttpUtil.clearToken();
      this.stateManager.updateUser(null);

      // 清空路由栈并跳转到主页
      const routerInstance = this.getUIContext().getRouter();
      routerInstance.clear();
      await routerInstance.pushUrl({ url: `pages/${PageNames.INDEX}` });
      SystemToast.show('已退出登录');
    } catch (error) {
      SystemToast.show('退出失败');
    }
  }

  build() {
    Column() {
      if (this.isLoading) {
        SystemLoadingView({ message: '正在更新...' })
      } else {
        // 标题栏
        Row() {
          Text('个人中心')
            .fontSize(20)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)

          if (this.globalState.currentUser && !this.isEditing) {
            Button('编辑')
              .type(ButtonType.Capsule)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .fontColor(Color.White)
              .fontSize(14)
              .padding({
                left: 16,
                right: 16,
                top: 8,
                bottom: 8
              })
              .onClick(() => this.startEdit())
          }
        }
        .width('100%')
        .padding(16)
        .backgroundColor($r('sys.color.ohos_id_color_background'))

        if (!this.globalState.isLoggedIn) {
          // 未登录状态
          Column({ space: 16 }) {
            Text('🔐')
              .fontSize(48)

            Text('请先登录\n查看个人信息')
              .fontSize(16)
              .fontColor($r('sys.color.ohos_id_color_text_secondary'))
              .textAlign(TextAlign.Center)

            Button('去登录')
              .type(ButtonType.Capsule)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .fontColor(Color.White)
              .onClick(async () => {
                try {
                  const routerInstance = this.getUIContext().getRouter();
                  await routerInstance.pushUrl({ url: `pages/${PageNames.LOGIN}` });
                } catch (error) {
                  SystemToast.showError('跳转失败');
                }
              })
          }
          .width('100%')
          .height('100%')
          .justifyContent(FlexAlign.Center)
        } else {
          Scroll() {
            Column({ space: 24 }) {
              // 用户信息区域
              if (this.isEditing) {
                this.buildEditForm()
              } else {
                this.buildUserInfo()
              }

              // 统计信息
              this.buildStatsSection()

              // 功能列表
              this.buildMenuSection()
            }
            .width('100%')
            .padding(16)
          }
          .layoutWeight(1)
        }
      }
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  @Builder
  private buildUserInfo(): void {
    Column({ space: 16 }) {
      SystemAvatar({
        avatarUrl: this.globalState.currentUser?.avatar_url || '',
        nickname: this.globalState.currentUser?.nickname || '',
        avatarSize: 80
      })

      Text(this.globalState.currentUser?.nickname || '')
        .fontSize(18)
        .fontWeight(FontWeight.Medium)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))

      if (this.globalState.currentUser?.bio) {
        Text(this.globalState.currentUser.bio)
          .fontSize(14)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .textAlign(TextAlign.Center)
          .maxLines(3)
      }

      if (this.isTemporaryUser(this.globalState.currentUser)) {
        Text('临时用户')
          .fontSize(12)
          .fontColor($r('sys.color.ohos_id_color_warning'))
          .backgroundColor($r('sys.color.ohos_id_color_background_transparent'))
          .padding({
            left: 8,
            right: 8,
            top: 4,
            bottom: 4
          })
          .borderRadius(12)
          .border({
            width: 1,
            color: $r('sys.color.ohos_id_color_warning')
          })
      }
    }
    .width('100%')
    .padding(24)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(12)
  }

  @Builder
  private buildEditForm(): void {
    Column({ space: 16 }) {
      SystemAvatar({
        avatarUrl: this.globalState.currentUser?.avatar_url || '',
        nickname: this.editNickname,
        avatarSize: 80
      })

      TextInput({ placeholder: '请输入昵称', text: this.editNickname })
        .fontSize(16)
        .padding(12)
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
        .borderRadius(8)
        .onChange((value: string) => {
          this.editNickname = value;
        })

      TextArea({ placeholder: '请输入个人简介', text: this.editBio })
        .fontSize(14)
        .padding(12)
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
        .borderRadius(8)
        .height(80)
        .onChange((value: string) => {
          this.editBio = value;
        })

      Row({ space: 12 }) {
        Button('取消')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_secondary'))
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .onClick(() => this.cancelEdit())

        Button('保存')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .layoutWeight(1)
          .onClick(() => this.saveEdit())
      }
      .width('100%')
    }
    .width('100%')
    .padding(24)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(12)
  }

  @Builder
  private buildStatsSection(): void {
    Column({ space: 12 }) {
      Text('我的数据')
        .fontSize(16)
        .fontWeight(FontWeight.Medium)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .alignSelf(ItemAlign.Start)

      Row() {
        Column({ space: 4 }) {
          Text(this.myMomentsCount.toString())
            .fontSize(20)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('sys.color.ohos_id_color_emphasize'))

          Text('我的瞬间')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)

        Column({ space: 4 }) {
          Text(this.myGroupsCount.toString())
            .fontSize(20)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('sys.color.ohos_id_color_emphasize'))

          Text('加入群组')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)

        Column({ space: 4 }) {
          Text(this.globalState.currentLocation ? '已获取' : '未获取')
            .fontSize(14)
            .fontWeight(FontWeight.Bold)
            .fontColor(this.globalState.currentLocation ?
            $r('sys.color.ohos_id_color_emphasize') :
            $r('sys.color.ohos_id_color_warning'))

          Text('位置信息')
            .fontSize(12)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
        .layoutWeight(1)
      }
      .width('100%')
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(12)
  }

  @Builder
  private buildMenuSection(): void {
    Column({ space: 8 }) {
      this.buildMenuItem('更新位置', '📍', () => this.updateLocation())
      this.buildMenuItem('我的瞬间', '✨', () => {
      })
      this.buildMenuItem('我的群组', '👥', () => {
      })
      this.buildMenuItem('设置', '⚙️', async () => {
        await this.getUIContext().getRouter().pushUrl({
          url: `pages/${PageNames.SETTINGS}`
        });
      })
      this.buildMenuItem('关于', 'ℹ️', () => {
      })
      this.buildMenuItem('退出登录', '🚪', (): void => {
        this.logout();
      })
    }
    .width('100%')
  }

  @Builder
  private buildMenuItem(title: string, icon: string, onTap: () => void): void {
    Row({ space: 12 }) {
      Text(icon)
        .fontSize(20)

      Text(title)
        .fontSize(16)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .layoutWeight(1)

      Text('>')
        .fontSize(16)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .width('100%')
    .padding(16)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(8)
    .margin({ bottom: 8 })
    .onClick(onTap)
  }
}