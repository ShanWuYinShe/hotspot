import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { SystemButton, ButtonVariant } from './SystemButton';

/**
 * 通用确认对话框（对齐前端 confirm-dialog）
 * 条件渲染模态：遮罩层 + 居中卡片 + 标题/描述 + 取消/确认按钮。
 *
 * 用法：父组件维护 @State visible，渲染 <SystemDialog visible={...} onConfirm onCancel />，
 * 在 onCancel / onConfirm 回调里把 visible 置 false。
 * 确认按钮的 confirmVariant 可传 'danger' 用于删除确认。
 */
@Component
export struct SystemDialog {
  @Prop visible: boolean = false;
  @Prop title: string = '';
  @Prop description: string = '';
  @Prop confirmText: string = '确认';
  @Prop cancelText: string = '取消';
  @Prop confirmVariant: ButtonVariant = 'primary';
  @Prop showCancel: boolean = true;
  /** 点击遮罩是否触发 onCancel（调用方需在 onCancel 里关闭） */
  @Prop closeOnMaskClick: boolean = true;
  onCancel?: () => void;
  onConfirm?: () => void;

  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    if (this.visible) {
      Stack() {
        // 遮罩层
        Column()
          .width('100%')
          .height('100%')
          .backgroundColor(this.getPalette().overlayStrong)
          .onClick(() => {
            if (this.closeOnMaskClick) {
              this.onCancel?.();
            }
          })

        // 居中卡片
        Column() {
          if (this.title.length > 0) {
            Text(this.title)
              .fontSize(DesignConstants.FONT.SIZE.LG_PLUS) // 18
              .fontWeight(FontWeight.Semibold)
              .fontColor(this.getPalette().textPrimary)
              .textAlign(TextAlign.Center)
              .width('100%')
          }
          if (this.description.length > 0) {
            Text(this.description)
              .fontSize(DesignConstants.FONT.SIZE.SM_PLUS) // 13
              .fontColor(this.getPalette().textSecondary)
              .textAlign(TextAlign.Center)
              .width('100%')
              .margin({ top: DesignConstants.SPACING.SM })
              .lineHeight(20)
          }

          Row() {
            if (this.showCancel) {
              Column() {
                SystemButton({
                  label: this.cancelText,
                  variant: 'secondary',
                  fullWidth: true,
                  onPress: () => {
                    this.onCancel?.();
                  }
                })
              }
              .layoutWeight(1)
            }
            Column() {
              SystemButton({
                label: this.confirmText,
                variant: this.confirmVariant,
                fullWidth: true,
                onPress: () => {
                  this.onConfirm?.();
                }
              })
            }
            .layoutWeight(1)
          }
          .width('100%')
          .margin({ top: DesignConstants.SPACING.XL })
        }
        .width('80%')
        .padding(DesignConstants.SPACING.XL)
        .backgroundColor(this.getPalette().surfaceBackground)
        .borderRadius(DesignConstants.SHAPE.CARD) // 22
        .alignItems(HorizontalAlign.Center)
      }
      .width('100%')
      .height('100%')
      .alignContent(Alignment.Center)
    }
  }
}
