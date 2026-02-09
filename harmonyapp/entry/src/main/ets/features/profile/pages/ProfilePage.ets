/**
 * 极简个人页面
 * 重构：使用 HarmonyOS 原生 List 组件
 */

import { GlobalStateManager, KEY_CURRENT_USER, KEY_IS_LOGGED_IN, KEY_CURRENT_LOCATION } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { PreferencesUtil } from '../../../core/storage/PreferencesUtil';
import { HttpUtil } from '../../../core/network/HttpUtil';
import { WebSocketService } from '../../../core/network/WebSocketService';
import { PageNames } from '../../../constants/AppConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemToast } from '../../../components/SystemToast';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { MyProfileData, UserType, UpdateLocationRequest } from '../../../core/types/api';
import { isTempUser } from '../../../core/types/CommonTypes';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, SystemIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemCard } from '../../../components/SystemCard';
import { SystemPageHeader } from '../../../components/SystemPageHeader';

const DOMAIN_NUMBER: number = 0xFF03;
const TAG: string = 'ProfilePage';

@Component
export struct ProfilePage {
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfileData | null = null;
  @StorageLink(KEY_IS_LOGGED_IN) isLoggedIn: boolean = false;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  
  private stateManager = GlobalStateManager.getInstance();
  @State isEditing: boolean = false;
  @State editNickname: string = '';
  @State editBio: string = '';
  @State isLoading: boolean = false;
  @State myMomentsCount: number = 0;
  @State myGroupsCount: number = 0;
  private webSocketService = WebSocketService.getInstance();

  async aboutToAppear() {
    if (this.currentUser) {
      this.editNickname = this.currentUser.nickname;
      this.editBio = this.currentUser.bio || '';
    }
    await this.loadUserStats();
  }

  private isTemporaryUser(userType?: UserType): boolean {
    if (!userType) return false;
    return isTempUser(userType);
  }

  private async loadUserStats(): Promise<void> {
    if (!this.isLoggedIn || !this.currentUser) {
      return;
    }
    // Stats come from getMyProfile() response, which is already in currentUser
    if (this.currentUser.stats) {
      this.myMomentsCount = this.currentUser.stats.moments_count || 0;
      this.myGroupsCount = this.currentUser.stats.groups_count || 0;
    }
  }

  private startEdit(): void {
    if (this.currentUser) {
      this.editNickname = this.currentUser.nickname;
      this.editBio = this.currentUser.bio || '';
      this.isEditing = true;
    }
  }

  private cancelEdit(): void {
    this.isEditing = false;
  }

  private async saveEdit(): Promise<void> {
    if (!this.editNickname.trim()) {
      SystemToast.showWarning('昵称不能为空', this.getUIContext());
      return;
    }

    this.isLoading = true;
    try {
      if (this.currentUser) {
        // 调用后端 API 更新用户资料
        await ApiClient.updateProfile({
          nickname: this.editNickname,
          bio: this.editBio
        });

        // 刷新本地数据
        const response = await ApiClient.getMyProfile();
        if (response.code === 0 && response.data) {
          const updatedUser = response.data;
          this.stateManager.updateUser(updatedUser);
          await PreferencesUtil.setObject('user_info', updatedUser);
        }
      }

      this.isEditing = false;
      SystemToast.showSuccess('资料更新成功', this.getUIContext());
    } catch (error) {
      SystemToast.showError('更新失败', this.getUIContext());
    } finally {
      this.isLoading = false;
    }
  }

  private async updateLocation(): Promise<void> {
    if (!this.currentLocation) {
      SystemToast.show('请先获取位置信息', this.getUIContext());
      return;
    }

    try {
      const request: UpdateLocationRequest = {
        location: {
          latitude: this.currentLocation.latitude,
          longitude: this.currentLocation.longitude,
          location_name: this.currentLocation.locationName || '南天门'
        }
      };
      await ApiClient.updateLocation(request);
      SystemToast.showSuccess('位置更新成功', this.getUIContext());
    } catch (error) {
      SystemToast.showError('位置更新失败', this.getUIContext());
    }
  }

  private async logout(): Promise<void> {
    try {
      this.webSocketService.disconnect();
      await PreferencesUtil.clear();
      await HttpUtil.clearToken();
      this.stateManager.updateUser(null);

      const routerInstance = this.getUIContext().getRouter();
      routerInstance.clear();
      await routerInstance.pushUrl({ url: `features/main/pages/${PageNames.INDEX}` });
      SystemToast.show('已退出登录', this.getUIContext());
    } catch (error) {
      SystemToast.show('退出失败', this.getUIContext());
    }
  }

  build() {
    Column() {
      SystemPageHeader({
        title: '个人中心',
        action: (this.currentUser && !this.isEditing && this.isLoggedIn)
          ? { label: '编辑', onClick: () => this.startEdit() }
          : undefined
      })

      if (this.isLoading) {
        SystemLoadingView({ message: '正在更新...' }).layoutWeight(1)
      } else if (!this.isLoggedIn) {
        // 未登录状态
        Column({ space: 16 }) {
          Text('🔐')
            .fontSize(48)
          Text('请先登录\n查看个人信息')
            .fontSize(16)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .textAlign(TextAlign.Center)
          Button('去登录')
            .type(ButtonType.Capsule)
            .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
            .fontColor(Color.White)
            .onClick(async () => {
              try {
                const routerInstance = this.getUIContext().getRouter();
                await routerInstance.pushUrl({ url: `features/auth/pages/${PageNames.LOGIN}` });
              } catch (error) {
                SystemToast.showError('跳转失败', this.getUIContext());
              }
            })
        }
        .width('100%')
        .layoutWeight(1)
        .justifyContent(FlexAlign.Center)
      } else {
        // 已登录，使用 List 布局
        List({ space: 12 }) {
          // 1. 用户信息卡片
          ListItem() {
             if (this.isEditing) {
               this.EditFormContent()
             } else {
               this.UserInfoContent()
             }
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: 16, right: 16 })
          .padding(DesignConstants.SPACING.LG)

          // 2. 统计数据卡片
          ListItem() {
            this.StatsContent()
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: 16, right: 16 })
          .padding(DesignConstants.SPACING.LG)

          // 3. 菜单列表
          ListItemGroup() {
            ListItem() {
              this.MenuItemContent('更新位置', AppIcon.LOCATION, () => this.updateLocation())
            }
            .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
            
            ListItem() {
              this.MenuItemContent('我的瞬间', AppIcon.CAMERA, () => {})
            }
            .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

            ListItem() {
              this.MenuItemContent('我的群组', AppIcon.GROUP, () => {})
            }
            .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

            ListItem() {
              this.MenuItemContent('设置', AppIcon.SETTINGS, async () => {
                 await this.getUIContext().getRouter().pushUrl({
                   url: `features/settings/pages/${PageNames.SETTINGS}`
                 });
              })
            }
            .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

            ListItem() {
              this.MenuItemContent('关于', AppIcon.INFO, () => {})
            }
            .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

            ListItem() {
              this.MenuItemContent('退出登录', AppIcon.CLOSE, () => this.logout())
            }
            .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
          }
          .borderRadius(DesignConstants.RADIUS.LG)
          .divider({ strokeWidth: 1, color: DesignConstants.COLOR.TERTIARY_BG_LIGHT, startMargin: 56, endMargin: 16 }) // startMargin 避开图标
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
  }

  @Builder
  private UserInfoContent(): void {
    Column({ space: 16 }) {
      SystemAvatar({
        avatarUrl: this.currentUser?.avatar_url || '',
        name: this.currentUser?.nickname || '',
        avatarSize: 80,
        isOnline: this.currentUser?.is_online || false
      })

      Text(this.currentUser?.nickname || '')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Medium)
        .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)

      if (this.currentUser?.bio) {
        Text(this.currentUser.bio)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
          .textAlign(TextAlign.Center)
          .maxLines(3)
      }

      if (this.isTemporaryUser(this.currentUser?.user_type as UserType)) {
        Text('临时用户')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(DesignConstants.COLOR.WARNING)
          .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
          .padding({
            left: 8,
            right: 8,
            top: 4,
            bottom: 4
          })
          .borderRadius(12)
          .border({
            width: 1,
            color: DesignConstants.COLOR.WARNING
          })
      }
    }
    .width('100%')
  }

  @Builder
  private EditFormContent(): void {
    Column({ space: 16 }) {
      SystemAvatar({
        avatarUrl: this.currentUser?.avatar_url || '',
        name: this.editNickname,
        avatarSize: 80
      })

      TextInput({ placeholder: '请输入昵称', text: this.editNickname })
        .fontSize(16)
        .height(48)
        .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
        .borderRadius(8)
        .onChange((value: string) => {
          this.editNickname = value;
        })

      TextArea({ placeholder: '请输入个人简介', text: this.editBio })
        .fontSize(14)
        .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
        .borderRadius(8)
        .height(80)
        .onChange((value: string) => {
          this.editBio = value;
        })

      Row({ space: 12 }) {
        Button('取消')
          .type(ButtonType.Capsule)
          .backgroundColor(DesignConstants.COLOR.TERTIARY_BG_LIGHT)
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)
          .onClick(() => this.cancelEdit())

        Button('保存')
          .type(ButtonType.Capsule)
          .backgroundColor(DesignConstants.COLOR.BRAND_PRIMARY)
          .fontColor(Color.White)
          .layoutWeight(1)
          .onClick(() => this.saveEdit())
      }
      .width('100%')
    }
    .width('100%')
  }

  @Builder
  private StatsContent(): void {
    Column({ space: 12 }) {
      Text('我的数据')
        .fontSize(16)
        .fontWeight(FontWeight.Medium)
        .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
        .alignSelf(ItemAlign.Start)

      Row() {
        Column({ space: 4 }) {
          Text(this.myMomentsCount.toString())
            .fontSize(20)
            .fontWeight(FontWeight.Bold)
            .fontColor(DesignConstants.COLOR.BRAND_PRIMARY)

          Text('我的瞬间')
            .fontSize(12)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        }
        .layoutWeight(1)

        Column({ space: 4 }) {
          Text(this.myGroupsCount.toString())
            .fontSize(20)
            .fontWeight(FontWeight.Bold)
            .fontColor(DesignConstants.COLOR.BRAND_PRIMARY)

          Text('加入群组')
            .fontSize(12)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        }
        .layoutWeight(1)

        Column({ space: 4 }) {
          Text(this.currentLocation ? '已获取' : '未获取')
            .fontSize(14)
            .fontWeight(FontWeight.Bold)
            .fontColor(this.currentLocation ?
            DesignConstants.COLOR.BRAND_PRIMARY :
            DesignConstants.COLOR.WARNING)

          Text('位置信息')
            .fontSize(12)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
        }
        .layoutWeight(1)
      }
      .width('100%')
    }
    .width('100%')
  }

  @Builder
  private MenuItemContent(title: string, icon: string, onTap: () => void): void {
    Row({ space: 12 }) {
      SystemIcon({ icon: icon as AppIcon, iconSize: 20, iconColor: DesignConstants.COLOR.BRAND_PRIMARY })

      Text(title)
        .fontSize(16)
        .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
        .layoutWeight(1)

      Text('>')
        .fontSize(16)
        .fontColor(DesignConstants.COLOR.TEXT_TERTIARY_LIGHT)
    }
    .width('100%')
    .padding({ top: 16, bottom: 16 })
    .onClick(onTap)
  }
}
