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
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'figma_neu_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getBorderOptions(): BorderOptions | undefined {
    if (!this.showBorder && !this.inset) {
      return undefined;
    }
    const borderOptions: BorderOptions = {
      width: this.inset ? 3 : 2,
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
            .width(this.avatarSize)
            .height(this.avatarSize)
            .backgroundColor(this.generateAvatarColor(this.name))
            .borderRadius(this.getBorderRadius())
            .border(this.getBorderOptions())
        }
      }

      if (this.isOnline) {
        Row()
          .width(this.avatarSize * 0.28)
          .height(this.avatarSize * 0.28)
          .backgroundColor(this.getPalette().accentBackground)
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
          .padding({ left: 2, right: 2 })
          .margin({ bottom: -2, right: -2 })
      }
    }
    .width(this.avatarSize)
    .height(this.avatarSize)
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
    if (this.shape === 'circle') return DesignConstants.RADIUS.FULL;
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
