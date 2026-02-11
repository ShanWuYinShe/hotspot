/**
 * LoginPromptView - 可复用的登录提示组件
 * 用于需要登录才能访问的页面/标签，提供统一的未登录状态展示
 */

import { DesignConstants } from '../constants/DesignConstants';
import { PageNames } from '../constants/AppConstants';
import { AppRouter } from '../core/router/AppRouter';
import { AppIcon } from './SystemIcon';

@Component
export struct LoginPromptView {
  /** 主标题，如 "附近动态需要登录" */
  @Prop title: string = '此功能需要登录';
  /** 副标题说明 */
  @Prop subtitle: string = '登录后即可查看完整内容';
  /** 底部提示文字 */
  @Prop hint: string = '';
  /** 图标（默认锁图标） */
  @Prop icon: AppIcon = AppIcon.PERSON;

  build() {
    Column({ space: 20 }) {
      // 图标
      Text() {
        SymbolSpan($r(this.icon))
          .fontSize(56)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }
      .margin({ bottom: 4 })

      // 主标题
      Text(this.title)
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(FontWeight.Bold)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .textAlign(TextAlign.Center)

      // 副标题
      Text(this.subtitle)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
        .lineHeight(22)

      // 按钮区
      Column({ space: 12 }) {
        Button('快速体验（无需注册）')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .width('75%')
          .height(44)
          .onClick(() => {
            this.navigateToLogin();
          })

        Button('登录 / 注册')
          .type(ButtonType.Capsule)
          .backgroundColor($r('sys.color.ohos_id_color_component_normal'))
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .width('75%')
          .height(44)
          .onClick(() => {
            this.navigateToLogin();
          })
      }
      .margin({ top: 8 })

      // 底部提示
      if (this.hint.length > 0) {
        Text(this.hint)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor($r('sys.color.ohos_id_color_text_secondary'))
          .textAlign(TextAlign.Center)
          .margin({ top: 4 })
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .padding(32)
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
  }

  private navigateToLogin(): void {
    AppRouter.push(PageNames.LOGIN);
  }
}
