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
          .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
          .layoutWeight(1)
          .textAlign(TextAlign.Center)

        Blank().width(48)
      }
      .width('100%')
      .height(56)
      .padding({ left: 8, right: 8 })
      .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

      List({ space: 16 }) {
        ListItemGroup() {
          ListItem() {
            this.SettingItem('清理缓存', () => {
              SystemToast.show('缓存已清理', this.getUIContext());
            })
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)

          ListItem() {
            this.SettingItem('调试面板', async () => {
               const routerInstance = this.getUIContext().getRouter();
               await routerInstance.pushUrl({ url: `features/debug/pages/${PageNames.DEBUG_PANEL}` });
            })
          }
          .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
        }
        .borderRadius(DesignConstants.RADIUS.LG)
        .divider({ strokeWidth: 1, color: DesignConstants.COLOR.TERTIARY_BG_LIGHT, startMargin: 16, endMargin: 16 })
        .margin({ left: 16, right: 16, top: 16 })
      }
      .width('100%')
      .layoutWeight(1)
      .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
    }
    .width('100%')
    .height('100%')
    .backgroundColor(DesignConstants.COLOR.SECONDARY_BG_LIGHT)
  }

  @Builder
  SettingItem(title: string, onClick: () => void) {
    Row() {
      Text(title)
        .fontSize(16)
        .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
        .layoutWeight(1)
      
      Text('>')
        .fontSize(16)
        .fontColor(DesignConstants.COLOR.TEXT_TERTIARY_LIGHT)
    }
    .width('100%')
    .padding(16)
    .onClick(onClick)
  }
}
