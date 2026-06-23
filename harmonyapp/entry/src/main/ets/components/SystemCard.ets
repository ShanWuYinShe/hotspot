import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { curves } from '@kit.ArkUI';

/**
 * 卡片样式变体
 */
export enum CardVariant {
  DEFAULT = 'default',      // 默认样式
  ELEVATED = 'elevated',    // 悬浮样式（更强阴影）
  OUTLINED = 'outlined',    // 边框样式
  FILLED = 'filled',        // 填充样式
  GLASS = 'glass',          // 毛玻璃样式
  GLOW = 'glow',            // 柔光浮雕样式
}

// 阴影配置接口
interface ShadowConfig {
  radius: number;
  color: string;
  offsetX: number;
  offsetY: number;
}

// 边框配置接口
interface BorderConfig {
  width: number;
  color: ResourceColor;
}

@Reusable
@Component
export struct SystemCard {
  @BuilderParam content: () => void;
  @Prop cardPadding: number = DesignConstants.SPACING.LG;
  @Prop showShadow: boolean = true;
  @Prop clickable: boolean = false;
  @Prop enableHoverEffect: boolean = false;

  @Prop paddingSize: number | string = DesignConstants.SPACING.LG;
  @Prop cardBackgroundColor: ResourceColor | string = '';
  @Prop useBlur: boolean = false;

  // 新增：卡片样式变体
  @Prop variant: CardVariant = CardVariant.DEFAULT;
  // 新增：圆角大小（重命名为 cardBorderRadius 避免冲突）
  @Prop cardBorderRadius: number = DesignConstants.SHAPE.CARD;
  // 新增：是否显示边框
  @Prop showBorder: boolean = true;
  // 新增：边框颜色（使用系统颜色）
  @Prop cardBorderColor: ResourceColor | string = '';

  @State isPressed: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private isNeuTheme(): boolean {
    return this.resolvedThemeId.startsWith('figma_neu');
  }

  private getThemeShadow(kind: 'default' | 'elevated'): ShadowConfig {
    const palette = this.getPalette();
    return kind === 'elevated' ? palette.elevatedShadow : palette.surfaceShadow;
  }

  private getNeuHighlightShadow(): ShadowConfig {
    const effects = this.getPalette().visuals.effects;
    return { radius: effects.highlightShadow.radius, color: effects.highlightShadow.color, offsetX: effects.highlightShadow.offsetX, offsetY: effects.highlightShadow.offsetY };
  }

  private getNeuInsetShadow(): ShadowConfig {
    const effects = this.getPalette().visuals.effects;
    return { radius: effects.insetShadow.radius, color: effects.insetShadow.color, offsetX: effects.insetShadow.offsetX, offsetY: effects.insetShadow.offsetY };
  }

  private getBackgroundColor(): ResourceColor {
    if (this.cardBackgroundColor) {
      return this.cardBackgroundColor as ResourceColor;
    }

    switch (this.variant) {
      case CardVariant.GLASS:
        return Color.Transparent;
      case CardVariant.FILLED:
        return this.getPalette().surfaceSubtle;
      case CardVariant.GLOW:
        return this.getPalette().surfaceElevated;
      case CardVariant.ELEVATED:
      case CardVariant.OUTLINED:
      case CardVariant.DEFAULT:
      default:
        return this.getPalette().surfaceBackground;
    }
  }

  private getShadow(): ShadowConfig {
    if (!this.showShadow) {
      return { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 };
    }

    switch (this.variant) {
      case CardVariant.ELEVATED:
        return this.getThemeShadow('elevated');
      case CardVariant.GLASS:
        return this.getThemeShadow('elevated');
      case CardVariant.GLOW:
        return this.getPalette().accentShadow;
      case CardVariant.DEFAULT:
      case CardVariant.FILLED:
      case CardVariant.OUTLINED:
      default:
        return this.getThemeShadow('default');
    }
  }

  private getBorder(): BorderConfig {
    if (!this.showBorder) {
      return { width: 0, color: Color.Transparent };
    }
    return { width: 1, color: (this.cardBorderColor || this.getPalette().floatingBorderColor || this.getPalette().borderColor) as ResourceColor };
  }

  // 获取毛玻璃效果
  private getBlurStyle(): BlurStyle {
    if (this.variant === CardVariant.GLASS || this.useBlur) {
      return BlurStyle.Thin;
    }
    if (this.variant === CardVariant.GLOW) {
      return BlurStyle.Regular;
    }
    return BlurStyle.NONE;
  }

  private getCardRadius(): number {
    if (this.cardBorderRadius === DesignConstants.SHAPE.CARD) {
      return AppThemeVisuals.getSurfaceCardStyle(this.resolvedThemeId).borderRadius;
    }
    return this.cardBorderRadius;
  }

  private getCardPadding(): number {
    if (this.cardPadding === DesignConstants.SPACING.LG) {
      return AppThemeVisuals.getSurfaceCardStyle(this.resolvedThemeId).padding;
    }
    return this.cardPadding;
  }

  private getScale(): ScaleOptions {
    if (this.clickable && this.isPressed) {
      return { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD };
    }
    return { x: 1, y: 1 };
  }

  private shouldUseNeuShadow(): boolean {
    return this.isNeuTheme() && this.showShadow &&
      (this.variant === CardVariant.DEFAULT || this.variant === CardVariant.ELEVATED ||
        this.variant === CardVariant.OUTLINED);
  }

  build() {
    if (this.shouldUseNeuShadow()) {
      Stack() {
        Column()
          .width('100%')
          .backgroundColor(this.getBackgroundColor())
          .borderRadius(this.getCardRadius())
          .shadow(this.getNeuHighlightShadow())

        Column() {
          this.content();
        }
        .width('100%')
        .backgroundColor(this.getBackgroundColor())
        .borderRadius(this.getCardRadius())
        .border(this.getBorder())
        .padding(this.getCardPadding())
        .shadow(this.getNeuInsetShadow())
      }
      .width('100%')
      .alignContent(Alignment.TopStart)
      .scale(this.getScale())
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
          DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
        )
      })
      .onTouch((event: TouchEvent) => {
        if (this.clickable) {
          if (event.type === TouchType.Down) {
            this.isPressed = true;
          } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
            this.isPressed = false;
          }
        }
      })
      .accessibilityGroup(this.clickable)
      .accessibilityText(this.clickable ? '卡片' : '')
    } else {
      Column() {
        this.content();
      }
      .width('100%')
      .backgroundColor(this.getBackgroundColor())
      .backgroundBlurStyle(this.getBlurStyle())
      .borderRadius(this.getCardRadius())
      .border(this.getBorder())
      .padding(this.getCardPadding())
      .shadow(this.getShadow())
      .scale(this.getScale())
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
          DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
        )
      })
      .onTouch((event: TouchEvent) => {
        if (this.clickable) {
          if (event.type === TouchType.Down) {
            this.isPressed = true;
          } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
            this.isPressed = false;
          }
        }
      })
      .accessibilityGroup(this.clickable)
      .accessibilityText(this.clickable ? '卡片' : '')
    }
  }
}

/**
 * 瀑布流卡片组件（小红书风格）
 */
@Reusable
@Component
export struct MasonryCard {
  @BuilderParam content: () => void;
  @BuilderParam imageContent?: () => void;

  @Prop cardRadius: number = DesignConstants.SHAPE.CARD;
  @Prop shadowRadius: number = 6;
  @Prop shadowOpacity: number = 0.04;
  @Prop clickable: boolean = false;

  @State isPressed: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    if (this.resolvedThemeId.startsWith('figma_neu')) {
      Stack() {
        Column()
          .width('100%')
          .backgroundColor(this.getPalette().surfaceBackground)
          .borderRadius(this.cardRadius)
          .shadow(this.getPalette().visuals.effects.highlightShadow)
        Column() {
          if (this.imageContent) {
            this.imageContent!();
          }
          this.content();
        }
        .width('100%')
        .backgroundColor(this.getPalette().surfaceBackground)
        .borderRadius(this.cardRadius)
        .border({ width: 1, color: this.getPalette().borderColor })
        .shadow(this.getPalette().visuals.effects.insetShadow)
      }
      .width('100%')
      .alignContent(Alignment.TopStart)
      .scale(this.clickable && this.isPressed ?
        { x: DesignConstants.INTERACTION.SCALE_PRESSED_MASONRY, y: DesignConstants.INTERACTION.SCALE_PRESSED_MASONRY } :
        { x: 1, y: 1 })
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
          DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
        )
      })
      .onTouch((event: TouchEvent) => {
        if (this.clickable) {
          if (event.type === TouchType.Down) {
            this.isPressed = true;
          } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
            this.isPressed = false;
          }
        }
      })
      .accessibilityGroup(this.clickable)
      .accessibilityText(this.clickable ? '卡片' : '')
    } else {
      Column() {
        if (this.imageContent) {
          this.imageContent!();
        }
        this.content();
      }
      .width('100%')
      .backgroundColor(this.getPalette().surfaceBackground)
      .borderRadius(this.cardRadius)
      .shadow(this.getPalette().surfaceShadow)
      .border({ width: 1, color: this.getPalette().borderColor })
      .scale(this.clickable && this.isPressed ?
        { x: DesignConstants.INTERACTION.SCALE_PRESSED_MASONRY, y: DesignConstants.INTERACTION.SCALE_PRESSED_MASONRY } :
        { x: 1, y: 1 })
      .animation({
        curve: curves.springMotion(
          DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
          DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
        )
      })
      .onTouch((event: TouchEvent) => {
        if (this.clickable) {
          if (event.type === TouchType.Down) {
            this.isPressed = true;
          } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
            this.isPressed = false;
          }
        }
      })
      .accessibilityGroup(this.clickable)
      .accessibilityText(this.clickable ? '卡片' : '')
    }
  }
}
