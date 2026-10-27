/**
 * ViewedMomentThumb —— 「最近看过」动态缩略图（图/文兜底 + 按压反馈 + 点击进详情）。
 * 从 ProfilePage.buildViewedMomentThumb 抽取；按压态由组件内部自持。
 */

import { curves } from '@kit.ArkUI';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { ImageUrlUtil } from '../../../core/utils/ImageUrlUtil';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { MomentDetailRouteParam } from '../../../core/router/AppRouter';
import { MomentItem } from '../../../core/types/api';
import { cleanPlainText } from '../../../core/utils/FormatUtil';
import { LucideIcon } from '../../../components/LucideIcon';
import { AppIcon } from '../../../components/SystemIcon';

@Component
export struct ViewedMomentThumb {
  @Prop moment: MomentItem;
  palette: AppThemePalette | null = null;
  @State pressed: boolean = false;
  @State imageLoaded: boolean = false;
  @State imageFailed: boolean = false;

  build() {
    Stack() {
      if (this.moment.images && this.moment.images.length > 0) {
        // R8 占位层：图标常驻图片底层，加载中/失败不出现纯灰块
        Column() {
          LucideIcon({
            icon: AppIcon.IMAGES,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_MD,
            color: this.palette?.textTertiary
          })
        }
        .width('100%')
        .height('100%')
        .justifyContent(FlexAlign.Center)
        .borderRadius(DesignConstants.RADIUS.LG)

        if (!this.imageFailed) {
          Image(ImageUrlUtil.optimized(MediaUrlUtil.normalize(this.moment.images[0].url), 320))
            .width('100%')
            .height('100%')
            .objectFit(ImageFit.Cover)
            .borderRadius(DesignConstants.RADIUS.LG)
            .opacity(this.imageLoaded || MotionUtil.reduced() ? 1 : 0)
            .animation({
              duration: MotionUtil.reduced() ? 0 : DesignConstants.ANIMATION.DURATION_SHORT,
              curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
            })
            .onComplete(() => { this.imageLoaded = true; })
            .onError(() => { this.imageFailed = true; })
        }
      } else {
        Column() {
          Text(cleanPlainText(this.moment.description))
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.palette?.textSecondary)
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .textAlign(TextAlign.Center)
        }
        .width('100%')
        .height('100%')
        .padding(DesignConstants.SPACING.XS)
        .justifyContent(FlexAlign.Center)
        .backgroundColor(this.palette?.surfaceSubtle)
        .borderRadius(DesignConstants.RADIUS.LG)
      }
    }
    .aspectRatio(1)
    .width('100%')
    .backgroundColor(this.palette?.surfaceSubtle)
    .borderRadius(DesignConstants.RADIUS.LG)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.palette?.borderColor })
    // 缩略图按压反馈（对齐 tap-97 语义）
    .scale(this.pressed ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressed = false;
      }
    })
    .onClick(() => {
      if (this.moment.moment_id) {
        const params = new MomentDetailRouteParam();
        params.momentId = this.moment.moment_id;
        AppRouter.push(PageNames.MOMENT_DETAIL, params);
      }
    })
  }
}
