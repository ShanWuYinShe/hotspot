import { SmartNotification, NotificationAction } from '../core/services/notification/NotificationDataManager';
import { DesignConstants } from '../constants/DesignConstants';
import { SystemIcon, AppIcon } from './SystemIcon';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';

@Reusable
@Component
export struct NotificationBanner {
  @Prop notification: SmartNotification;
  onClose?: () => void;
  onAction?: (action: NotificationAction) => void;
  onBannerClick?: () => void;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'figma_neu_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

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
            .fontColor(this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })

          Text(this.notification.message)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(this.getPalette().textSecondary)
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
          iconColor: this.getPalette().textTertiary,
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
              .type(ButtonType.Normal)
              .height(28)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .backgroundColor(action.type === 'primary' ? this.getPalette().accentBackground : this.getPalette().buttonSecondaryBackground)
              .fontColor(action.type === 'primary' ? this.getPalette().accentText : this.getPalette().buttonSecondaryText)
              .borderRadius(DesignConstants.RADIUS.XL)
              .onClick(() => {
                if (this.onAction) {
                  this.onAction(action);
                }
              })
          }, (action: NotificationAction) => action.label)
        }
        .width('100%')
        .margin({ top: DesignConstants.SPACING.MD })
      }
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(this.getPalette().surfaceBackground)
    .borderRadius(DesignConstants.RADIUS.XL)
    .border({ width: 1, color: this.getPalette().borderColor })
    .shadow(this.getPalette().elevatedShadow)
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

  private getPriorityColor(priority: string): string | Resource {
    switch (priority) {
      case 'critical': return this.getPalette().textTertiary;
      case 'high': return this.getPalette().textSecondary;
      case 'medium': return this.getPalette().accentBackground;
      default: return this.getPalette().textSecondary;
    }
  }
}
