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
import { SystemAvatar } from '../../../components/SystemAvatar';
import { DesignConstants } from '../../../constants/DesignConstants';
import { CreateGroupRequest, GeoPoint } from '../../../core/types/api';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter, LoginRouteParam, GroupDetailRouteParam, LocationPickerRouteParam } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { common } from '@kit.AbilityKit';

const GROUP_NAME_MAX: number = 50;
const GROUP_DESCRIPTION_MAX: number = 500;
const GROUP_PASSWORD_MIN: number = 4;
const GROUP_PASSWORD_MAX: number = 32;
const GROUP_MEMBERS_MIN: number = 2;
const GROUP_MEMBERS_MAX: number = 2000;
const LOCATION_NAME_MAX: number = 100;

@Preview
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
  @State groupAvatarPreviewUrl: string = '';
  @State groupAvatarLocalUri: string = '';
  @State groupAvatarFileId: string = '';
  @State isGroupAvatarUploading: boolean = false;

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

    const trimmedName = this.groupName.trim();
    const trimmedDescription = this.groupDescription.trim();
    const trimmedPassword = this.groupPassword.trim();

    if (!trimmedName) {
      SystemToast.show('请输入群组名称', this.getUIContext());
      return;
    }
    if (trimmedName.length > GROUP_NAME_MAX) {
      SystemToast.show(`群组名称最多 ${GROUP_NAME_MAX} 字符`, this.getUIContext());
      return;
    }
    if (trimmedDescription.length > GROUP_DESCRIPTION_MAX) {
      SystemToast.show(`群组描述最多 ${GROUP_DESCRIPTION_MAX} 字符`, this.getUIContext());
      return;
    }
    if (!this.localLocation) {
      SystemToast.show('位置信息不可用，请选择位置', this.getUIContext());
      return;
    }
    if (this.maxMembers < GROUP_MEMBERS_MIN || this.maxMembers > GROUP_MEMBERS_MAX) {
      SystemToast.show(`最大成员数需在 ${GROUP_MEMBERS_MIN}-${GROUP_MEMBERS_MAX} 之间`, this.getUIContext());
      return;
    }
    if (this.isPasswordRequired && !trimmedPassword) {
      SystemToast.show('请输入群组密码', this.getUIContext());
      return;
    }
    if (trimmedPassword && (trimmedPassword.length < GROUP_PASSWORD_MIN || trimmedPassword.length > GROUP_PASSWORD_MAX)) {
      SystemToast.show(`群组密码长度需在 ${GROUP_PASSWORD_MIN}-${GROUP_PASSWORD_MAX} 字符之间`, this.getUIContext());
      return;
    }

    const localLoc = this.localLocation;
    const locationName = (localLoc.locationName || '未知位置').trim();
    if (!locationName || locationName.length > LOCATION_NAME_MAX) {
      SystemToast.show(`位置名称需在 1-${LOCATION_NAME_MAX} 字符之间`, this.getUIContext());
      return;
    }
    this.isCreating = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
        const avatarFileId = await this.ensureGroupAvatarUploaded(context);
        const geoPoint: GeoPoint = {
          latitude: localLoc.latitude,
          longitude: localLoc.longitude
        };
        const request: CreateGroupRequest = {
          name: trimmedName,
          description: trimmedDescription || undefined,
          avatar_file_id: avatarFileId,
          location: {
            point: geoPoint,
            location_name: locationName
          },
          password: this.isPasswordRequired ? trimmedPassword : undefined,
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

  private async selectGroupAvatar(): Promise<void> {
    if (this.isCreating || this.isGroupAvatarUploading) {
      return;
    }
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selectedUris = await MediaServiceManager.selectImages(context, 1);
      if (selectedUris.length === 0) {
        return;
      }
      this.groupAvatarLocalUri = selectedUris[0];
      this.groupAvatarPreviewUrl = selectedUris[0];
      this.groupAvatarFileId = '';
      SystemToast.show('群头像已选择，创建时自动上传', this.getUIContext());
    } catch (error) {
      SystemToast.showError('选择群头像失败，请重试', this.getUIContext());
    }
  }

  private clearGroupAvatarSelection(): void {
    this.groupAvatarPreviewUrl = '';
    this.groupAvatarLocalUri = '';
    this.groupAvatarFileId = '';
  }

  private async ensureGroupAvatarUploaded(context: common.UIAbilityContext): Promise<string | undefined> {
    if (!this.groupAvatarLocalUri) {
      return undefined;
    }
    if (this.groupAvatarFileId) {
      return this.groupAvatarFileId;
    }
    this.isGroupAvatarUploading = true;
    try {
      const compressedPath = await MediaServiceManager.compressImage(context, this.groupAvatarLocalUri, 80, 1024 * 1024 * 2);
      const uploadUri = await MediaServiceManager.prepareFileForUpload(context, compressedPath);
      const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'group_avatar');
      if (!uploadResult.file_id) {
        throw new Error('群头像上传失败');
      }
      this.groupAvatarFileId = uploadResult.file_id;
      return this.groupAvatarFileId;
    } finally {
      this.isGroupAvatarUploading = false;
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
            bgColor: $r('app.color.bg_tertiary'),
            onButtonClick: () => {
              AppRouter.pop();
            }
          })

          Text('创建群组')
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontWeight(FontWeight.Medium)
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
        .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
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
                  .maxLength(GROUP_NAME_MAX)
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
                .color($r('sys.color.ohos_id_color_list_separator'))
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
                  .maxLength(GROUP_DESCRIPTION_MAX)
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
            .borderRadius(DesignConstants.RADIUS.XXL)
            .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
            .shadow(DesignConstants.SHADOW.SMALL)

            // 群组头像
            this.SectionHeader('群组头像')
            Column() {
              Row({ space: DesignConstants.SPACING.MD }) {
                SystemAvatar({
                  avatarUrl: this.groupAvatarPreviewUrl,
                  name: this.groupName || '群组',
                  avatarSize: 56,
                  shape: 'squircle'
                })

                Column({ space: 4 }) {
                  Text('建议使用清晰、易识别的头像')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                  Text(this.isGroupAvatarUploading ? '群头像上传中...' : '未设置时将使用默认群头像')
                    .fontSize(12)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)

                Row({ space: 8 }) {
                  Button(this.groupAvatarPreviewUrl.length > 0 ? '重选' : '选择')
                    .type(ButtonType.Normal)
                    .backgroundColor($r('app.color.bg_tertiary'))
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .borderRadius(DesignConstants.RADIUS.XXL)
                    .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
                    .height(34)
                    .enabled(!this.isCreating && !this.isGroupAvatarUploading)
                    .onClick(() => this.selectGroupAvatar())

                  if (this.groupAvatarPreviewUrl.length > 0) {
                    Button('移除')
                      .type(ButtonType.Normal)
                      .backgroundColor($r('app.color.bg_tertiary'))
                      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .borderRadius(DesignConstants.RADIUS.XXL)
                      .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
                      .height(34)
                      .enabled(!this.isCreating && !this.isGroupAvatarUploading)
                      .onClick(() => this.clearGroupAvatarSelection())
                  }
                }
              }
              .width('100%')
              .padding({
                left: DesignConstants.SPACING.LG,
                right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.MD,
                bottom: DesignConstants.SPACING.MD
              })
            }
            .backgroundColor($r('app.color.bg_secondary'))
            .borderRadius(DesignConstants.RADIUS.XXL)
            .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
            .shadow(DesignConstants.SHADOW.SMALL)

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
                    .type(ButtonType.Normal)
                    .backgroundColor($r('app.color.bg_tertiary'))
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .borderRadius(DesignConstants.RADIUS.XXL)
                    .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
                    .height(36)
                    .onClick(() => this.openLocationPicker())

                  Button('刷新')
                    .type(ButtonType.Normal)
                    .backgroundColor($r('app.color.bg_tertiary'))
                    .fontColor($r('sys.color.ohos_id_color_emphasize'))
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .borderRadius(DesignConstants.RADIUS.XXL)
                    .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
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
            .borderRadius(DesignConstants.RADIUS.XXL)
            .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
            .shadow(DesignConstants.SHADOW.SMALL)

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
                        .fontColor([this.maxMembers > GROUP_MEMBERS_MIN
                          ? $r('sys.color.ohos_id_color_text_primary')
                          : $r('sys.color.ohos_id_color_text_tertiary')])
                    }
                  }
                  .type(ButtonType.Normal)
                  .width(36)
                  .height(36)
                  .backgroundColor($r('app.color.bg_tertiary'))
                  .borderRadius(DesignConstants.RADIUS.XXL)
                  .enabled(this.maxMembers > GROUP_MEMBERS_MIN)
                  .onClick(() => {
                    if (this.maxMembers > GROUP_MEMBERS_MIN) this.maxMembers = Math.max(GROUP_MEMBERS_MIN, this.maxMembers - 10);
                  })

                  TextInput({ text: this.maxMembers.toString() })
                    .type(InputType.Number)
                    .width(64)
                    .height(36)
                    .textAlign(TextAlign.Center)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor($r('sys.color.ohos_id_color_text_primary'))
                    .backgroundColor($r('app.color.bg_tertiary'))
                    .borderRadius(DesignConstants.RADIUS.MD)
                    .padding(0)
                    .onChange((value: string) => {
                      const num = parseInt(value);
                      if (!isNaN(num)) {
                        this.maxMembers = Math.max(GROUP_MEMBERS_MIN, Math.min(GROUP_MEMBERS_MAX, num));
                      }
                    })
                    .onBlur(() => {
                      if (this.maxMembers < GROUP_MEMBERS_MIN) this.maxMembers = GROUP_MEMBERS_MIN;
                      if (this.maxMembers > GROUP_MEMBERS_MAX) this.maxMembers = GROUP_MEMBERS_MAX;
                    })

                  Button() {
                    Text() {
                      SymbolSpan($r(AppIcon.ADD))
                        .fontSize(16)
                        .fontColor([this.maxMembers < GROUP_MEMBERS_MAX
                          ? $r('sys.color.ohos_id_color_text_primary')
                          : $r('sys.color.ohos_id_color_text_tertiary')])
                    }
                  }
                  .type(ButtonType.Normal)
                  .width(36)
                  .height(36)
                  .backgroundColor($r('app.color.bg_tertiary'))
                  .borderRadius(DesignConstants.RADIUS.XXL)
                  .enabled(this.maxMembers < GROUP_MEMBERS_MAX)
                  .onClick(() => {
                    if (this.maxMembers < GROUP_MEMBERS_MAX) this.maxMembers = Math.min(GROUP_MEMBERS_MAX, this.maxMembers + 10);
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
                .color($r('sys.color.ohos_id_color_list_separator'))
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
                  .color($r('sys.color.ohos_id_color_list_separator'))
                  .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

                Column() {
                  Text('群组密码')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor($r('sys.color.ohos_id_color_text_secondary'))
                    .width('100%')
                    .margin({ bottom: 4 })

                  TextInput({ placeholder: `请输入群组密码（${GROUP_PASSWORD_MIN}-${GROUP_PASSWORD_MAX}位）`, text: this.groupPassword })
                    .type(InputType.Password)
                    .onChange((value: string) => {
                      this.groupPassword = value;
                    })
                    .maxLength(GROUP_PASSWORD_MAX)
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
            .borderRadius(DesignConstants.RADIUS.XXL)
            .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
            .shadow(DesignConstants.SHADOW.SMALL)

            // 提交按钮
            Button('立即创建')
              .type(ButtonType.Normal)
              .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
              .fontColor(Color.White)
              .borderRadius(DesignConstants.RADIUS.XXL)
              .shadow(DesignConstants.SHADOW.SMALL)
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
