import { DesignConstants } from '../constants/DesignConstants';
import { UserInfo } from '../core/types/api';
import { UserTypeUtil } from '../core/utils/UserTypeUtil';
import { MediaUrlUtil } from '../core/utils/MediaUrlUtil';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/**
 * SystemAvatar - 头像组件
 * 支持图片、首字母、在线状态、用户类型标识
 */
@Reusable
@Component
export struct SystemAvatar {
  @Prop src: string | Resource | undefined;
  @Prop avatarUrl: string | undefined;
  @Prop name: string = '';
  @Prop avatarSize: number = 40;
  @Prop isOnline: boolean = false;
  @Prop showBorder: boolean = false;
  @Prop shape: 'circle' | 'squircle' | 'square' = 'circle';
  @Prop userInfo: UserInfo | undefined;
  @Prop userType: string = '';
  @Prop inset: boolean = false;
  @State private imageLoadFailed: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getBorderOptions(): BorderOptions {
    // 无描边时返回 0 宽边框而非 undefined：.border(undefined) 在部分 ArkUI 版本
    // 会破坏同链前序属性（实测 borderRadius 失效渲染直角）
    if (!this.showBorder && !this.inset) {
      const none: BorderOptions = { width: 0, color: DesignConstants.COLOR.TRANSPARENT };
      return none;
    }
    const borderOptions: BorderOptions = {
      width: this.inset ? DesignConstants.BORDER_WIDTH.HEAVY : DesignConstants.BORDER_WIDTH.STRONG,
      color: this.getPalette().surfaceBackground
    };
    return borderOptions;
  }

  private getAvatarShadow(): ShadowOptions {
    return this.avatarSize >= 80
      ? AppThemeVisuals.getAvatarHaloShadow(this.resolvedThemeId)
      : AppThemeVisuals.getAvatarShadow(this.resolvedThemeId);
  }

  private getActualSrc(): string | Resource | undefined {
    const source = this.avatarUrl ?? this.src;
    if (typeof source === 'string') {
      return MediaUrlUtil.normalize(source);
    }
    return source;
  }

  @Builder
  private buildAvatarContent() {
    Stack({ alignContent: Alignment.BottomEnd }) {
      if (this.getActualSrc() && !this.imageLoadFailed) {
        Image(this.getActualSrc()!)
          .width(this.avatarSize)
          .height(this.avatarSize)
          .objectFit(ImageFit.Cover)
          .borderRadius(this.getBorderRadius())
          .border(this.getBorderOptions())
          .backgroundColor(this.getPalette().surfaceBackground)
          .alt($r('app.media.foreground'))
          .accessibilityText(this.name ? `头像：${this.name}` : '用户头像')
          .onError(() => {
            this.imageLoadFailed = true;
          })
      } else {
        Flex({ justifyContent: FlexAlign.Center, alignItems: ItemAlign.Center }) {
          Text(this.getInitials(this.name))
            .fontSize(this.avatarSize * 0.4)
            .fontWeight(DesignConstants.FONT.WEIGHT.SEMIBOLD)
            .fontColor(this.getPalette().accentText as ResourceColor)
            .textAlign(TextAlign.Center)
        }
        .width(this.avatarSize)
        .height(this.avatarSize)
        .backgroundColor(this.generateAvatarColor(this.name))
        .borderRadius(this.avatarSize / 2)
        .clip(true)
        .border(this.getBorderOptions())
      }

      if (this.isOnline) {
        Row()
          .width(this.avatarSize * 0.28)
          .height(this.avatarSize * 0.28)
          .backgroundColor(this.getPalette().online)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .border(this.getBorderOptions())
          .shadow(this.getPalette().controlShadow)
          .margin({ bottom: -1, right: -1 })
      } else if (this.userType && UserTypeUtil.isTempUser(this.userType)) {
        Text('临时')
          .fontSize(this.avatarSize * 0.25)
          .fontColor(this.getPalette().textSecondary)
          .backgroundColor(this.getPalette().surfaceSubtle)
          .borderRadius(DesignConstants.RADIUS.SM)
          .padding({ left: DesignConstants.SPACING.HALF, right: DesignConstants.SPACING.HALF })
          .margin({ bottom: -2, right: -2 })
      }
    }
    .width(this.avatarSize)
    .height(this.avatarSize)
    .borderRadius(this.getBorderRadius())
    .shadow(this.getAvatarShadow())
  }

  build() {
    Stack({ alignContent: Alignment.Center }) {
      if (this.avatarSize >= 80) {
        Row()
          .width(this.avatarSize + 24)
          .height(this.avatarSize + 24)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getPalette().heroGlow)
          .opacity(this.getPalette().isDarkTheme ? 0.22 : 0.5)
          .shadow(AppThemeVisuals.getAvatarHaloShadow(this.resolvedThemeId))
      }

      this.buildAvatarContent()
    }
  }

  private getBorderRadius(): Length {
    // 圆形用百分比：部分 ArkUI 版本对超过尺寸的数值圆角（999）不生效，渲染成方角
    if (this.shape === 'circle') return '50%';
    if (this.shape === 'squircle') return (this.avatarSize * 0.22) + 'vp';
    return DesignConstants.RADIUS.SM;
  }

  private getInitials(name: string): string {
    if (!name) return '?';
    return name.substring(0, 2).toUpperCase();
  }

  private generateAvatarColor(name: string): ResourceColor {
    if (!name) return this.getPalette().iconMuted;

    const colors: ResourceColor[] = [
      this.getPalette().accentBackground,
      this.getPalette().surfaceSubtle,
      this.getPalette().buttonSecondaryBackground,
      this.getPalette().iconMuted,
      this.getPalette().borderColor
    ];

    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }

    const index = Math.abs(hash % colors.length);
    return colors[index];
  }
}
