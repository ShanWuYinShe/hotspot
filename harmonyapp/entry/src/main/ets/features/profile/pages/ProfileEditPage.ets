/**
 * 编辑资料页面
 * Figma Make EditProfile 对齐
 * 包含头像编辑、昵称、简介、兴趣标签、保存按钮
 */

import { AppRouter } from '../../../core/router/AppRouter';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemToast } from '../../../components/SystemToast';
import { SystemPageHeader, PageHeaderAction } from '../../../components/SystemPageHeader';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppIcon } from '../../../components/SystemIcon';
import { KEY_CURRENT_USER } from '../../../core/state/GlobalStateManager';
import { MyProfile } from '../../../core/types/api';
import { KeyboardDismissManager } from '../../../core/utils/KeyboardDismissManager';

const TAG_OPTIONS: string[] = ['摄影', '旅行', '音乐', '美食', '运动', '阅读', '艺术', '科技', '游戏', '户外', '时尚', '电影'];
const MAX_TAGS: number = 5;
const MAX_NICKNAME: number = 30;
const MAX_BIO: number = 150;

@Preview
@Component
export struct ProfileEditPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;
  @StorageLink(KEY_CURRENT_USER) currentUser: MyProfile | null = null;

  @State nickname: string = '';
  @State bio: string = '';
  @State tags: string[] = [];
  @State isSaving: boolean = false;
  @State isSaved: boolean = false;

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

  private handleSave(): void {
    const trimmedNickname = this.nickname.trim();
    if (!trimmedNickname) {
      SystemToast.showWarning('昵称不能为空', this.getUIContext());
      return;
    }

    this.isSaving = true;
    // 模拟保存动画
    setTimeout(() => {
      this.isSaving = false;
      this.isSaved = true;
      SystemToast.showSuccess('资料已保存', this.getUIContext());
      setTimeout(() => {
        this.isSaved = false;
        AppRouter.pop();
      }, 600);
    }, 400);
  }

  private getHeaderAction(): PageHeaderAction | null {
    return {
      label: this.isSaved ? '' : (this.isSaving ? '保存中...' : '保存'),
      onClick: () => this.handleSave(),
      disabled: this.isSaving || this.isSaved
    };
  }

  build() {
    NavDestination() {
      Column() {
        SystemPageHeader({
          title: '编辑资料',
          showBackButton: true,
          onBack: () => { AppRouter.pop(); },
          action: this.isSaving || this.isSaved ? null : this.getHeaderAction()
        })

        Scroll() {
          Column({ space: DesignConstants.SPACING.LG }) {
            // Banner 占位（使用 Stack 代替 overlay builder 以确保兼容性）
            Stack({ alignContent: Alignment.TopStart }) {
              Column()
                .width('100%')
                .height(140)
                .borderRadius({ topLeft: 20, topRight: 20 })
                .linearGradient({
                  direction: GradientDirection.Bottom,
                  colors: [
                    [this.getCurrentThemePalette().accentBackground + '60', 0.0],
                    [this.getCurrentThemePalette().accentBackground + '20', 1.0]
                  ]
                })

              // 更换横幅按钮：贴横幅右上角（用 alignContent 取代硬编码坐标）
              Stack({ alignContent: Alignment.TopEnd }) {
                Button() {
                  Row({ space: 4 }) {
                    Text() {
                      SymbolSpan($r(AppIcon.CAMERA))
                        .fontSize(12)
                        .fontColor([Color.White])
                    }
                    Text('更换横幅')
                      .fontSize(12)
                      .fontColor(Color.White)
                  }
                }
                .type(ButtonType.Normal)
                .padding({ left: 10, right: 10, top: 5, bottom: 5 })
                .borderRadius(DesignConstants.SHAPE.BUTTON)
                .backgroundColor(DesignConstants.COLOR.OVERLAY_STRONG)
                .onClick(() => {
                  SystemToast.show('横幅更换功能开发中', this.getUIContext());
                })
              }
              .width('100%')
              .height('100%')
              .padding({ top: 10, right: 12 })
            }
            .width('100%')
            .height(140)
            .borderRadius({ topLeft: 20, topRight: 20 })

            // 头像 + 名称区
            Row({ space: DesignConstants.SPACING.MD }) {
              Stack({ alignContent: Alignment.BottomEnd }) {
                SystemAvatar({
                  avatarUrl: this.currentUser?.avatar_url ?? '',
                  name: this.nickname || this.currentUser?.nickname || '',
                  avatarSize: 72,
                  shape: 'squircle'
                })

                // 相机编辑按钮
                Button() {
                  Text() {
                    SymbolSpan($r(AppIcon.CAMERA))
                      .fontSize(10)
                      .fontColor([Color.White])
                  }
                }
                .type(ButtonType.Normal)
                .width(26)
                .height(26)
                .borderRadius(DesignConstants.RADIUS.FULL)
                .backgroundColor(this.getCurrentThemePalette().accentBackground)
                .border({ width: 2.5, color: this.getSurfaceBackground() })
                .onClick(() => {
                  SystemToast.show('头像更换功能开发中', this.getUIContext());
                })
              }

              Column({ space: 2 }) {
                Text(this.nickname || (this.currentUser?.nickname ?? ''))
                  .fontSize(18)
                  .fontWeight(FontWeight.Bold)
                  .fontColor(this.getCurrentThemePalette().textPrimary)

                Text('@' + (this.currentUser?.user_id?.substring(0, 8) ?? 'user'))
                  .fontSize(12)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
              }
              .alignItems(HorizontalAlign.Start)
              .layoutWeight(1)
              .margin({ top: DesignConstants.SPACING.LG })
            }
            .width('100%')
            .margin({ top: -40 })
            .padding({ left: DesignConstants.SPACING.LG })

            // 表单字段
            this.buildInputField('昵称', '你的名字', this.nickname, MAX_NICKNAME, false, (v: string) => { this.nickname = v; })

            Column({ space: DesignConstants.SPACING.SM }) {
              Row({ space: 4 }) {
                Text() {
                  SymbolSpan($r(AppIcon.EDIT))
                    .fontSize(12)
                    .fontColor([this.getCurrentThemePalette().accentBackground])
                }
                Text('个人简介')
                  .fontSize(11)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .textCase(TextCase.UpperCase)
                  .letterSpacing(0.08 * 16)
              }
              .width('100%')

              Column() {
                TextArea({ placeholder: '介绍一下自己…', text: this.bio })
                  .fontSize(14)
                  .fontColor(this.getCurrentThemePalette().textPrimary)
                  .backgroundColor(Color.Transparent)
                  .borderRadius(0)
                  .maxLength(MAX_BIO)
                  .onChange((v: string) => { this.bio = v; })
                  .onBlur(() => KeyboardDismissManager.getInstance().dismiss())

                Text(this.bio.length + '/' + MAX_BIO)
                  .fontSize(11)
                  .fontColor(this.bio.length > MAX_BIO * 0.85
                    ? this.getCurrentThemePalette().accentBackground
                    : this.getCurrentThemePalette().textTertiary)
                  .width('100%')
                  .textAlign(TextAlign.End)
                  .margin({ top: 4 })
              }
              .width('100%')
              .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
              .backgroundColor(this.getSurfaceSubtle())
              .borderRadius(DesignConstants.SHAPE.INPUT)
            }
            .width('100%')

            // 兴趣标签
            Column({ space: DesignConstants.SPACING.SM }) {
              Row({ space: 4 }) {
                Text('兴趣标签')
                  .fontSize(11)
                  .fontColor(this.getCurrentThemePalette().textSecondary)
                  .textCase(TextCase.UpperCase)
                  .letterSpacing(0.08 * 16)

                Text('(' + this.tags.length + '/' + MAX_TAGS + ')')
                  .fontSize(11)
                  .fontColor(this.getCurrentThemePalette().textTertiary)
              }
              .width('100%')

              Flex({ wrap: FlexWrap.Wrap }) {
                ForEach(TAG_OPTIONS, (tag: string) => {
                  Button() {
                    Text('#' + tag)
                      .fontSize(12)
                      .fontColor(this.isTagSelected(tag) ? Color.White : (this.isTagDisabled(tag) ? this.getCurrentThemePalette().textTertiary : this.getCurrentThemePalette().textSecondary))
                  }
                  .type(ButtonType.Normal)
                  .padding({ left: 12, right: 12, top: 6, bottom: 6 })
                  .borderRadius(DesignConstants.SHAPE.BUTTON)
                  .backgroundColor(this.isTagSelected(tag) ? this.getCurrentThemePalette().accentBackground : this.getSurfaceBackground())
                  .border({ width: 1, color: this.isTagSelected(tag) ? Color.Transparent : this.getSurfaceBorder() })
                  .opacity(this.isTagDisabled(tag) ? 0.4 : 1.0)
                  .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
                  .enabled(!this.isTagDisabled(tag))
                  .onClick(() => {
                    this.toggleTag(tag);
                  })
                }, (tag: string) => tag)
              }
              .width('100%')
            }
            .width('100%')

            // 底部保存按钮
            Button() {
              Row({ space: DesignConstants.SPACING.SM }) {
                if (this.isSaved) {
                  Text() {
                    SymbolSpan($r(AppIcon.CHECK))
                      .fontSize(16)
                      .fontColor([Color.White])
                  }
                  Text('已保存')
                    .fontSize(15)
                    .fontColor(Color.White)
                } else {
                  Text(this.isSaving ? '保存中...' : '保存')
                    .fontSize(15)
                    .fontColor(Color.White)
                    .fontWeight(FontWeight.Medium)
                }
              }
            }
            .type(ButtonType.Normal)
            .width('100%')
            .height(48)
            .borderRadius(DesignConstants.RADIUS.FULL)
            .backgroundColor(this.isSaved ? DesignConstants.COLOR.SUCCESS : this.getCurrentThemePalette().accentBackground)
            .shadow(this.getCurrentThemePalette().accentShadow)
            .enabled(!this.isSaving && !this.isSaved)
            .onClick(() => this.handleSave())
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
        .scrollBar(BarState.Auto)
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder
  private buildInputField(label: string, placeholder: string, value: string, maxLen: number, multiline: boolean, onChange: (v: string) => void) {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row({ space: 4 }) {
        Text() {
          SymbolSpan($r(AppIcon.EDIT))
            .fontSize(12)
            .fontColor([this.getCurrentThemePalette().accentBackground])
        }
        Text(label)
          .fontSize(11)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .textCase(TextCase.UpperCase)
          .letterSpacing(0.08 * 16)
      }
      .width('100%')

      Column() {
        if (multiline) {
          TextArea({ placeholder: placeholder, text: value })
            .fontSize(14)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .backgroundColor(Color.Transparent)
            .borderRadius(0)
            .maxLength(maxLen)
            .onChange((v: string) => { onChange(v); })
            .onBlur(() => KeyboardDismissManager.getInstance().dismiss())
        } else {
          TextInput({ placeholder: placeholder, text: value })
            .fontSize(14)
            .fontColor(this.getCurrentThemePalette().textPrimary)
            .backgroundColor(Color.Transparent)
            .borderRadius(0)
            .maxLength(maxLen)
            .onChange((v: string) => { onChange(v); })
            .onBlur(() => KeyboardDismissManager.getInstance().dismiss())
        }

        Text(value.length + '/' + maxLen)
          .fontSize(11)
          .fontColor(value.length > maxLen * 0.85
            ? this.getCurrentThemePalette().accentBackground
            : this.getCurrentThemePalette().textTertiary)
          .width('100%')
          .textAlign(TextAlign.End)
          .margin({ top: 4 })
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: 10, bottom: 10 })
      .backgroundColor(this.getSurfaceSubtle())
      .borderRadius(DesignConstants.SHAPE.INPUT)
    }
    .width('100%')
  }
}

@Builder
export function ProfileEditPageBuilder() {
  ProfileEditPage()
}
