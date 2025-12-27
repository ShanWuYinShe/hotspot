import { SmartNotification, NotificationAction } from '../core/services/notification/NotificationDataManager';
import { DesignConstants } from '../constants/DesignConstants';
import { SystemIcon, AppIcon } from './SystemIcon';

@Component
export struct NotificationBanner {
  @ObjectLink notification: SmartNotification; // 使用ObjectLink以响应变化，或者Prop如果数据简单
  onClose?: () => void;
  onAction?: (action: NotificationAction) => void;
  onClick?: () => void;

  build() {
    Column() {
      Row() {
        // 图标
        SystemIcon({
          icon: this.getIcon(this.notification.type),
          iconSize: 24,
          iconColor: this.getPriorityColor(this.notification.priority)
        })
        .margin({ right: 12 })

        // 内容
        Column() {
          Text(this.notification.title)
            .fontSize(16)
            .fontWeight(FontWeight.Bold)
            .fontColor(DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(this.notification.message)
            .fontSize(14)
            .fontColor(DesignConstants.COLOR.TEXT_SECONDARY_LIGHT)
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .margin({ top: 4 })
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
        .margin({ left: 8 })
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)

      // 操作按钮
      if (this.notification.actions && this.notification.actions.length > 0) {
        Row({ space: 12 }) {
          ForEach(this.notification.actions, (action: NotificationAction) => {
            Button(action.label)
              .type(ButtonType.Capsule)
              .height(28)
              .fontSize(12)
              .backgroundColor(action.type === 'primary' ? DesignConstants.COLOR.BRAND_PRIMARY : DesignConstants.COLOR.TERTIARY_BG_LIGHT)
              .fontColor(action.type === 'primary' ? Color.White : DesignConstants.COLOR.TEXT_PRIMARY_LIGHT)
              .onClick(() => {
                if (this.onAction) {
                  this.onAction(action);
                }
              })
          })
        }
        .width('100%')
        .margin({ top: 12 })
      }
    }
    .width('100%')
    .padding(16)
    .backgroundColor(DesignConstants.COLOR.PRIMARY_BG_LIGHT)
    .borderRadius(DesignConstants.RADIUS.LG)
    .shadow(DesignConstants.SHADOW.MEDIUM)
    .onClick(() => {
      if (this.onClick) {
        this.onClick();
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










