/**
 * 编辑资料页面
 * 包含头像编辑、昵称、简介、兴趣标签、保存按钮
 */

import { AppRouter } from '../../../core/router/AppRouter';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemButton } from '../../../components/SystemButton';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader, PageHeaderAction } from '../../../components/SystemPageHeader';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppIcon } from '../../../components/SystemIcon';
import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { AccountInfo } from '../../../core/types/api';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { AppThemeVisuals, InputBorderResolution } from '../../../core/theme/AppThemeVisuals';
import { validateField, FieldRule } from '../../../core/util/FormValidator';
import { ApiClient } from '../../../core/api/ApiClient';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { common } from '@kit.AbilityKit';
import { LucideIcon } from '../../../components/LucideIcon';

const TAG_OPTIONS: string[] = ['摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术', '科技', '游戏', '户外', '时尚', '电影'];
const MAX_TAGS: number = 5;
const MAX_NICKNAME: number = 30;
const MAX_BIO: number = 150;

const NICKNAME_RULES: FieldRule[] = [
  { required: true, requiredMessage: '昵称不能为空' },
  { maxLength: MAX_NICKNAME, maxLengthMessage: '昵称长度需在 1-30 字符之间' }
];
const BIO_RULES: FieldRule[] = [
  { maxLength: MAX_BIO, maxLengthMessage: `个人简介不能超过 ${MAX_BIO} 字符` }
];

@Preview
@Component
export struct ProfileEditPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;

  @State nickname: string = '';
  @State bio: string = '';
  @State tags: string[] = [];
  @State isSaving: boolean = false;
  @State isSaved: boolean = false;
  @State nicknameFocused: boolean = false;
  @State bioFocused: boolean = false;
  private saveAnimTimer: number = -1;
  @State nicknameTouched: boolean = false;
  @State bioTouched: boolean = false;
  @State isAvatarUploading: boolean = false;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
    // 初始化表单数据
    if (this.currentUser) {
      this.nickname = this.currentUser.nickname || '';
      this.bio = this.currentUser.bio || '';
      this.tags = this.currentUser.tags ? [...this.currentUser.tags] : [];
    }
  }

  aboutToDisappear(): void {
    KeyboardDismissManager.getInstance().dismiss();
    if (this.saveAnimTimer !== -1) {
      clearTimeout(this.saveAnimTimer);
      this.saveAnimTimer = -1;
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

  private toggleTag(tag: string): void {
    const idx = this.tags.indexOf(tag);
    const nextTags: string[] = [];
    if (idx >= 0) {
      // 移除标签
      for (let i = 0; i < this.tags.length; i++) {
        if (i !== idx) {
          nextTags.push(this.tags[i]);
        }
      }
      this.tags = nextTags;
    } else {
      if (this.tags.length >= MAX_TAGS) {
        SystemToast.showWarning(`最多选择 ${MAX_TAGS} 个标签`, this.getUIContext());
        return;
      }
      // 添加标签
      for (let i = 0; i < this.tags.length; i++) {
        nextTags.push(this.tags[i]);
      }
      nextTags.push(tag);
      this.tags = nextTags;
    }
  }

  private isTagSelected(tag: string): boolean {
    return this.tags.indexOf(tag) >= 0;
  }

  private canSelectMoreTags(): boolean {
    return this.tags.length < MAX_TAGS;
  }

  private isTagDisabled(tag: string): boolean {
    return !this.isTagSelected(tag) && !this.canSelectMoreTags();
  }

  private async handleSave(): Promise<void> {
    this.nicknameTouched = true;
    this.bioTouched = true;
    const trimmedNickname = this.nickname.trim();
    const trimmedBio = this.bio.trim();
    if (!trimmedNickname) {
      SystemToast.showWarning('昵称不能为空', this.getUIContext());
      return;
    }
    const nicknameError = validateField(trimmedNickname, NICKNAME_RULES);
    if (nicknameError) {
      SystemToast.showWarning(nicknameError, this.getUIContext());
      return;
    }
    const bioError = validateField(trimmedBio, BIO_RULES);
    if (bioError) {
      SystemToast.showWarning(bioError, this.getUIContext());
      return;
    }
    if (this.isSaving || this.isSaved) {
      return;
    }
    this.isSaving = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const updateResponse = await ApiClient.updateProfile({
          nickname: trimmedNickname,
          bio: trimmedBio,
          tags: this.tags
        });
        if (updateResponse.code !== 0) {
          throw new Error(getErrorMessage(updateResponse, '资料更新失败'));
        }
        const accountResponse = await ApiClient.getAccountInfo();
        if (accountResponse.code === 0 && accountResponse.data) {
          GlobalStateManager.getInstance().updateUser(accountResponse.data);
        }
      }, LoadingUtil.DEFAULT_ACTION_MS);
      this.isSaving = false;
      this.isSaved = true;
      SystemToast.showSuccess('资料已保存', this.getUIContext());
      this.saveAnimTimer = setTimeout(() => {
        this.isSaved = false;
        AppRouter.pop();
      }, 600);
    } catch (error) {
      this.isSaving = false;
      const message = error instanceof Error ? error.message : '更新失败';
      SystemToast.showError(message, this.getUIContext());
    }
  }

  /**
   * 更换头像：选图 → 压缩 → 上传 → 更新资料 → 刷新本地用户状态。
   * 与 ProfilePage.changeAvatarDirectly 同一套上传流程。
   */
  private async changeAvatar(): Promise<void> {
    if (this.isSaving || this.isAvatarUploading) {
      return;
    }
    try {
      const context = this.getUIContext().getHostContext() as common.UIAbilityContext;
      const selectedUris = await MediaServiceManager.selectImages(context, 1);
      if (selectedUris.length === 0) {
        return;
      }
      this.isAvatarUploading = true;
      const compressedPath = await MediaServiceManager.compressImage(context, selectedUris[0], 80, 1024 * 1024 * 2);
      const uploadUri = await MediaServiceManager.prepareFileForUpload(context, compressedPath);
      const uploadResult = await ApiClient.uploadFile(context, uploadUri, 'avatar');
      if (!uploadResult.file_id) {
        throw new Error('头像上传失败');
      }
      const updateResponse = await ApiClient.updateProfile({ avatar_file_id: uploadResult.file_id });
      if (updateResponse.code !== 0) {
        throw new Error(getErrorMessage(updateResponse, '头像更新失败'));
      }
      const accountResponse = await ApiClient.getAccountInfo();
      if (accountResponse.code === 0 && accountResponse.data) {
        GlobalStateManager.getInstance().updateUser(accountResponse.data);
      }
      SystemToast.showSuccess('头像已更新', this.getUIContext());
    } catch (error) {
      const message = error instanceof Error ? error.message : '头像更换失败，请重试';
      SystemToast.showError(message, this.getUIContext());
    } finally {
      this.isAvatarUploading = false;
    }
  }

  private getHeaderAction(): PageHeaderAction | null {
    return {
      label: this.isSaved ? '' : (this.isSaving ? '保存中…' : '保存'),
      onClick: () => this.handleSave(),
      disabled: this.isSaving || this.isSaved
    };
  }

  build() {
    NavDestination() {
    PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint }) {
      Column() {
        SystemPageHeader({
          title: '编辑资料',
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          showBackButton: true,
          onBack: () => { AppRouter.pop(); },
          action: this.isSaving || this.isSaved ? null : this.getHeaderAction()
        })

        Scroll() {
          Column({ space: DesignConstants.SPACING.LG }) {
            // Banner 占位（渐变装饰）。
            // 注：后端用户模型（core/types/api/user.ets 的 AccountInfo / UpdateAccountRequest）
            // 暂无 banner 字段，「更换横幅」入口已移除（原「开发中」Toast 一并删除），
            // 待后端支持 banner 字段后恢复。
            Column()
              .width('100%')
              .height(DesignConstants.COMPONENT.BANNER_HEIGHT)
              .borderRadius({ topLeft: DesignConstants.SHAPE.SHEET_TOP_RADIUS, topRight: DesignConstants.SHAPE.SHEET_TOP_RADIUS })
              .linearGradient({
                direction: GradientDirection.Bottom,
                colors: [
                  [AppThemeVisuals.withAlpha(this.getCurrentThemePalette().accentBackground, 0.38), 0.0],
                  [AppThemeVisuals.withAlpha(this.getCurrentThemePalette().accentBackground, 0.12), 1.0]
                ]
              })

            // 头像 + 名称区（对齐前端 me/edit：Avatar 72 rounded + 昵称 text-md + @userId）
            Row({ space: DesignConstants.SPACING.MD }) {
              Stack({ alignContent: Alignment.BottomEnd }) {
                SystemAvatar({
                  avatarUrl: this.currentUser?.avatar_url ?? '',
                  name: this.nickname || this.currentUser?.nickname || '',
                  avatarSize: 72,
                  shape: 'circle'
                })

                // 相机编辑按钮
                Button() {
                  LucideIcon({
                      icon: AppIcon.CAMERA,
                      iconSize: DesignConstants.FONT.SIZE.SM,
                      color: this.getCurrentThemePalette().accentText
                    })
                }
                .type(ButtonType.Normal)
                .width(DesignConstants.COMPONENT.STATUS_ICON_SIZE + DesignConstants.SPACING.XS)
                .height(DesignConstants.COMPONENT.STATUS_ICON_SIZE + DesignConstants.SPACING.XS)
                .borderRadius(DesignConstants.RADIUS.FULL)
                .backgroundColor(this.getCurrentThemePalette().accentBackground)
                .border({ width: DesignConstants.BORDER_WIDTH.HEAVY, color: this.getSurfaceBackground() })
                .enabled(!this.isSaving && !this.isAvatarUploading)
                .opacity(this.isAvatarUploading ? DesignConstants.OPACITY.LOADING : 1)
                .onClick(() => {
                  this.changeAvatar();
                })
                .accessibilityText('编辑头像')
                .responseRegion({
                  x: -DesignConstants.SPACING.SM,
                  y: -DesignConstants.SPACING.SM,
                  width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
                  height: DesignConstants.LAYOUT.MIN_HIT_TARGET
                })
              }

              Column({ space: DesignConstants.SPACING.HALF }) {
                Text(this.nickname || (this.currentUser?.nickname ?? ''))
                  .fontSize(DesignConstants.FONT.SIZE.MD)
                  .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
                  .fontColor(this.getCurrentThemePalette().textPrimary)

                Text('@' + (this.currentUser?.user_id?.substring(0, 8) ?? 'user'))
                  .fontSize(DesignConstants.FONT.SIZE.SM)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)
              .margin({ top: DesignConstants.SPACING.LG })
            }
            .width('100%')
            // 头像区上移叠入 hero 头图（负 margin 特例收口到 DesignConstants，审查 P3-3）
            .margin({ top: DesignConstants.COMPONENT.PROFILE_AVATAR_HEADER_OVERLAP })
            .padding({ left: DesignConstants.SPACING.LG })

            // 表单字段
            this.buildInputField('昵称', '你的名字', this.nickname, MAX_NICKNAME, false, (v: string) => { this.nickname = v; }, this.nicknameFocused, this.getNicknameError() ?? '', () => { this.nicknameFocused = true; }, () => { this.nicknameFocused = false; this.nicknameTouched = true; }, false)

            this.buildInputField('个人简介', '介绍一下自己…', this.bio, MAX_BIO, true, (v: string) => { this.bio = v; }, this.bioFocused, this.getBioError() ?? '', () => { this.bioFocused = true; }, () => { this.bioFocused = false; this.bioTouched = true; }, true)

            // 兴趣标签
            Column({ space: DesignConstants.SPACING.SM }) {
              Row({ space: DesignConstants.SPACING.XS }) {
                Text('兴趣标签')
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .textCase(TextCase.UpperCase)
                  .letterSpacing(DesignConstants.FONT.LETTER_SPACING.FIELD)

                Text('(' + this.tags.length + '/' + MAX_TAGS + ')')
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .fontColor(this.getCurrentThemePalette().textTertiary)
              }
              .width('100%')

              Flex({ wrap: FlexWrap.Wrap }) {
                ForEach(TAG_OPTIONS, (tag: string) => {
                  SystemButton({
                    label: '#' + tag,
                    variant: this.isTagSelected(tag) ? 'primary' : 'secondary',
                    buttonSize: 'small',
                    pill: true,
                    disabled: this.isTagDisabled(tag),
                    onPress: () => {
                      this.toggleTag(tag);
                    }
                  })
                    .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                }, (tag: string) => tag)
              }
              .width('100%')
            }
            .width('100%')

            // 底部保存按钮
            SystemButton({
              label: this.isSaved ? '已保存' : (this.isSaving ? '保存中…' : '保存'),
              variant: this.isSaved ? 'secondary' : 'primary',
              buttonSize: 'large',
              fullWidth: true,
              pill: true,
              loading: this.isSaving,
              disabled: this.isSaving || this.isSaved,
              onPress: () => this.handleSave()
            })
          }
          .width('100%')
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.MD,
            bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.XXL, DesignConstants.SPACING.LG)
          })
          .alignItems(HorizontalAlign.Start)
        }
        .layoutWeight(1)
        .width('100%')
        .scrollBar(BarState.Off)
      }
      .width('100%')
      .height('100%')
    }
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  private getNicknameError(): string | undefined {
    return this.nicknameTouched ? validateField(this.nickname, NICKNAME_RULES) : undefined;
  }

  private getBioError(): string | undefined {
    return this.bioTouched ? validateField(this.bio, BIO_RULES) : undefined;
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

  @Builder
  private buildInputField(label: string, placeholder: string, value: string, maxLen: number, multiline: boolean, onChange: (v: string) => void, focused: boolean, error: string, onFocus: () => void, onBlur: () => void, showCounter: boolean = true) {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row({ space: DesignConstants.SPACING.XS }) {
        LucideIcon({
            icon: AppIcon.EDIT,
            iconSize: DesignConstants.FONT.SIZE.SM,
            color: this.getCurrentThemePalette().accentBackground
          })
        Text(label)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .textCase(TextCase.UpperCase)
          .letterSpacing(DesignConstants.FONT.LETTER_SPACING.FIELD)
      }
      .width('100%')

      Column() {
        if (multiline) {
          TextArea({ placeholder: placeholder, text: value })
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .placeholderColor(this.getCurrentThemePalette().textTertiary)
            .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
            .background(DesignConstants.COLOR.TRANSPARENT)
            .border({ width: 0 })
            .borderRadius(0)
            .maxLength(maxLen)
            .onChange((v: string) => { onChange(v); })
            .onFocus(() => { onFocus(); })
            .onBlur(() => { onBlur(); KeyboardDismissManager.getInstance().dismiss(); })
        } else {
          TextInput({ placeholder: placeholder, text: value })
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .placeholderColor(this.getCurrentThemePalette().textTertiary)
            .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
            .background(DesignConstants.COLOR.TRANSPARENT)
            .border({ width: 0 })
            .borderRadius(0)
            .maxLength(maxLen)
            .onChange((v: string) => { onChange(v); })
            .onFocus(() => { onFocus(); })
            .onBlur(() => { onBlur(); KeyboardDismissManager.getInstance().dismiss(); })
        }

        if (showCounter) {
          // 计数变色阈值 127/150 对齐前端 me/edit（bio.length > 127 起 accent 色）
          Text(value.length + '/' + maxLen)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(value.length > MAX_BIO - 23
              ? this.getCurrentThemePalette().accentBackground
              : this.getCurrentThemePalette().textTertiary)
            .width('100%')
            .textAlign(TextAlign.End)
            .margin({ top: DesignConstants.SPACING.XS })
        }

        if (error.length > 0) {
          Text(error)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.getCurrentThemePalette().error)
            .width('100%')
            .margin({ top: DesignConstants.SPACING.XS })
        }
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.MD2, bottom: DesignConstants.SPACING.MD2 })
      .backgroundColor(this.getSurfaceSubtle())
      .borderRadius(DesignConstants.SHAPE.INPUT)
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.fieldBorder(focused, error.length > 0).borderColor })
      .shadow(this.fieldBorder(focused, error.length > 0).shadow)
      // 裁剪子组件（TextInput/TextArea），防止系统默认视觉溢出圆角
      .clip(true)
    }
    .width('100%')
  }
}

@Builder
export function ProfileEditPageBuilder() {
  ProfileEditPage()
}