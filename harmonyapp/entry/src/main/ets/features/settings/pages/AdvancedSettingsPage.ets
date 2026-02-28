/**
 * 高级设置页面
 * 重构：使用原生 List 组件
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemToast } from '../../../components/SystemToast';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';

@Preview
@Component
export struct AdvancedSettingsPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;

  aboutToAppear(): void {
    SystemUiController.useSurface({
      baseColor: DesignConstants.COLOR.PRIMARY_BG_LIGHT,      preferDarkContent: true
    });
  }

  build() {
    NavDestination() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: $r('sys.color.ohos_id_color_text_primary'),
          bgColor: Color.Transparent,
          onButtonClick: () => {
            AppRouter.pop();
          }
        })

        Text('高级设置')
          .fontSize(DesignConstants.FONT.SIZE.XL)
          .fontWeight(FontWeight.Medium)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop),
        bottom: DesignConstants.SPACING.SM
      })
      .backgroundColor($r('app.color.bg_primary'))
      .expandSafeArea([SafeAreaType.SYSTEM], [SafeAreaEdge.TOP])

      Column() {
        Column() {
          this.SettingItem(AppIcon.CLEAN, '清理缓存', '清除本地缓存数据', () => {
            SystemToast.show('缓存已清理', this.getUIContext());
          })

          Divider()
            .strokeWidth(0.5)
            .color($r('sys.color.ohos_id_color_list_separator'))
            .margin({ left: 20, right: 20 })

          this.SettingItem(AppIcon.DEBUG, '调试面板', '网络诊断、状态检查', () => {
            AppRouter.push(PageNames.DEBUG_PANEL);
          })
        }
        .backgroundColor($r('app.color.bg_secondary'))
        .borderRadius(DesignConstants.RADIUS.XXL)
        .border({ width: 1, color: $r('sys.color.ohos_id_color_list_separator') })
        .shadow({ radius: 10, color: $r('app.color.color_shadow'), offsetX: 0, offsetY: 3 })
        .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
          top: DesignConstants.SPACING.LG })
      }
      .width('100%')
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)
      .backgroundColor($r('app.color.bg_primary'))
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('app.color.bg_primary'))
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder
  SettingItem(icon: AppIcon, title: string, subtitle: string, onClick: () => void) {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        Text() {
          SymbolSpan($r(icon))
            .fontSize(18)
            .fontColor([$r('sys.color.ohos_id_color_emphasize')])
        }
      }
      .width(38)
      .height(38)
      .borderRadius(DesignConstants.RADIUS.XL)
      .backgroundColor($r('app.color.bg_tertiary'))
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Column({ space: 4 }) {
        Text(title)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontColor($r('sys.color.ohos_id_color_text_primary'))

        if (subtitle) {
          Text(subtitle)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
        }
      }
      .layoutWeight(1)
      .alignItems(HorizontalAlign.Start)

      Row() {
        Text() {
          SymbolSpan($r(AppIcon.ARROW_RIGHT))
            .fontSize(15)
            .fontColor([$r('sys.color.ohos_id_color_text_secondary')])
        }
      }
      .width(28)
      .height(28)
    .borderRadius(DesignConstants.RADIUS.XL)
      .backgroundColor($r('app.color.bg_tertiary'))
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .borderRadius(DesignConstants.RADIUS.XL)
    .onClick(onClick)
  }
}

@Builder
export function AdvancedSettingsPageBuilder() {
  AdvancedSettingsPage()
}
