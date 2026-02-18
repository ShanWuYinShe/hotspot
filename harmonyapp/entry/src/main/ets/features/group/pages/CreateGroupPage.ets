/**
 * 创建群组页面
 * 重构：使用 HarmonyOS 原生 List 组件构建表单
 */

import { GlobalStateManager, KEY_CURRENT_LOCATION, KEY_BROWSE_ANCHOR, KEY_LOCATION_PICK_RESULT } from '../../../core/state/GlobalStateManager';
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
import { AppRouter, LoginRouteParam, GroupDetailRouteParam, LocationPickerRouteParam } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

@Component
export struct CreateGroupPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_LOCATION_PICK_RESULT) pickResult: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  
  private stateManager = GlobalStateManager.getInstance();
  @State groupName: string = '';
  @State groupDescription: string = '';
  @State groupPassword: string = '';
  @State isPasswordRequired: boolean = false;
  @State maxMembers: number = 100;
  @State isCreating: boolean = false;
  @State localLocation: LocationInfo | null = null;

  async aboutToAppear() {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
    // 优先消费“地图选点”结果（从 LocationPickerPage 返回时会写入）
    if (this.pickResult) {
      this.localLocation = {
        latitude: this.pickResult.latitude,
        longitude: this.pickResult.longitude,
        locationName: this.pickResult.locationName || '未知位置'
      };
      AppStorage.setOrCreate<LocationInfo | null>(KEY_LOCATION_PICK_RESULT, null);
      return;
    }

    this.initDefaultLocation();
  }

  /**
   * 初始化默认位置：优先浏览锚点 → 当前定位。
   * 浏览锚点是用户正在浏览的区域中心，更符合用户创建群组的意图。
   */
  private initDefaultLocation(): void {
    const refLoc: LocationInfo | null = this.browseAnchor ?? this.currentLocation;
    if (refLoc) {
      this.localLocation = {
        latitude: refLoc.latitude,
        longitude: refLoc.longitude,
        locationName: refLoc.locationName || '未知位置'
      };
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
      SystemToast.show('位置信息不可用，请选择位置', this.getUIContext());
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
            location_name: localLoc.locationName || "未知位置"
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
          locationName: location.locationName || '未知位置'
        };
        SystemToast.show('位置已更新', this.getUIContext());
      }
    } catch (error) {
      SystemToast.showError('获取位置失败', this.getUIContext());
    }
  }

  private openLocationPicker(): void {
    const param = new LocationPickerRouteParam();
    param.mode = 'location_pick';
    AppRouter.push(PageNames.LOCATION_PICKER, param);
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
        .padding({
          left: DesignConstants.SPACING.LG,
          right: DesignConstants.SPACING.LG,
          top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
          bottom: DesignConstants.SPACING.SM
        })
        .backgroundColor($r('app.color.bg_secondary'))
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

        // 表单内容
        Scroll() {
          Column({ space: DesignConstants.SPACING.LG }) {
            // 基本信息
            this.SectionHeader('基本信息')
            Column() {
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

              Divider()
                .strokeWidth(1)
                .color($r('sys.color.ohos_id_color_component_normal'))
                .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

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
            .backgroundColor($r('app.color.bg_secondary'))
            .borderRadius(DesignConstants.RADIUS.LG)

            // 位置信息
            this.SectionHeader('位置信息')
            Column() {
              Row() {
                Column() {
                  if (this.localLocation) {
                    Text(this.localLocation.locationName)
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor($r('sys.color.ohos_id_color_text_primary'))
                      .maxLines(2)
                      .textOverflow({ overflow: TextOverflow.Ellipsis })
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
                    .height(36)
                    .onClick(() => this.openLocationPicker())

                  Button('刷新')
                    .type(ButtonType.Capsule)
                    .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .height(36)
                    .onClick(() => this.refreshLocation())
                }
              }
              .width('100%')
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD
              })
            }
            .backgroundColor($r('app.color.bg_secondary'))
            .borderRadius(DesignConstants.RADIUS.LG)

            // 隐私与限制
            this.SectionHeader('隐私与限制')
            Column() {
              Row() {
                Text('最大成员数')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  .layoutWeight(1)

                Row({ space: 4 }) {
                  Button() {
                    Text() {
                      SymbolSpan($r(AppIcon.REMOVE))
                        .fontSize(16)
                        .fontColor([this.maxMembers > 10
                          ? $r('sys.color.ohos_id_color_text_primary')
                          : $r('sys.color.ohos_id_color_text_tertiary')])
                    }
                  }
                  .type(ButtonType.Circle)
                  .width(36)
                  .height(36)
                  .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                  .enabled(this.maxMembers > 10)
                  .onClick(() => {
                    if (this.maxMembers > 10) this.maxMembers = Math.max(10, this.maxMembers - 10);
                  })

                  TextInput({ text: this.maxMembers.toString() })
                    .type(InputType.Number)
                    .width(64)
                    .height(36)
                    .textAlign(TextAlign.Center)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                    .borderRadius(DesignConstants.RADIUS.MD)
                    .padding(0)
                    .onChange((value: string) => {
                      const num = parseInt(value);
                      if (!isNaN(num)) {
                        this.maxMembers = Math.max(10, Math.min(500, num));
                      }
                    })
                    .onBlur(() => {
                      if (this.maxMembers < 10) this.maxMembers = 10;
                      if (this.maxMembers > 500) this.maxMembers = 500;
                    })

                  Button() {
                    Text() {
                      SymbolSpan($r(AppIcon.ADD))
                        .fontSize(16)
                        .fontColor([this.maxMembers < 500
                          ? $r('sys.color.ohos_id_color_text_primary')
                          : $r('sys.color.ohos_id_color_text_tertiary')])
                    }
                  }
                  .type(ButtonType.Circle)
                  .width(36)
                  .height(36)
                  .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
                  .enabled(this.maxMembers < 500)
                  .onClick(() => {
                    if (this.maxMembers < 500) this.maxMembers = Math.min(500, this.maxMembers + 10);
                  })
                }
              }
              .width('100%')
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM
              })

              Divider()
                .strokeWidth(1)
                .color($r('sys.color.ohos_id_color_component_normal'))
                .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

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

              if (this.isPasswordRequired) {
                Divider()
                  .strokeWidth(1)
                  .color($r('sys.color.ohos_id_color_component_normal'))
                  .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

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
            .backgroundColor($r('app.color.bg_secondary'))
            .borderRadius(DesignConstants.RADIUS.LG)

            // 提交按钮
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
              .margin({ top: DesignConstants.SPACING.SM })
          }
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.SM,
            bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, 48, 0)
          })
        }
        .layoutWeight(1)
        .scrollBar(BarState.Off)
        .backgroundColor($r('app.color.bg_primary'))
      }
      .width('100%')
      .height('100%')
      .backgroundColor($r('app.color.bg_primary'))

      if (this.isCreating) {
        SystemLoadingView()
          .transition(TransitionEffect.OPACITY.animation({ duration: 300, curve: Curve.FastOutSlowIn }))
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
  SectionHeader(title: string) {
    Text(title)
      .fontSize(DesignConstants.FONT.SIZE.SM)
      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
      .margin({ bottom: -DesignConstants.SPACING.XS })
  }
}

@Builder
export function CreateGroupPageBuilder() {
  CreateGroupPage()
}
