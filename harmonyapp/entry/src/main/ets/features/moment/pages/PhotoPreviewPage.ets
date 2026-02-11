/**
 * 图片预览页面
 * 支持手势缩放、滑动切换
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconButton, IconSize } from '../../../components/SystemIcon';
import { AppRouter } from '../../../core/router/AppRouter';

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

  build() {
    NavDestination() {
    Stack() {
      // 黑色背景
      Column()
        .width('100%')
        .height('100%')
        .backgroundColor(Color.Black)

      // 图片轮播
      Swiper(this.swiperController) {
        ForEach(this.images, (imageUrl: string, index: number) => {
          Image(imageUrl)
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
                      this.getUIContext().animateTo({ duration: 300, curve: Curve.FastOutSlowIn }, () => {
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
                        this.getUIContext().animateTo({ duration: 300, curve: Curve.FastOutSlowIn }, () => {
                          this.offsetX = targetX;
                          this.offsetY = targetY;
                        })
                      }
                    }
                  }),
                TapGesture({ count: 2 })
                  .onAction(() => {
                    this.getUIContext().animateTo({ duration: 300, curve: Curve.FastOutSlowIn }, () => {
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

      // 顶部关闭按钮
      Row() {
        IconButton({
          icon: AppIcon.CLOSE,
          iconSize: IconSize.LARGE,
          iconColor: Color.White,
          bgColor: 'rgba(0,0,0,0.3)',
          onButtonClick: () => {
            AppRouter.pop();
          }
        })
        .margin({ left: 16, top: 16 })
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)
      .position({ x: 0, y: 0 })
      .hitTestBehavior(HitTestMode.Transparent) // 允许点击穿透到空白处（如果有）

      // 底部页码指示器
      Row() {
        Text(`${this.currentIndex + 1} / ${this.images.length}`)
          .fontColor(Color.White)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(FontWeight.Medium)
          .backgroundColor('rgba(0,0,0,0.3)')
          .padding({ left: 12, right: 12, top: 6, bottom: 6 })
          .borderRadius(16)
      }
      .width('100%')
      .justifyContent(FlexAlign.Center)
      .position({ x: 0, y: '90%' })
    }
    .width('100%')
    .height('100%')
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

