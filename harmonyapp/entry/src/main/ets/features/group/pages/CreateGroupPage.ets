/**
 * 创建群组页面
 * 重构：使用 HarmonyOS 原生 List 组件构建表单
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
import { LocationInfo } from '../../../core/location/LocationUtil';
import { ApiClient } from '../../../core/api/ApiClient';
import { LocationUtil } from '../../../core/location/LocationUtil';
import { PageNames } from '../../../constants/AppConstants';
import { SystemLoadingView } from '../../../components/SystemLoadingView';
import { SystemToast } from '../../../components/SystemToast';
import { IconButton, AppIcon, IconSize } from '../../../components/SystemIcon';
import { DesignConstants } from '../../../constants/DesignConstants';
import { PasswordSecurityUtil } from '../../../core/security/PasswordSecurityUtil';
import { CreateGroupRequest, GeoPoint } from '../../../core/types/api';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter, LoginRouteParam, GroupDetailRouteParam } from '../../../core/router/AppRouter';

@Component
export struct CreateGroupPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_LOCATION_PICK_RESULT) pickResult: LocationInfo | null = null;
  
  private stateManager = GlobalStateManager.getInstance();
  @State groupName: string = '';
  @State groupDescription: string = '';
  @State groupPassword: string = '';
  @State isPasswordRequired: boolean = false;
  @State maxMembers: number = 100;
  @State isCreating: boolean = false;
  @State localLocation: LocationInfo | null = null;

  async aboutToAppear() {
    // 优先消费“地图选点”结果（从 LocationPickerPage 返回时会写入）
    if (this.pickResult) {
      this.localLocation = {
        latitude: this.pickResult.latitude,
        longitude: this.pickResult.longitude,
        locationName: this.pickResult.locationName || '南天门'
      };
      AppStorage.setOrCreate<LocationInfo | null>(KEY_LOCATION_PICK_RESULT, null);
      return;
    }

    await this.getCurrentLocation();
  }

  private async getCurrentLocation(): Promise<void> {
    try {
      if (this.currentLocation) {
        this.localLocation = {
          latitude: this.currentLocation.latitude,
          longitude: this.currentLocation.longitude,
          locationName: this.currentLocation.locationName || '南天门'
        };
      } else {
        const location = await LocationUtil.getCurrentLocation();
        if (location) {
          this.localLocation = {
            latitude: location.latitude,
            longitude: location.longitude,
            locationName: location.locationName || '南天门'
          };
        }
      }
    } catch (error) {
      SystemToast.showError('获取位置信息失败', this.getUIContext());
    }
  }

  private async createGroup(): Promise<void> {
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再创建群组', this.getUIContext());
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '创建群组';
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }

    if (!this.groupName.trim()) {
      SystemToast.show('请输入群组名称', this.getUIContext());
      return;
    }
    if (!this.localLocation) {
      SystemToast.show('请先获取位置信息', this.getUIContext());
      return;
    }
    if (this.isPasswordRequired && !this.groupPassword.trim()) {
      SystemToast.show('请输入群组密码', this.getUIContext());
      return;
    }

    const localLoc = this.localLocation;
    this.isCreating = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const geoPoint: GeoPoint = {
          latitude: localLoc.latitude,
          longitude: localLoc.longitude
        };
        const request: CreateGroupRequest = {
          name: this.groupName.trim(),
          description: this.groupDescription.trim() || undefined,
          location: {
            point: geoPoint,
            location_name: localLoc.locationName || "南天门"
          },
          password: this.isPasswordRequired ? this.groupPassword.trim() : undefined,
          max_members: this.maxMembers,
          join_method: 'open'
        };

        const response = await ApiClient.createGroup(request);

        if (response.code === 0 && response.data) {
          SystemToast.showSuccess('群组创建成功', this.getUIContext());
          const detailParams = new GroupDetailRouteParam();
          detailParams.groupId = response.data.group.group_id;
          AppRouter.replacePath(PageNames.GROUP_DETAIL, detailParams);
        } else {
          SystemToast.showError(response.message || '创建群组失败', this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch (error) {
      SystemToast.showError('创建群组失败', this.getUIContext());
    } finally {
      this.isCreating = false;
    }
  }

  private async refreshLocation(): Promise<void> {
    try {
      const location = await LocationUtil.getCurrentLocation();
      if (location) {
        this.localLocation = {
          latitude: location.latitude,
          longitude: location.longitude,
          locationName: location.locationName || '南天门'
        };
        SystemToast.show('位置已更新', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('获取位置失败', this.getUIContext());
    }
  }

  private async openLocationPicker(): Promise<void> {
    try {
      AppRouter.push(PageNames.LOCATION_PICKER);
    } catch (error) {
      SystemToast.showError('跳转失败', this.getUIContext());
    }
  }

  build() {
    NavDestination() {
    Stack() {
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

          Text('创建群组')
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .layoutWeight(1)
            .textAlign(TextAlign.Center)
          
          Blank().width(48)
        }
        .width('100%')
        .height(56)
        .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        .backgroundColor($r('sys.color.ohos_id_color_background'))

        // 表单内容
        List({ space: 16 }) {
          // 基本信息组
          ListItemGroup({ header: this.ListHeader('基本信息') }) {
            ListItem() {
              Column() {
                Text('群组名称')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .width('100%')
                  .margin({ bottom: 4 })

                TextInput({ placeholder: '请输入群组名称', text: this.groupName })
                  .onChange((value: string) => {
                    this.groupName = value;
                  })
                  .maxLength(30)
                  .height(40)
                  .backgroundColor(Color.Transparent)
                  .padding({ left: 0 })
              }
              .alignItems(HorizontalAlign.Start)
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM
              })
            }

            ListItem() {
              Column() {
                Text('群组描述')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  .width('100%')
                  .margin({ bottom: 4 })

                TextArea({ placeholder: '请输入群组描述（可选）', text: this.groupDescription })
                  .onChange((value: string) => {
                    this.groupDescription = value;
                  })
                  .maxLength(200)
                  .height(80)
                  .backgroundColor(Color.Transparent)
                  .padding({ left: 0 })
              }
              .alignItems(HorizontalAlign.Start)
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM
              })
            }
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .divider({
            strokeWidth: 1, color: $r('sys.color.ohos_id_color_component_normal'),
            startMargin: DesignConstants.SPACING.LG, endMargin: DesignConstants.SPACING.LG
          })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

          // 位置信息组
          ListItemGroup({ header: this.ListHeader('位置信息') }) {
            ListItem() {
              Row() {
                Column() {
                  if (this.localLocation) {
                    Text(this.localLocation.locationName)
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    Text(`${this.localLocation.latitude.toFixed(6)}, ${this.localLocation.longitude.toFixed(6)}`)
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      .margin({ top: 2 })
                  } else {
                    Text('正在获取位置...')
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                  }
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)

                Row({ space: 8 }) {
                  Button('选点')
                    .type(ButtonType.Capsule)
                    .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .height(28)
                    .onClick(() => this.openLocationPicker())

                  Button('刷新')
                    .type(ButtonType.Capsule)
                    .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .height(28)
                    .onClick(() => this.refreshLocation())
                }
              }
              .width('100%')
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD
              })
            }
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

          // 隐私设置组
          ListItemGroup({ header: this.ListHeader('隐私与限制') }) {
            ListItem() {
              Row() {
                Text('最大成员数')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .layoutWeight(1)

                Counter() {
                  Text(this.maxMembers.toString())
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                }
                .onInc(() => {
                  if (this.maxMembers < 500) this.maxMembers += 10;
                })
                .onDec(() => {
                  if (this.maxMembers > 10) this.maxMembers -= 10;
                })
              }
              .width('100%')
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM
              })
            }

            ListItem() {
              Row() {
                Text('密码保护')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .layoutWeight(1)

                Toggle({ type: ToggleType.Switch, isOn: this.isPasswordRequired })
                  .selectedColor($r('sys.color.ohos_id_color_emphasize'))
                  .onChange((isOn: boolean) => {
                    this.isPasswordRequired = isOn;
                    if (!isOn) this.groupPassword = '';
                  })
              }
              .width('100%')
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM
              })
            }

            if (this.isPasswordRequired) {
              ListItem() {
                Column() {
                  Text('群组密码')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .width('100%')
                    .margin({ bottom: 4 })

                  TextInput({ placeholder: '请输入群组密码', text: this.groupPassword })
                    .type(InputType.Password)
                    .onChange((value: string) => {
                      this.groupPassword = value;
                    })
                    .maxLength(20)
                    .height(40)
                    .backgroundColor(Color.Transparent)
                    .padding({ left: 0 })
                }
                .alignItems(HorizontalAlign.Start)
                .padding({
                  left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                  top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM
                })
              }
            }
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
          .borderRadius(DesignConstants.RADIUS.LG)
          .divider({
            strokeWidth: 1, color: $r('sys.color.ohos_id_color_component_normal'),
            startMargin: DesignConstants.SPACING.LG, endMargin: DesignConstants.SPACING.LG
          })
          .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

          // 提交按钮
          ListItem() {
            Button('立即创建')
              .type(ButtonType.Capsule)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .fontColor(Color.White)
              .width('100%')
              .height(48)
              .fontSize(16)
              .enabled(!this.isCreating)
              .opacity(this.isCreating ? 0.7 : 1)
              .animation({ duration: 200, curve: Curve.EaseInOut })
              .onClick(() => this.createGroup())
          }
          .margin({ top: 24, bottom: 48, left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
        }
        .layoutWeight(1)
        .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
      }
      .width('100%')
      .height('100%')
      .backgroundColor($r('sys.color.ohos_id_color_sub_background'))

      if (this.isCreating) {
        SystemLoadingView()
      }
    }
    .width('100%')
    .height('100%')
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      // CreateGroupPage 无路由参数，参数逻辑在 aboutToAppear 中处理（pickResult / getCurrentLocation）
    })
  }

  @Builder
  ListHeader(title: string) {
    Text(title)
      .fontSize(DesignConstants.FONT.SIZE.MD)
      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      .margin({ left: DesignConstants.SPACING.XS, bottom: DesignConstants.SPACING.SM, top: DesignConstants.SPACING.LG })
      .width('100%')
  }
}

@Builder
export function CreateGroupPageBuilder() {
  CreateGroupPage()
}
