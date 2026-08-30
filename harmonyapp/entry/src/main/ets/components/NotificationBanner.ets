import { SmartNotification, NotificationAction } from '../core/services/notification/NotificationDataManager';
import { DesignConstants } from '../constants/DesignConstants';
import { SystemIcon, AppIcon } from './SystemIcon';
import { SystemButton } from './SystemButton';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { curves } from '@kit.ArkUI';

@Reusable
@Component
export struct NotificationBanner {
  @Prop notification: SmartNotification;
  onClose?: () => void;
  onAction?: (action: NotificationAction) => void;
  onBannerClick?: () => void;
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private isPressed: boolean = false;

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
            .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
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
        // 透明 padding 扩命中区至 40vp（16 + 2*12），视觉尺寸不变
        .padding(DesignConstants.SPACING.MD)
        .margin({ left: DesignConstants.SPACING.SM })
      }
      .width('100%')
      .alignItems(VerticalAlign.Top)

      // 操作按钮
      if (this.notification.actions && this.notification.actions.length > 0) {
        Row({ space: DesignConstants.SPACING.MD }) {
          ForEach(this.notification.actions, (action: NotificationAction) => {
            SystemButton({
              label: action.label,
              variant: action.type === 'primary' ? 'primary' : 'secondary',
              buttonSize: 'compact',
              onPress: () => {
                if (this.onAction) {
                  this.onAction(action);
                }
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
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getPalette().borderColor })
    .shadow(this.getPalette().elevatedShadow)
    // 整卡按压反馈（对齐 tap-97 语义）
    .scale(this.isPressed ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.isPressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.isPressed = false;
      }
    })
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
      case 'critical': return this.getPalette().error;
      case 'high': return this.getPalette().warning;
      case 'medium': return this.getPalette().textSecondary;
      default: return this.getPalette().textTertiary;
    }
  }
}
