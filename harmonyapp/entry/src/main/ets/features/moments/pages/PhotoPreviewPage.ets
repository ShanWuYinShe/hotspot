/**
 * 图片预览页面
 * 支持手势缩放、滑动切换
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { AppRouter } from '../../../core/router/AppRouter';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';

@Preview
@Component
export struct PhotoPreviewPage {
  pathStack: NavPathStack = new NavPathStack();
  @State images: string[] = [];
  @State currentIndex: number = 0;
  @State momentId: string = '';
  private swiperController: SwiperController = new SwiperController();

  @State scaleValue: number = 1;
  @State pinchValue: number = 1;
  @State offsetX: number = 0;
  @State offsetY: number = 0;
  @State positionX: number = 0;
  @State positionY: number = 0;
  @State compWidth: number = 0;
  @State compHeight: number = 0;

  private getSurfaceBorder(): string {
    return DesignConstants.COLOR.OVERLAY_LIGHT;
  }

  aboutToAppear(): void {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.BLACK,
      overlayOpacity: 1,
      preferDarkContent: false
    });
  }

  build() {
    NavDestination() {
    Stack() {
      // 黑色背景
      Column()
        .width('100%')
        .height('100%')
        .backgroundColor(DesignConstants.COLOR.BLACK)
        .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP, SafeAreaEdge.BOTTOM])

      // 图片轮播
      Swiper(this.swiperController) {
        ForEach(this.images, (imageUrl: string, index: number) => {
          Image(MediaUrlUtil.normalize(imageUrl))
            .width('100%')
            .height('100%')
            .objectFit(ImageFit.Contain)
            .geometryTransition(this.momentId ? 'moment_img_' + this.momentId + '_' + index : '')
            .scale({ x: this.currentIndex === index ? this.scaleValue : 1, y: this.currentIndex === index ? this.scaleValue : 1, z: 1 })
            .translate({ x: this.currentIndex === index ? this.offsetX : 0, y: this.currentIndex === index ? this.offsetY : 0 })
            .onAreaChange((oldValue: Area, newValue: Area) => {
              this.compWidth = newValue.width as number;
              this.compHeight = newValue.height as number;
            })
            .gesture(
              GestureGroup(GestureMode.Exclusive,
                PinchGesture({ fingers: 2 })
                  .onActionStart(() => {
                    this.pinchValue = this.scaleValue;
                  })
                  .onActionUpdate((event: GestureEvent) => {
                    if (event) {
                      this.scaleValue = this.pinchValue * event.scale;
                      // 限制缩放范围
                      if (this.scaleValue < 0.5) this.scaleValue = 0.5;
                      if (this.scaleValue > 3.0) this.scaleValue = 3.0;
                    }
                  })
                  .onActionEnd(() => {
                    // 回弹效果：如果小于1则恢复到1
                    if (this.scaleValue < 1) {
                      this.getUIContext().animateTo({ duration: DesignConstants.ANIMATION.DURATION_MEDIUM, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }, () => {
                        this.scaleValue = 1;
                        this.offsetX = 0;
                        this.offsetY = 0;
                      })
                    }
                  }),
                PanGesture({ fingers: 1 })
                  .onActionStart(() => {
                    this.positionX = this.offsetX;
                    this.positionY = this.offsetY;
                  })
                  .onActionUpdate((event: GestureEvent) => {
                    if (this.scaleValue > 1) {
                      this.offsetX = this.positionX + event.offsetX;
                      this.offsetY = this.positionY + event.offsetY;
                    }
                  })
                  .onActionEnd(() => {
                    // 边界回弹
                    if (this.scaleValue > 1 && this.compWidth > 0 && this.compHeight > 0) {
                      const maxOffsetX = (this.scaleValue - 1) * this.compWidth / 2;
                      const maxOffsetY = (this.scaleValue - 1) * this.compHeight / 2;
                      
                      let targetX = this.offsetX;
                      let targetY = this.offsetY;
                      let needBounce = false;
                      
                      if (this.offsetX > maxOffsetX) { 
                        targetX = maxOffsetX; 
                        needBounce = true; 
                      } else if (this.offsetX < -maxOffsetX) { 
                        targetX = -maxOffsetX; 
                        needBounce = true; 
                      }
                      
                      if (this.offsetY > maxOffsetY) { 
                        targetY = maxOffsetY; 
                        needBounce = true; 
                      } else if (this.offsetY < -maxOffsetY) { 
                        targetY = -maxOffsetY; 
                        needBounce = true; 
                      }
                      
                      if (needBounce) {
                        this.getUIContext().animateTo({ duration: DesignConstants.ANIMATION.DURATION_MEDIUM, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }, () => {
                          this.offsetX = targetX;
                          this.offsetY = targetY;
                        })
                      }
                    }
                  }),
                TapGesture({ count: 2 })
                  .onAction(() => {
                    this.getUIContext().animateTo({ duration: DesignConstants.ANIMATION.DURATION_MEDIUM, curve: DesignConstants.ANIMATION.CURVE_FAST_OUT_SLOW_IN }, () => {
                      if (this.scaleValue > 1) {
                        this.scaleValue = 1;
                        this.offsetX = 0;
                        this.offsetY = 0;
                      } else {
                        this.scaleValue = 2;
                      }
                    })
                  })
              )
            )
        }, (item: string) => item)
      }
      .index(this.currentIndex)
      .width('100%')
      .height('100%')
      .indicator(false)
      .loop(false)
      .onChange((index: number) => {
        this.currentIndex = index;
        // 切换页面时重置缩放
        this.scaleValue = 1;
        this.offsetX = 0;
        this.offsetY = 0;
      })

      // 顶部关闭按钮：媒体镀铬白字（iconColor 逃逸口），透明底浮于照片上
      Row() {
        SurfaceIconButton({
          icon: AppIcon.CLOSE,
          kind: 'neutral',
          buttonSize: DesignConstants.LAYOUT.BACK_BUTTON_SIZE,
          glyphSize: DesignConstants.FONT.SIZE.LG_PLUS,
          iconColor: DesignConstants.COLOR.MEDIA_CHROME_TEXT,
          accessibilityLabel: '关闭预览',
          onPress: () => {
            AppRouter.pop();
          }
        })
        .margin({ left: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.LG })
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)
      .position({ x: 0, y: 0 })
      .hitTestBehavior(HitTestMode.Transparent) // 允许点击穿透到空白处（如果有）

      // 底部页码指示器（凸起胶囊化）
      Row() {
        Text(`${this.currentIndex + 1} / ${this.images.length}`)
          .fontColor(DesignConstants.COLOR.MEDIA_CHROME_TEXT)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .backgroundColor(DesignConstants.COLOR.OVERLAY_STRONG)
          .padding({
            left: DesignConstants.SPACING.LG,
            right: DesignConstants.SPACING.LG,
            top: DesignConstants.SPACING.SM,
            bottom: DesignConstants.SPACING.SM
          })
          .borderRadius(DesignConstants.RADIUS.FULL)
          .shadow(DesignConstants.SHADOW.SMALL) // 扁平设计无投影（对齐 SHADOW 全零契约）
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
      }
      .width('100%')
      .justifyContent(FlexAlign.Center)
      .position({ x: 0, y: '90%' })
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.BLACK)
    .onClick(() => {
      // 单击空白/图片退出预览（业内惯例；双击缩放由 PinchGesture/Pinch 识别器消费，不冲突）
      this.pathStack?.pop();
    })
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as Record<string, Object> | undefined;
      if (params) {
        this.images = (params['images'] as string[]) || [];
        this.currentIndex = (params['index'] as number) ?? 0;
        this.momentId = (params['momentId'] as string) || '';
      }
    })
  }
}

@Builder
export function PhotoPreviewPageBuilder() {
  PhotoPreviewPage()
}