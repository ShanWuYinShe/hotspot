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
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemButton } from '../../../components/SystemButton';
import { SystemInput } from '../../../components/SystemInput';
import { SystemTextArea } from '../../../components/SystemTextArea';
import { IconButton, AppIcon, IconSize } from '../../../components/SystemIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SoftDivider } from '../../../components/SoftDivider';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppThemeVisuals, InputBorderResolution } from '../../../core/theme/AppThemeVisuals';
import { validateField, FieldRule } from '../../../core/util/FormValidator';
import { CreateGroupRequest, GeoPoint } from '../../../core/types/api';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { AppRouter, LoginRouteParam, GroupDetailRouteParam, LocationPickerRouteParam } from '../../../core/router/AppRouter';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { common } from '@kit.AbilityKit';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';

const GROUP_NAME_MAX: number = 50;
const GROUP_DESCRIPTION_MAX: number = 500;
const GROUP_PASSWORD_MIN: number = 4;
const GROUP_PASSWORD_MAX: number = 32;
const GROUP_MEMBERS_MIN: number = 2;
const GROUP_MEMBERS_MAX: number = 2000;

const GROUP_NAME_RULES: FieldRule[] = [
  { required: true, requiredMessage: '请输入群组名称' },
  { maxLength: GROUP_NAME_MAX, maxLengthMessage: `群组名称最多 ${GROUP_NAME_MAX} 字符` }
];
const GROUP_DESC_RULES: FieldRule[] = [
  { maxLength: GROUP_DESCRIPTION_MAX, maxLengthMessage: `群组描述最多 ${GROUP_DESCRIPTION_MAX} 字符` }
];
const LOCATION_NAME_MAX: number = 100;

@Preview
@Component
export struct CreateGroupPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_BROWSE_ANCHOR) browseAnchor: LocationInfo | null = null;
  @StorageLink(KEY_CURRENT_LOCATION) currentLocation: LocationInfo | null = null;
  @StorageLink(KEY_LOCATION_PICK_RESULT) pickResult: LocationInfo | null = null;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  
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
  @State groupNameFocused: boolean = false;
  @State groupDescriptionFocused: boolean = false;
  @State groupPasswordFocused: boolean = false;
  @State maxMembersFocused: boolean = false;
  @State groupNameTouched: boolean = false;
  @State groupDescriptionTouched: boolean = false;
  @State groupPasswordTouched: boolean = false;

  private isDestroyed: boolean = false;

  async aboutToAppear() {
    SystemUiController.syncWithCurrentTheme();
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

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private fieldBorder(focused: boolean, hasError: boolean): InputBorderResolution {
    return AppThemeVisuals.resolveInputBorder({
      backgroundColor: this.getSurfaceSubtle(),
      borderColor: this.getSurfaceBorder(),
      borderWidth: DesignConstants.BORDER_WIDTH.HAIRLINE,
      borderRadius: DesignConstants.SHAPE.INPUT,
      shadow: { radius: 0, color: DesignConstants.COLOR.TRANSPARENT, offsetX: 0, offsetY: 0 },
      height: DesignConstants.COMPONENT.INPUT_HEIGHT,
      horizontalPadding: DesignConstants.SPACING.MD
    }, focused, hasError, this.resolvedThemeId);
  }

  private getGroupNameError(): string | undefined {
    return this.groupNameTouched ? validateField(this.groupName, GROUP_NAME_RULES) : undefined;
  }

  private getGroupDescriptionError(): string | undefined {
    return this.groupDescriptionTouched ? validateField(this.groupDescription, GROUP_DESC_RULES) : undefined;
  }

  private getGroupPasswordError(): string | undefined {
    if (!this.isPasswordRequired || !this.groupPasswordTouched) {
      return undefined;
    }
    return validateField(this.groupPassword, [
      { required: true, requiredMessage: '请输入群组密码' },
      { minLength: GROUP_PASSWORD_MIN, minLengthMessage: `群组密码长度需在 ${GROUP_PASSWORD_MIN}-${GROUP_PASSWORD_MAX} 字符之间` },
      { maxLength: GROUP_PASSWORD_MAX, maxLengthMessage: `群组密码长度需在 ${GROUP_PASSWORD_MIN}-${GROUP_PASSWORD_MAX} 字符之间` }
    ]);
  }

  aboutToDisappear(): void {
    this.isDestroyed = true;
  }

  private async createGroup(): Promise<void> {
    if (this.isDestroyed) return;
    const currentUser = this.stateManager.getCurrentUser();
    if (!currentUser) {
      SystemToast.showWarning('请先登录后再创建群组', this.getUIContext());
      const loginParams = new LoginRouteParam();
      loginParams.reason = 'action_required';
      loginParams.action = '创建群组';
      loginParams.redirectUrl = PageNames.CREATE_GROUP;
      AppRouter.push(PageNames.LOGIN, loginParams);
      return;
    }

    this.groupNameTouched = true;
    this.groupDescriptionTouched = true;
    if (this.isPasswordRequired) {
      this.groupPasswordTouched = true;
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
      if (this.isDestroyed) return;
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
          // 有密码 → password 准入；否则 direct 直接加入（requires_approval 默认 false）
          join_method: this.isPasswordRequired ? 'password' : 'direct'
        };

        const response = await ApiClient.createGroup(request);

        if (response.code === 0 && response.data) {
          SystemToast.showSuccess('群组创建成功', this.getUIContext());
          const detailParams = new GroupDetailRouteParam();
          detailParams.groupId = response.data.group.group_id;
          AppRouter.replacePath(PageNames.GROUP_DETAIL, detailParams);
        } else {
          SystemToast.showError(getErrorMessage(response, '创建群组失败'), this.getUIContext());
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
    } catch (error) {
      SystemToast.showError('创建群组失败', this.getUIContext());
    } finally {
      this.isCreating = false;
    }
  }

  private async refreshLocation(): Promise<void> {
    if (this.isDestroyed) return;
    try {
      const location = await LocationUtil.getCurrentLocation();
      if (this.isDestroyed) return;
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
    if (this.isDestroyed) return;
    if (this.isCreating || this.isGroupAvatarUploading) {
      return;
    }
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selectedUris = await MediaServiceManager.selectImages(context, 1);
      if (this.isDestroyed) return;
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
    if (this.isDestroyed) return undefined;
    if (!this.groupAvatarLocalUri) {
      return undefined;
    }
    if (this.groupAvatarFileId) {
      return this.groupAvatarFileId;
    }
    this.isGroupAvatarUploading = true;
    try {
      const compressedPath = await MediaServiceManager.compressImage(context, this.groupAvatarLocalUri, 80, 1024 * 1024 * 2);
      if (this.isDestroyed) return undefined;
      const uploadUri = await MediaServiceManager.prepareFileForUpload(context, compressedPath);
      if (this.isDestroyed) return undefined;
      const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'group_avatar');
      if (this.isDestroyed) return undefined;
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
    PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint }) {
    Stack() {
      Column() {
        // 标题栏 —— 统一走 SystemPageHeader（居中标题 + 返回）
        SystemPageHeader({
          title: '创建群组',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); },
          titleAlign: TextAlign.Center,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          backgroundVariant: 'transparent'
        })

        // 表单内容
        Scroll() {
          Column({ space: DesignConstants.SPACING.LG }) {
            // 基本信息
            this.SectionHeader('基本信息')
            Column() {
              Column() {
                Text('群组名称')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .width('100%')
                  .margin({ bottom: DesignConstants.SPACING.XS })

                SystemInput({
                  value: this.groupName,
                  placeholder: '请输入群组名称',
                  inputHeight: DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD,
                  maxLength: GROUP_NAME_MAX,
                  hasError: this.getGroupNameError() !== undefined,
                  onChange: (value: string) => { this.groupName = value; },
                  onFocusChange: (focused: boolean) => { this.groupNameFocused = focused; }
                })
                  .shadow(this.fieldBorder(this.groupNameFocused, this.getGroupNameError() !== undefined).shadow)
                  .borderRadius(DesignConstants.RADIUS.XL)
                  .padding({ left: DesignConstants.SPACING.MD })
                  .onFocus(() => { this.groupNameFocused = true; })
                  .onBlur(() => { this.groupNameFocused = false; this.groupNameTouched = true; KeyboardDismissManager.getInstance().dismiss(); })

                if (this.getGroupNameError()) {
                  Text(this.getGroupNameError())
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().error)
                    .width('100%')
                    .margin({ top: DesignConstants.SPACING.XS })
                }
              }
              .alignItems(HorizontalAlign.Start)
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM
              })

              SoftDivider({ color: this.getSurfaceBorder() })
                .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

              Column() {
                Text('群组描述')
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .width('100%')
                  .margin({ bottom: DesignConstants.SPACING.XS })

                SystemTextArea({
                  value: this.groupDescription,
                  placeholder: '请输入群组描述（可选）',
                  minHeight: DesignConstants.COMPONENT.AVATAR_SIZE.LARGE,
                  maxLength: GROUP_DESCRIPTION_MAX,
                  fontSize: DesignConstants.FONT.SIZE.MD,
                  hasError: this.getGroupDescriptionError() !== undefined,
                  onChange: (value: string) => {
                    this.groupDescription = value;
                  },
                  onFocusChange: (focused: boolean) => {
                    this.groupDescriptionFocused = focused;
                    if (!focused) {
                      this.groupDescriptionTouched = true;
                      KeyboardDismissManager.getInstance().dismiss();
                    }
                  }
                })

                if (this.getGroupDescriptionError()) {
                  Text(this.getGroupDescriptionError())
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().error)
                    .width('100%')
                    .margin({ top: DesignConstants.SPACING.XS })
                }
              }
              .alignItems(HorizontalAlign.Start)
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM
              })
            }
            .backgroundColor(this.getSurfaceBackground())
            .borderRadius(DesignConstants.RADIUS.LG)
            .shadow(this.getCurrentThemePalette().elevatedShadow)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
            .margin({ bottom: DesignConstants.SPACING.MD3 })

            // 群组头像
            this.SectionHeader('群组头像')
            Column() {
              Row({ space: DesignConstants.SPACING.MD }) {
                SystemAvatar({
                  avatarUrl: this.groupAvatarPreviewUrl,
                  name: this.groupName || '群组',
                  avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.MEDIUM,
                  shape: 'squircle'
                })

                Column({ space: DesignConstants.SPACING.XS }) {
                  Text('建议使用清晰、易识别的头像')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                  Text(this.isGroupAvatarUploading ? '群头像上传中...' : '未设置时将使用默认群头像')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)

                Row({ space: DesignConstants.SPACING.SM }) {
                  Button(this.groupAvatarPreviewUrl.length > 0 ? '重选' : '选择')
                    .type(ButtonType.Normal)
                    .backgroundColor(this.getSurfaceSubtle())
                    .fontColor(this.getCurrentThemePalette().accentBackground)
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .borderRadius(DesignConstants.RADIUS.LG)
                    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
                    .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_LG)
                    .enabled(!this.isCreating && !this.isGroupAvatarUploading)
                    .onClick(() => this.selectGroupAvatar())

                  if (this.groupAvatarPreviewUrl.length > 0) {
                    Button('移除')
                      .type(ButtonType.Normal)
                      .backgroundColor(this.getSurfaceSubtle())
                      .fontColor(this.getCurrentThemePalette().textSecondary)
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .borderRadius(DesignConstants.RADIUS.LG)
                      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
                      .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_LG)
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
            .backgroundColor(this.getSurfaceBackground())
            .borderRadius(DesignConstants.RADIUS.LG)
            .shadow(this.getCurrentThemePalette().elevatedShadow)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
            .margin({ bottom: DesignConstants.SPACING.MD3 })

            // 位置信息
            this.SectionHeader('位置信息')
            Column() {
              Row() {
                Column() {
                  if (this.localLocation) {
                    Text(this.localLocation.locationName)
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor(this.getCurrentThemePalette().textPrimary)
                      .maxLines(2)
                      .textOverflow({ overflow: TextOverflow.Ellipsis })
                  } else {
                    Text('正在获取位置...')
                      .fontSize(DesignConstants.FONT.SIZE.MD)
                      .fontColor(this.getCurrentThemePalette().textSecondary)
                  }
                }
                .layoutWeight(1)
                .alignItems(HorizontalAlign.Start)

                Row({ space: DesignConstants.SPACING.SM }) {
                  SystemButton({
                    label: '选点',
                    variant: 'surface-accent',
                    buttonSize: 'small',
                    onPress: () => { this.openLocationPicker(); }
                  })

                  SystemButton({
                    label: '刷新',
                    variant: 'surface-accent',
                    buttonSize: 'small',
                    onPress: () => { this.refreshLocation(); }
                  })
                }
              }
              .width('100%')
              .padding({
                left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD
              })
            }
            .backgroundColor(this.getSurfaceBackground())
            .borderRadius(DesignConstants.RADIUS.LG)
            .shadow(this.getCurrentThemePalette().elevatedShadow)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
            .margin({ bottom: DesignConstants.SPACING.MD3 })

            // 隐私与限制
            this.SectionHeader('隐私与限制')
            Column() {
              Row() {
                Text('最大成员数')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                  .layoutWeight(1)

                Row({ space: DesignConstants.SPACING.XS }) {
                  Button() {
                    Text() {
                      SymbolSpan($r(AppIcon.REMOVE))
                        .fontSize(DesignConstants.FONT.SIZE.LG)
                        .fontColor([this.maxMembers > GROUP_MEMBERS_MIN
                          ? this.getCurrentThemePalette().textPrimary
                          : this.getCurrentThemePalette().textTertiary])
                    }
                  }
                  .type(ButtonType.Capsule)
                  .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
                  .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
                  .backgroundColor(this.getSurfaceSubtle())
                  .enabled(this.maxMembers > GROUP_MEMBERS_MIN)
                  .onClick(() => {
                    if (this.maxMembers > GROUP_MEMBERS_MIN) this.maxMembers = Math.max(GROUP_MEMBERS_MIN, this.maxMembers - 10);
                  })

                  TextInput({ text: this.maxMembers.toString() })
                    .type(InputType.Number)
                    .width(DesignConstants.COMPONENT.NUMBER_INPUT_WIDTH)
                    .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
                    .textAlign(TextAlign.Center)
                    .fontSize(DesignConstants.FONT.SIZE.MD)
                    .fontColor(this.getCurrentThemePalette().textPrimary)
                    .backgroundColor(this.getSurfaceSubtle())
                    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.fieldBorder(this.maxMembersFocused, false).borderColor })
                    .shadow(this.fieldBorder(this.maxMembersFocused, false).shadow)
                    .borderRadius(DesignConstants.RADIUS.MD)
                    .padding(0)
                    .onFocus(() => { this.maxMembersFocused = true; })
                    .onChange((value: string) => {
                      const num = parseInt(value);
                      if (!isNaN(num)) {
                        this.maxMembers = Math.max(GROUP_MEMBERS_MIN, Math.min(GROUP_MEMBERS_MAX, num));
                      }
                    })
                    .onBlur(() => {
                      if (this.maxMembers < GROUP_MEMBERS_MIN) this.maxMembers = GROUP_MEMBERS_MIN;
                      if (this.maxMembers > GROUP_MEMBERS_MAX) this.maxMembers = GROUP_MEMBERS_MAX;
                      KeyboardDismissManager.getInstance().dismiss();
                    })

                  Button() {
                    Text() {
                      SymbolSpan($r(AppIcon.ADD))
                        .fontSize(DesignConstants.FONT.SIZE.LG)
                        .fontColor([this.maxMembers < GROUP_MEMBERS_MAX
                          ? this.getCurrentThemePalette().textPrimary
                          : this.getCurrentThemePalette().textTertiary])
                    }
                  }
                  .type(ButtonType.Capsule)
                  .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
                  .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
                  .backgroundColor(this.getSurfaceSubtle())
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

              SoftDivider({ color: this.getSurfaceBorder() })
                .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

              Row() {
                Text('密码保护')
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                  .layoutWeight(1)

                Toggle({ type: ToggleType.Switch, isOn: this.isPasswordRequired })
                  .selectedColor(this.getCurrentThemePalette().accentBackground)
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
                SoftDivider({ color: this.getSurfaceBorder() })
                  .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })

                Column() {
                  Text('群组密码')
                    .fontSize(DesignConstants.FONT.SIZE.SM)
                    .fontColor(this.getCurrentThemePalette().textSecondary)
                    .width('100%')
                    .margin({ bottom: DesignConstants.SPACING.XS })

                  SystemInput({
                    value: this.groupPassword,
                    placeholder: `请输入群组密码（${GROUP_PASSWORD_MIN}-${GROUP_PASSWORD_MAX}位）`,
                    passwordMode: true,
                    inputHeight: DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT_MD,
                    maxLength: GROUP_PASSWORD_MAX,
                    hasError: this.getGroupPasswordError() !== undefined,
                    onChange: (value: string) => { this.groupPassword = value; },
                    onFocusChange: (focused: boolean) => {
                      this.groupPasswordFocused = focused;
                      if (!focused) {
                        this.groupPasswordTouched = true;
                        KeyboardDismissManager.getInstance().dismiss();
                      }
                    }
                  })

                  if (this.getGroupPasswordError()) {
                    Text(this.getGroupPasswordError())
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .fontColor(this.getCurrentThemePalette().error)
                      .width('100%')
                      .margin({ top: DesignConstants.SPACING.XS })
                  }
                }
                .alignItems(HorizontalAlign.Start)
                .padding({
                  left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
                  top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM
                })
              }
            }
            .backgroundColor(this.getSurfaceBackground())
            .borderRadius(DesignConstants.RADIUS.LG)
            .shadow(this.getCurrentThemePalette().elevatedShadow)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
            .margin({ bottom: DesignConstants.SPACING.MD3 })

            // 提交按钮
            SystemButton({
              label: '立即创建',
              variant: 'primary',
              buttonSize: 'large',
              pill: true,
              fullWidth: true,
              disabled: this.isCreating,
              onPress: () => { this.createGroup(); }
            })
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
        .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
      }
      .width('100%')
      .height('100%')

      if (this.isCreating) {
        SystemLoadingView()
          .transition(
            TransitionEffect.OPACITY.animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN })
              .combine(TransitionEffect.move(TransitionEdge.BOTTOM).animation({ duration: DesignConstants.ANIMATION.DURATION_SHEET, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }))
          )
      }
    }
    .width('100%')
    .height('100%')
    }
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
      .fontColor(this.getCurrentThemePalette().textSecondary)
      .margin({ bottom: -DesignConstants.SPACING.XS })
  }
}

@Builder
export function CreateGroupPageBuilder() {
  CreateGroupPage()
}