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
  @Prop cardBorderRadius: number = DesignConstants.RADIUS.XL;
  // 新增：是否显示边框
  @Prop showBorder: boolean = false;
  // 新增：边框颜色（使用系统颜色）
  @Prop cardBorderColor: ResourceColor = $r('sys.color.ohos_id_color_component_normal');

  @State isPressed: boolean = false;

  // 根据变体获取背景色
  private getBackgroundColor(): ResourceColor {
    if (this.cardBackgroundColor) {
      return this.cardBackgroundColor as ResourceColor;
    }

    switch (this.variant) {
      case CardVariant.FILLED:
        return $r('sys.color.ohos_id_color_component_normal');
      case CardVariant.GLASS:
        return Color.Transparent;
      case CardVariant.ELEVATED:
      case CardVariant.OUTLINED:
      case CardVariant.DEFAULT:
      default:
        return $r('app.color.bg_secondary');
    }
  }

  // 根据变体获取阴影
  private getShadow(): ShadowConfig {
    if (!this.showShadow) {
      return { radius: 0, color: '#00000000', offsetX: 0, offsetY: 0 };
    }

    switch (this.variant) {
      case CardVariant.ELEVATED:
        return {
          radius: 12,
          color: '#1A000000',
          offsetX: 0,
          offsetY: 4
        };
      case CardVariant.GLASS:
        return {
          radius: 16,
          color: '#26000000',
          offsetX: 0,
          offsetY: 8
        };
      case CardVariant.DEFAULT:
      case CardVariant.FILLED:
      case CardVariant.OUTLINED:
      default:
        return {
          radius: 8,
          color: '#1A000000',
          offsetX: 0,
          offsetY: 2
        };
    }
  }

  // 根据变体获取边框
  private getBorder(): BorderConfig {
    if (this.variant === CardVariant.OUTLINED || this.showBorder) {
      return {
        width: 1,
        color: this.cardBorderColor
      };
    }
    return { width: 0, color: $r('app.color.bg_secondary') };
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
    .borderWidth(this.getBorder().width)
    .borderColor(this.getBorder().color)
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

  @Prop cardRadius: number = 12;
  @Prop shadowRadius: number = 8;
  @Prop shadowOpacity: number = 0.05;
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
    .shadow({
      radius: this.shadowRadius,
      color: `rgba(0,0,0,${this.shadowOpacity})`,
      offsetX: 0,
      offsetY: 2
    })
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
