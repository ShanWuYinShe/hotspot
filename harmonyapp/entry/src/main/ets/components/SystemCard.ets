import { DesignConstants } from '../constants/DesignConstants';

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
  @Prop cardBorderRadius: number = DesignConstants.RADIUS.XXL;
  // 新增：是否显示边框
  @Prop showBorder: boolean = true;
  // 新增：边框颜色（使用系统颜色）
  @Prop cardBorderColor: ResourceColor = $r('sys.color.ohos_id_color_list_separator');

  @State isPressed: boolean = false;

  private getBackgroundColor(): ResourceColor {
    if (this.cardBackgroundColor) {
      return this.cardBackgroundColor as ResourceColor;
    }

    switch (this.variant) {
      case CardVariant.GLASS:
        return Color.Transparent;
      case CardVariant.FILLED:
      case CardVariant.ELEVATED:
      case CardVariant.OUTLINED:
      case CardVariant.DEFAULT:
      default:
        return $r('app.color.bg_secondary');
    }
  }

  private getShadow(): ShadowConfig {
    switch (this.variant) {
      case CardVariant.ELEVATED:
        return DesignConstants.SHADOW.LARGE;
      case CardVariant.GLASS:
        return this.showShadow ? DesignConstants.SHADOW.LARGE : { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 };
      case CardVariant.DEFAULT:
      case CardVariant.FILLED:
      case CardVariant.OUTLINED:
      default:
        return DesignConstants.SHADOW.MEDIUM;
    }
  }

  private getBorder(): BorderConfig {
    if (!this.showBorder) {
      return { width: 0, color: Color.Transparent };
    }
    return { width: 1, color: this.cardBorderColor };
  }

  // 获取毛玻璃效果
  private getBlurStyle(): BlurStyle {
    if (this.variant === CardVariant.GLASS || this.useBlur) {
      return BlurStyle.Thin;
    }
    return BlurStyle.NONE;
  }

  // 获取点击缩放比例
  private getScale(): ScaleOptions {
    if (this.clickable && this.isPressed) {
      return { x: 0.97, y: 0.97 };
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
    .borderRadius(this.cardBorderRadius)
    .border(this.getBorder())
    .padding(this.cardPadding)
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

  build() {
    Column() {
      if (this.imageContent) {
        this.imageContent!();
      }
      this.content();
    }
    .width('100%')
    .backgroundColor($r('app.color.bg_secondary'))
    .borderRadius(this.cardRadius)
    .shadow(DesignConstants.SHADOW.MEDIUM)
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
