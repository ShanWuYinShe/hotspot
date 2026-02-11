/**
 * 高级设置页面
 * 重构：使用原生 List 组件
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemToast } from '../../../components/SystemToast';
import { AppRouter } from '../../../core/router/AppRouter';

@Component
export struct AdvancedSettingsPage {
  pathStack: NavPathStack = new NavPathStack();

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
      .height(56)
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
      .backgroundColor($r('sys.color.ohos_id_color_background'))

      List({ space: 16 }) {
        ListItemGroup() {
          ListItem() {
            this.SettingItem('清理缓存', () => {
              SystemToast.show('缓存已清理', this.getUIContext());
            })
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))

          ListItem() {
            this.SettingItem('调试面板', () => {
              AppRouter.push('DebugPanel');
            })
          }
          .backgroundColor($r('sys.color.ohos_id_color_background'))
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .divider({ strokeWidth: 1, color: $r('sys.color.ohos_id_color_component_normal'), startMargin: 16, endMargin: 16 })
        .margin({ left: 16, right: 16, top: 16 })
      }
      .width('100%')
      .layoutWeight(1)
      .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    }
    .width('100%')
    .height('100%')
    .backgroundColor($r('sys.color.ohos_id_color_sub_background'))
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
    })
  }

  @Builder
  SettingItem(title: string, onClick: () => void) {
    Row() {
      Text(title)
        .fontSize(16)
        .fontColor($r('sys.color.ohos_id_color_text_primary'))
        .layoutWeight(1)
      
      Text('>')
        .fontSize(16)
        .fontColor($r('sys.color.ohos_id_color_text_secondary'))
    }
    .width('100%')
    .padding(16)
    .onClick(onClick)
  }
}

@Builder
export function AdvancedSettingsPageBuilder() {
  AdvancedSettingsPage()
}
