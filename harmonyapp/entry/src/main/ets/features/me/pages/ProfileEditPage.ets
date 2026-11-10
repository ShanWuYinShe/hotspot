/**
 * 编辑资料页面（对齐 web routes/(app)/me/edit/+page.svelte 移动端）
 *
 * 结构：
 *   页头：36 返回钮 +「编辑资料」18/600（SystemPageHeader 统一安全区/返回）
 *   头像卡：68 头像（页底色环）+ 28 accent 相机钮（压右下）+ 昵称 14/500 + @user_id 11 + 说明 11
 *   「基本资料」分区：标题 14/600 + 说明 11（卡外）→ 软岛卡：昵称输入 48 + 简介 textarea + 计数
 *   「更多信息」分区：性别 chips（男/女/其他）+ 兴趣标签 chips（最多 5）
 *   底部通栏 sticky 主按钮「保存资料」（48 高，含底部安全区）。
 *
 * 保存仍走 api_spec_user.md → users/update_account（nickname/bio/tags/gender 的 patch 语义）。
 */

import { AppRouter } from '../../../core/router/AppRouter';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants, lineHeightFor, textLineHeight } from '../../../constants/DesignConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemButton } from '../../../components/SystemButton';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SystemInput } from '../../../components/SystemInput';
import { SystemTextArea } from '../../../components/SystemTextArea';
import { Pressable } from '../../../components/Pressable';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { GlobalStateManager, KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { AccountInfo, Gender } from '../../../core/types/api';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';
import { validateField, FieldRule } from '../../../core/util/FormValidator';
import { ApiClient } from '../../../core/api/ApiClient';
import { LoadingUtil } from '../../../core/utils/LoadingUtil';
import { getErrorMessage } from '../../../core/network/ApiErrorUtil';
import { MediaServiceManager } from '../../../core/media/MediaServiceManager';
import { common } from '@kit.AbilityKit';

const TAG_OPTIONS: string[] = ['摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术', '科技', '游戏', '户外', '时尚', '电影'];
const MAX_TAGS: number = 5;
const MAX_NICKNAME: number = 30;
const MAX_BIO: number = 150;
const BIO_WARN: number = MAX_BIO - 23;

const GENDER_OPTIONS: GenderOption[] = [
  { id: 'male', label: '男' },
  { id: 'female', label: '女' },
  { id: 'other', label: '其他' }
];

interface GenderOption {
  id: Gender;
  label: string;
}

const NICKNAME_RULES: FieldRule[] = [
  { required: true, requiredMessage: '昵称不能为空' },
  { maxLength: MAX_NICKNAME, maxLengthMessage: '昵称长度需在 1-30 字符之间' }
];
const BIO_RULES: FieldRule[] = [
  { maxLength: MAX_BIO, maxLengthMessage: '个人简介不能超过 ' + MAX_BIO.toString() + ' 字符' }
];

@Preview
@Component
export struct ProfileEditPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_CURRENT_USER) currentUser: AccountInfo | null = null;

  @State nickname: string = '';
  @State bio: string = '';
  @State tags: string[] = [];
  @State gender: Gender | null = null;
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
    if (this.currentUser) {
      this.nickname = this.currentUser.nickname || '';
      this.bio = this.currentUser.bio || '';
      this.tags = this.currentUser.tags ? [...this.currentUser.tags] : [];
      this.gender = this.currentUser.gender ?? null;
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

  private toggleTag(tag: string): void {
    const idx = this.tags.indexOf(tag);
    const nextTags: string[] = [];
    if (idx >= 0) {
      for (let i = 0; i < this.tags.length; i++) {
        if (i !== idx) {
          nextTags.push(this.tags[i]);
        }
      }
      this.tags = nextTags;
    } else {
      if (this.tags.length >= MAX_TAGS) {
        SystemToast.showWarning('最多选择 ' + MAX_TAGS.toString() + ' 个标签', this.getUIContext());
        return;
      }
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

  private isTagDisabled(tag: string): boolean {
    return !this.isTagSelected(tag) && this.tags.length >= MAX_TAGS;
  }

  private toggleGender(value: Gender): void {
    this.gender = this.gender === value ? null : value;
  }

  private getNicknameError(): string | undefined {
    return this.nicknameTouched ? validateField(this.nickname, NICKNAME_RULES) : undefined;
  }

  private getBioError(): string | undefined {
    return this.bioTouched ? validateField(this.bio, BIO_RULES) : undefined;
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
    if (this.isSaving || this.isSaved || this.isAvatarUploading) {
      return;
    }
    this.isSaving = true;
    try {
      await LoadingUtil.withMinimumTime(async () => {
        const updateResponse = await ApiClient.updateProfile({
          nickname: trimmedNickname,
          bio: trimmedBio,
          tags: this.tags,
          // UpdateAccountRequest.gender 为 optional（undefined=不修改），null 需转 undefined
          gender: this.gender === null ? undefined : this.gender
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

  /** 分区头（web SectionHead）：标题 14/600 + 说明 11 次色，位于卡外 */
  @Builder
  private buildSectionHead(title: string, description: string) {
    Column({ space: DesignConstants.SPACING.HALF }) {
      Text(title)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
        .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .width('100%')
      Text(description)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.XS, 1.625))
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .width('100%')
    }
    .width('100%')
    .alignItems(HorizontalAlign.Start)
    .margin({ bottom: DesignConstants.SPACING.MD })
  }

  /** 字段标签（web forms/Input label：11 大写字距 0.08em） */
  @Builder
  private buildFieldLabel(label: string) {
    Text(label)
      .fontSize(DesignConstants.FONT.SIZE.XS)
      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
      .letterSpacing(DesignConstants.FONT.LETTER_SPACING.FIELD)
      .textCase(TextCase.UpperCase)
      .fontColor(this.getCurrentThemePalette().textSecondary)
  }

  /** 选择 chip（对齐 web TagPill：h-7 / px-3 / 11 号；选中 accent-soft 底 + accent 边） */
  @Builder
  private buildChip(label: string, active: boolean, disabled: boolean, onTap: () => void) {
    Pressable({
      onPress: () => {
        if (!disabled) {
          onTap();
        }
      },
      pressScale: DesignConstants.INTERACTION.SCALE_PRESSED_PILL,
      hoverFill: this.getCurrentThemePalette().accentMuted
    }) {
      Row() {
        Text(label)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontWeight(active ? DesignConstants.FONT.WEIGHT.SEMIBOLD : DesignConstants.FONT.WEIGHT.REGULAR)
          .fontColor(active ? this.getCurrentThemePalette().textAccentOnSoft : this.getCurrentThemePalette().textSecondary)
          .maxLines(1)
      }
      .height(DesignConstants.COMPONENT.ACTION_BUTTON_HEIGHT)
      .padding({
        left: DesignConstants.SPACING.MD,
        right: DesignConstants.SPACING.MD
      })
      .alignItems(VerticalAlign.Center)
      .justifyContent(FlexAlign.Center)
      .border({
        width: DesignConstants.BORDER_WIDTH.HAIRLINE,
        color: active ? this.getCurrentThemePalette().accentBackground : this.getCurrentThemePalette().borderColor
      })
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(active ? this.getCurrentThemePalette().accentMuted : this.getCurrentThemePalette().surfaceBackground)
      .opacity(disabled ? DesignConstants.OPACITY.DISABLED : 1)
    }
  }

  @Builder
  private buildAvatarCard() {
    Row({ space: DesignConstants.SPACING.LG }) {
      Stack({ alignContent: Alignment.BottomEnd }) {
        SystemAvatar({
          avatarUrl: this.currentUser?.avatar_url ?? '',
          name: this.nickname,
          avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.PROFILE,
          shape: 'circle',
          decorative: true
        })

        Button() {
          LucideIcon({
            icon: AppIcon.CAMERA,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.getCurrentThemePalette().accentText
          })
        }
        .type(ButtonType.Normal)
        .width(DesignConstants.COMPONENT.STATUS_ICON_SIZE)
        .height(DesignConstants.COMPONENT.STATUS_ICON_SIZE)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getCurrentThemePalette().accentBackground)
        .border({ width: DesignConstants.BORDER_WIDTH.HEAVY, color: this.getCurrentThemePalette().surfaceIsland })
        .enabled(!this.isSaving && !this.isAvatarUploading)
        .opacity(this.isAvatarUploading ? DesignConstants.OPACITY.LOADING : 1)
        .onClick(() => { this.changeAvatar(); })
        .accessibilityText('更换头像')
        .responseRegion({
          x: -DesignConstants.SPACING.SM,
          y: -DesignConstants.SPACING.SM,
          width: DesignConstants.LAYOUT.MIN_HIT_TARGET,
          height: DesignConstants.LAYOUT.MIN_HIT_TARGET
        })
      }
      .padding(DesignConstants.SPACING.XS)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getCurrentThemePalette().pageBackground)

      Column({ space: DesignConstants.SPACING.HALF }) {
        Text(this.nickname || (this.currentUser?.nickname ?? '未设置昵称'))
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.MD))
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        if (this.currentUser?.user_id) {
          Text('@' + this.currentUser.user_id)
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
            .fontColor(this.getCurrentThemePalette().textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        Text(this.isAvatarUploading ? '头像上传中…' : '点头像可更换照片')
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .margin({ top: DesignConstants.SPACING.XS })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
    .padding(DesignConstants.SPACING.LG)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
    .shadow(this.getCurrentThemePalette().surfaceShadow)
  }

  build() {
    NavDestination() {
      PageShellContainer({ variant: 'form', currentBreakpoint: this.currentBreakpoint }) {
        Column() {
          SystemPageHeader({
            title: '编辑资料',
            // 对齐 web /me/edit（integrated 移动端页头 18/600）
            titleSize: DesignConstants.FONT.SIZE.LG_PLUS,
            titleWeight: DesignConstants.FONT.WEIGHT.SEMIBOLD,
            showBackButton: true,
            onBack: () => { AppRouter.pop(); }
          })

          Scroll() {
            Column({ space: DesignConstants.SPACING.LG }) {
              this.buildAvatarCard()

              // ===== 基本资料 =====
              this.buildSectionHead('基本资料', '公开显示在你的主页上，附近的人会看到这些信息。')
              Column({ space: DesignConstants.SPACING.LG }) {
                Column({ space: DesignConstants.SPACING.SM }) {
                  this.buildFieldLabel('昵称')
                  SystemInput({
                    value: this.nickname,
                    placeholder: '你的名字',
                    inputHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT_LG,
                    fontSize: DesignConstants.FONT.SIZE.MD,
                    maxLength: MAX_NICKNAME,
                    hasError: this.getNicknameError() !== undefined,
                    onChange: (value: string) => {
                      this.nickname = value;
                      if (this.getNicknameError()) {
                        this.nicknameTouched = false;
                      }
                    },
                    onFocusChange: (focused: boolean) => {
                      this.nicknameFocused = focused;
                      if (!focused) {
                        this.nicknameTouched = true;
                        KeyboardDismissManager.getInstance().dismiss();
                      }
                    }
                  })
                  if (this.getNicknameError()) {
                    Text(this.getNicknameError())
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                      .fontColor(this.getCurrentThemePalette().error)
                      .width('100%')
                  }
                }
                .width('100%')
                .alignItems(HorizontalAlign.Start)

                Column({ space: DesignConstants.SPACING.SM }) {
                  Row() {
                    this.buildFieldLabel('个人简介')
                    Blank()
                    Text(this.bio.length.toString() + '/' + MAX_BIO.toString())
                      .fontSize(DesignConstants.FONT.SIZE.XS)
                      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                      .fontColor(this.bio.length > BIO_WARN
                        ? this.getCurrentThemePalette().warning
                        : this.getCurrentThemePalette().textSecondary)
                  }
                  .width('100%')
                  .alignItems(VerticalAlign.Center)

                  SystemTextArea({
                    value: this.bio,
                    placeholder: '介绍一下自己…',
                    minHeight: DesignConstants.COMPONENT.AVATAR_SIZE.XLARGE + DesignConstants.SPACING.XXL,
                    fontSize: DesignConstants.FONT.SIZE.MD,
                    maxLength: MAX_BIO,
                    hasError: this.getBioError() !== undefined,
                    onChange: (value: string) => {
                      this.bio = value;
                      if (this.getBioError()) {
                        this.bioTouched = false;
                      }
                    },
                    onFocusChange: (focused: boolean) => {
                      this.bioFocused = focused;
                      if (!focused) {
                        this.bioTouched = true;
                        KeyboardDismissManager.getInstance().dismiss();
                      }
                    }
                  })
                  if (this.getBioError()) {
                    Text(this.getBioError())
                      .fontSize(DesignConstants.FONT.SIZE.SM)
                      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.SM))
                      .fontColor(this.getCurrentThemePalette().error)
                      .width('100%')
                  }
                }
                .width('100%')
                .alignItems(HorizontalAlign.Start)
              }
              .width('100%')
              .alignItems(HorizontalAlign.Start)
              .padding(DesignConstants.SPACING.LG)
              .borderRadius(DesignConstants.SHAPE.CARD)
              .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
              .shadow(this.getCurrentThemePalette().surfaceShadow)

              // ===== 更多信息 =====
              this.buildSectionHead('更多信息', '帮助附近的人更快了解你，随时可以修改或清空。')
              Column({ space: DesignConstants.SPACING.LG }) {
                Column({ space: DesignConstants.SPACING.SM }) {
                  this.buildFieldLabel('性别')
                  Flex({ wrap: FlexWrap.Wrap }) {
                    ForEach(GENDER_OPTIONS, (option: GenderOption) => {
                      Column() {
                        this.buildChip(option.label, this.gender === option.id, false, () => {
                          this.toggleGender(option.id);
                        })
                      }
                      .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                    }, (option: GenderOption) => option.id)
                  }
                  .width('100%')
                }
                .width('100%')
                .alignItems(HorizontalAlign.Start)

                Column({ space: DesignConstants.SPACING.SM }) {
                  Row({ space: DesignConstants.SPACING.XS }) {
                    this.buildFieldLabel('兴趣标签')
                    Text('(' + this.tags.length.toString() + '/' + MAX_TAGS.toString() + ')')
                      .fontSize(DesignConstants.FONT.SIZE.XS)
                      .lineHeight(textLineHeight(DesignConstants.FONT.SIZE.XS))
                      .fontColor(this.getCurrentThemePalette().textTertiary)
                  }
                  .width('100%')
                  .alignItems(VerticalAlign.Center)

                  Flex({ wrap: FlexWrap.Wrap }) {
                    ForEach(TAG_OPTIONS, (tag: string) => {
                      Column() {
                        this.buildChip('#' + tag, this.isTagSelected(tag), this.isTagDisabled(tag), () => {
                          this.toggleTag(tag);
                        })
                      }
                      .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                    }, (tag: string) => tag)
                  }
                  .width('100%')
                }
                .width('100%')
                .alignItems(HorizontalAlign.Start)
              }
              .width('100%')
              .alignItems(HorizontalAlign.Start)
              .padding(DesignConstants.SPACING.LG)
              .borderRadius(DesignConstants.SHAPE.CARD)
              .backgroundColor(this.getCurrentThemePalette().surfaceIsland)
              .shadow(this.getCurrentThemePalette().surfaceShadow)
            }
            .width('100%')
            .padding({
              top: DesignConstants.PAGE_SHELL.PAGE_TOP_GAP,
              bottom: DesignConstants.SPACING.XXL
            })
            .alignItems(HorizontalAlign.Start)
          }
          .layoutWeight(1)
          .width('100%')
          .scrollBar(BarState.Off)
          .align(Alignment.Top)

          // ===== 底部通栏 sticky 保存条（web 移动端 nav-glass 底栏） =====
          Column() {
            SystemButton({
              label: this.isSaved ? '已保存' : (this.isSaving ? '保存中…' : '保存资料'),
              variant: this.isSaved ? 'secondary' : 'primary',
              buttonSize: 'large',
              fullWidth: true,
              loading: this.isSaving,
              disabled: this.isSaving || this.isSaved || this.isAvatarUploading,
              onPress: () => { this.handleSave(); }
            })
          }
          .width('100%')
          .padding({
            top: DesignConstants.SPACING.MD,
            bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.LG, DesignConstants.SPACING.LG)
          })
          .backgroundColor(this.getCurrentThemePalette().pageBackground)
          .border({
            width: { top: DesignConstants.BORDER_WIDTH.HAIRLINE },
            color: { top: this.getCurrentThemePalette().borderColor }
          })
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
}

@Builder
export function ProfileEditPageBuilder() {
  ProfileEditPage()
}
