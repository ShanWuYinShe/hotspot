import { DesignConstants } from '../constants/DesignConstants';

@Component
export struct SystemCard {
  @BuilderParam content: () => void;
  @Prop cardPadding: number = DesignConstants.SPACING.LG;
  @Prop borderRadius: number = DesignConstants.RADIUS.XL;
  @Prop showShadow: boolean = true;
  @Prop clickable: boolean = false;
  @Prop hoverEffect: boolean = false; // New prop for hover effect
  
  // Backwards compatibility props mapping
  @Prop paddingSize: number | string = DesignConstants.SPACING.LG; 
  @Prop cardBackgroundColor: ResourceColor | string = DesignConstants.COLOR.SECONDARY_BG_LIGHT;
  @Prop useBlur: boolean = false;

  @State isPressed: boolean = false;

  build() {
    Column() {
      // Content slot
      this.content();
    }
    .width('100%')
    .backgroundColor(this.useBlur ? Color.Transparent : this.cardBackgroundColor)
    .backgroundBlurStyle(this.useBlur ? BlurStyle.Thin : BlurStyle.NONE)
    .borderRadius(this.borderRadius)
    .padding(this.cardPadding)
    .shadow(this.showShadow ? DesignConstants.SHADOW.SMALL : undefined)
    .scale(this.clickable && this.isPressed ? { x: 0.98, y: 0.98 } as Object as { x: number, y: number } : { x: 1, y: 1 } as Object as { x: number, y: number }) // Press down effect
    .animation({ duration: DesignConstants.ANIMATION.DURATION_SHORT, curve: DesignConstants.ANIMATION.CURVE_EASE_IN_OUT } as Object as { duration: number, curve: Curve })
    .onTouch((event: TouchEvent) => {
      if (this.clickable) {
        if (event.type === TouchType.Down) {
          this.isPressed = true;
        } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
          this.isPressed = false;
        }
      }
    })
    .hoverEffect(this.hoverEffect ? HoverEffect.Scale : HoverEffect.None)
  }
}
