import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../core/theme/AppThemeVisuals';
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
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

  @Prop cardBackgroundColor: ResourceColor | string = '';
  @Prop useBlur: boolean = false;

  // 新增：卡片样式变体
  @Prop variant: CardVariant = CardVariant.DEFAULT;
  // 新增：圆角大小（重命名为 cardBorderRadius 避免冲突）
  @Prop cardBorderRadius: number = DesignConstants.SHAPE.CARD;
  // 是否显示边框（默认关闭：卡片层级靠 surface 分层与阴影表达，1vp 描边满屏
  // 叠加会让移动端呈现「框框」碎片感；需要描边的调用方显式传 true 或用 OUTLINED 变体）
  @Prop showBorder: boolean = false;
  // 新增：边框颜色（使用系统颜色）
  @Prop cardBorderColor: ResourceColor | string = '';

  @State isPressed: boolean = false;
  /** hover（仅键鼠外接触发）：enableHoverEffect 打开时边框提亮 */
  @State isHovered: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

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
        return DesignConstants.COLOR.TRANSPARENT;
      case CardVariant.FILLED:
        return this.getPalette().surfaceSubtle;
      case CardVariant.ELEVATED:
      case CardVariant.OUTLINED:
      case CardVariant.DEFAULT:
      default:
        // r3 软岛：默认卡改用 85% surface（镜像 web bg-surface-primary/85）
        return this.getPalette().surfaceIsland;
    }
  }

  private getShadow(): ShadowConfig {
    if (!this.showShadow) {
      return { radius: 0, color: DesignConstants.COLOR.TRANSPARENT, offsetX: 0, offsetY: 0 };
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
    // OUTLINED 变体的语义就是描边样式，无视 showBorder 默认值
    if (!this.showBorder && this.variant !== CardVariant.OUTLINED) {
      return { width: 0, color: DesignConstants.COLOR.TRANSPARENT };
    }
    const baseColor = (this.cardBorderColor || this.getPalette().floatingBorderColor ||
      this.getPalette().borderColor) as ResourceColor;
    // hover（键鼠）+ enableHoverEffect：边框提亮为 accent（阴影全零的扁平契约下，
    // 边框是 hover 的唯一可见反馈；对齐前端 surface-flat:hover 描边加深）
    if (this.enableHoverEffect && this.isHovered && this.clickable) {
      return { width: DesignConstants.BORDER_WIDTH.REGULAR, color: this.getPalette().accentBackground };
    }
    return { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: baseColor };
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

  private getScale(): ScaleOptions {
    if (this.clickable && this.isPressed) {
      return { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD };
    }
    return { x: 1, y: 1 };
  }

  build() {
    // 已删除主题族收敛：全主题统一走单栏普通卡片路径（vibrant/forest 同一样式）
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
      .onHover((isHover: boolean) => {
        if (this.clickable) {
          this.isHovered = isHover;
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

  @Prop cardRadius: number = DesignConstants.SHAPE.CARD;
  @Prop clickable: boolean = false;

  @State isPressed: boolean = false;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    // 已删除主题族收敛：瀑布流卡统一走普通单栏路径（vibrant/forest 同一样式，不自创第三种）
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
