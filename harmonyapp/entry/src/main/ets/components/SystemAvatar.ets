import { DesignConstants } from '../constants/DesignConstants';
import { UserTypeUtil } from '../core/utils/UserTypeUtil';
import { MediaUrlUtil } from '../core/utils/MediaUrlUtil';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

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
  /** 在线角标尺寸（对齐前端 OnlineDot sizeMap：sm 8 / md 10 / lg 12）；
   *  0 = 按头像尺寸自适应（avatarSize*0.28，旧行为，大头像场景保留） */
  @Prop onlineDotSize: number = 0;
  @Prop showBorder: boolean = false;
  @Prop shape: 'circle' | 'squircle' | 'square' = 'circle';
  @Prop userType: string = '';
  @Prop inset: boolean = false;
  @State private imageLoadFailed: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

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

  private getOnlineDotSize(): number {
    return this.onlineDotSize > 0 ? this.onlineDotSize : this.avatarSize * 0.28;
  }

  private getAvatarShadow(): ShadowOptions {
    // 对齐前端 Avatar：统一普通头像投影，无 halo 光晕特例
    return AppThemeVisuals.getAvatarShadow(this.resolvedThemeId);
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
        // 无图占位对齐前端 Avatar.svelte:accentMuted 底 + textSecondary 单首字母
        // (0.42×尺寸)。不再用高饱和色块白字——大尺寸时是廉价感主要来源(UI 审查 D7)
        Flex({ justifyContent: FlexAlign.Center, alignItems: ItemAlign.Center }) {
          Text(this.getInitials(this.name))
            .fontSize(this.avatarSize * 0.42)
            .fontWeight(DesignConstants.FONT.WEIGHT.REGULAR)
            .fontColor(this.getPalette().textSecondary)
            .textAlign(TextAlign.Center)
        }
        .width(this.avatarSize)
        .height(this.avatarSize)
        .backgroundColor(this.getPalette().accentMuted)
        .borderRadius(this.avatarSize / 2)
        .clip(true)
        .border(this.getBorderOptions())
      }

      if (this.isOnline) {
        // 对齐前端 OnlineDot：absolute bottom-0 right-0 贴角（无负外扩）+ 固定档尺寸
        // + border-strong/surface-primary 描边把角标与头像图分开
        Row()
          .width(this.getOnlineDotSize())
          .height(this.getOnlineDotSize())
          .backgroundColor(this.getPalette().online)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .border({ width: DesignConstants.BORDER_WIDTH.STRONG, color: this.getPalette().surfaceBackground })
          .shadow(this.getPalette().controlShadow)
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
    // 对齐前端 shared/Avatar.svelte：无 halo 光晕层（前端去炫光治理已删除 glow，
    // 大尺寸头像同样只保留白卡/描边语言），仅头像本体 + 状态点
    Stack({ alignContent: Alignment.Center }) {
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
    // 对齐前端 Avatar.svelte:仅取首字符(原双字符+白字已随占位克制化移除)
    if (!name) return '?';
    return name.substring(0, 1).toUpperCase();
  }
}

