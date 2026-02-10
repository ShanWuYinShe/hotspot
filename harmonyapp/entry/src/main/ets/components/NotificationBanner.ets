import { SmartNotification, NotificationAction } from '../core/services/notification/NotificationDataManager';
import { DesignConstants } from '../constants/DesignConstants';
import { SystemIcon, AppIcon } from './SystemIcon';

@Component
export struct NotificationBanner {
  @ObjectLink notification: SmartNotification;
  onClose?: () => void;
  onAction?: (action: NotificationAction) => void;
  onBannerClick?: () => void;

  build() {
    Column() {
      Row() {
        // 图标
        SystemIcon({
          icon: this.getIcon(this.notification.type),
          iconSize: 24,
          iconColor: this.getPriorityColor(this.notification.priority)
        })
        .margin({ right: DesignConstants.SPACING.MD })

        // 内容
        Column() {
          Text(this.notification.title)
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontWeight(FontWeight.Bold)
            .fontColor($r('sys.color.ohos_id_color_text_primary'))
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(this.notification.message)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor($r('sys.color.ohos_id_color_text_secondary'))
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .margin({ top: DesignConstants.SPACING.XS })
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        // 关闭按钮
        SystemIcon({
          icon: AppIcon.CLOSE,
          iconSize: 16,
          iconColor: DesignConstants.COLOR.TEXT_TERTIARY_LIGHT,
          clickable: true,
          onIconClick: () => {
            if (this.onClose) {
              this.onClose();
            }
          }
        })
        .margin({ left: DesignConstants.SPACING.SM })
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)

      // 操作按钮
      if (this.notification.actions && this.notification.actions.length > 0) {
        Row({ space: DesignConstants.SPACING.MD }) {
          ForEach(this.notification.actions, (action: NotificationAction) => {
            Button(action.label)
              .type(ButtonType.Capsule)
              .height(28)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .backgroundColor(action.type === 'primary' ? $r('sys.color.ohos_id_color_emphasize') : $r('sys.color.ohos_id_color_component_normal'))
              .fontColor(action.type === 'primary' ? Color.White : $r('sys.color.ohos_id_color_text_primary'))
              .onClick(() => {
                if (this.onAction) {
                  this.onAction(action);
                }
              })
          })
        }
        .width('100%')
        .margin({ top: DesignConstants.SPACING.MD })
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor($r('sys.color.ohos_id_color_background'))
    .borderRadius(DesignConstants.RADIUS.LG)
    .shadow(DesignConstants.SHADOW.MEDIUM)
    .onClick(() => {
      if (this.onBannerClick) {
        this.onBannerClick();
      }
    })
  }

  private getIcon(type: string): AppIcon {
    switch (type) {
      case 'system': return AppIcon.SETTINGS;
      case 'performance': return AppIcon.PERFORMANCE;
      case 'social': return AppIcon.PERSON;
      case 'security': return AppIcon.SHIELD;
      default: return AppIcon.NOTIFICATION;
    }
  }

  private getPriorityColor(priority: string): string {
    switch (priority) {
      case 'critical': return DesignConstants.COLOR.ERROR;
      case 'high': return DesignConstants.COLOR.WARNING;
      case 'medium': return DesignConstants.COLOR.BRAND_PRIMARY;
      default: return DesignConstants.COLOR.TEXT_SECONDARY_LIGHT;
    }
  }
}
