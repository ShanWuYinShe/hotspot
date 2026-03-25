import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../core/theme/AppThemeVisuals';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

/**
 * 卡片样式变体
 */
export enum CardVariant {
  DEFAULT = 'default',      // 默认样式
  ELEVATED = 'elevated',    // 悬浮样式（更强阴影）
  OUTLINED = 'outlined',    // 边框样式
  FILLED = 'filled',        // 填充样式
  GLASS = 'glass',          // 毛玻璃样式
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

  private getThemeShadow(kind: 'default' | 'elevated'): ShadowConfig {
    const palette = this.getPalette();
    return kind === 'elevated' ? palette.elevatedShadow : palette.surfaceShadow;
  }

  private getBackgroundColor(): ResourceColor {
    if (this.cardBackgroundColor) {
      return this.cardBackgroundColor as ResourceColor;
    }

    switch (this.variant) {
      case CardVariant.GLASS:
        return Color.Transparent;
      case CardVariant.FILLED:
        return this.getPalette().buttonSecondaryBackground;
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
    return { width: 1, color: (this.cardBorderColor || this.getPalette().borderColor) as ResourceColor };
  }

  // 获取毛玻璃效果
  private getBlurStyle(): BlurStyle {
    if (this.variant === CardVariant.GLASS || this.useBlur) {
      return BlurStyle.Thin;
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

  // 获取点击缩放比例
  private getScale(): ScaleOptions {
    if (this.clickable && this.isPressed) {
      return { x: 0.985, y: 0.985 };
    }
    return { x: 1, y: 1 };
  }

  build() {
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
      duration: DesignConstants.ANIMATION.DURATION_SHORT,
      curve: Curve.EaseOut
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

  @Prop cardRadius: number = DesignConstants.RADIUS.XL;
  @Prop shadowRadius: number = 6;
  @Prop shadowOpacity: number = 0.04;
  @Prop clickable: boolean = false;

  @State isPressed: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
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
    .scale(this.clickable && this.isPressed ? { x: 0.97, y: 0.97 } : { x: 1, y: 1 })
    .animation({ duration: 100, curve: Curve.EaseOut })
    .onTouch((event: TouchEvent) => {
      if (this.clickable) {
        if (event.type === TouchType.Down) {
          this.isPressed = true;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.isPressed = false;
        }
      }
    })
  }
}
