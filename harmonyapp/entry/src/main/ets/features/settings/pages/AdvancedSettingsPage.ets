/**
 * 高级设置页面
 * 重构：使用原生 List 组件
 */

import { PageNames } from '../../../constants/AppConstants';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon, IconSize, IconButton } from '../../../components/SystemIcon';
import { SystemToast } from '../../../components/SystemToast';

@Entry
@Component
struct AdvancedSettingsPage {
  build() {
    Column() {
      // 标题栏
      Row() {
        IconButton({
          icon: AppIcon.BACK,
          iconSize: IconSize.LARGE,
          iconColor: DesignConstants.COLOR.TEXT_PRIMARY_LIGHT,
          bgColor: Color.Transparent,
          onButtonClick: () => {
            const routerInstance = this.getUIContext().getRouter();
            routerInstance.back();
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
      .padding({ left: 8, right: 8 })
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
            this.SettingItem('调试面板', async () => {
               const routerInstance = this.getUIContext().getRouter();
               await routerInstance.pushUrl({ url: `features/debug/pages/${PageNames.DEBUG_PANEL}` });
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
