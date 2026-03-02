/**
 * 通用空状态组件
 * 用于列表无数据、加载错误等场景的统一展示
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppIcon } from './SystemIcon';

@Reusable
@Component
export struct EmptyState {
  @Prop icon: AppIcon = AppIcon.EMPTY;
  @Prop message: string = '';
  @Prop actionLabel: string = '';
  onAction?: () => void;

  build() {
    Column({ space: DesignConstants.SPACING.LG }) {
      Text() {
        SymbolSpan($r(this.icon))
          .fontSize(48)
          .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
      }

      Text(this.message ?? '')
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        .textAlign(TextAlign.Center)
        .lineHeight(24)

      if ((this.actionLabel ?? '').length > 0 && this.onAction) {
        Button(this.actionLabel ?? '')
          .type(ButtonType.Normal)
          .stateEffect(true)
          .backgroundColor($r('sys.color.ohos_id_color_emphasize'))
          .fontColor(Color.White)
          .height(DesignConstants.COMPONENT.BUTTON_HEIGHT)
          .borderRadius(DesignConstants.RADIUS.XL)
          .onClick(() => {
            this.onAction?.();
          })
      }
    }
    .width('100%')
    .height('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.XXL)
  }
}

/**
 * 加载更多指示器
 * 用于列表底部的加载更多状态展示
 */
@Reusable
@Component
export struct LoadingMoreIndicator {
  @Prop message: string = '加载更多...';

  build() {
    Row({ space: DesignConstants.SPACING.SM }) {
      LoadingProgress().width(20).height(20)
      Text(this.message)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .width('100%')
    .justifyContent(FlexAlign.Center)
    .padding(DesignConstants.SPACING.MD)
  }
}

/**
 * 下拉刷新指示器
 * 用于 Refresh 组件的 builder 参数
 */
@Builder
export function RefreshIndicator() {
  Row() {
    LoadingProgress().width(24).height(24)
    Text('正在刷新...')
      .fontSize(DesignConstants.FONT.SIZE.MD)
      .margin({ left: DesignConstants.SPACING.SM })
      .fontColor($r('sys.color.ohos_id_color_text_secondary'))
  }
  .width('100%')
  .justifyContent(FlexAlign.Center)
  .padding(DesignConstants.SPACING.SM)
}
