/**
 * ChatPageHeader
 * - 浮动圆角卡片头部（替代 ChatPage 手搓头部）
 * - 结构：返回按钮 + 可选 leading 头像 + 标题 + 尾随 INFO 操作
 * - 卡片样式：elevatedShadow + titleBarBorder + RADIUS.XL + zIndex
 * - onTouch 下沉键盘（点头部空白处收起输入法）
 *
 * 与 SystemPageHeader 的差异：浮动卡片而非全宽底栏，且需要 leading 头像 + 键盘下沉，
 * 故独立组件，仍复用同套主题/DesignConstants 原语。
 */

import { DesignConstants } from '../constants/DesignConstants';
import { AppThemeManager } from '../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { ChatThemeManager } from '../core/theme/ChatTheme';
import { AppIcon } from './SystemIcon';
import { SystemAvatar } from './SystemAvatar';
import { KeyboardDismissManager } from '../core/utils/KeyboardDismissManager';
import { SafeAreaUtils } from '../core/utils/SafeAreaUtils';

@Component
export struct ChatPageHeader {
  /** 顶部安全区高度（全屏 edge-to-edge 下标题避让状态栏，SystemPageHeader 内聚同款逻辑） */
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  /** 标题（会话名）。 */
  @Prop title: string = '';
  /** 私聊时对方头像 URL；提供则显示 leading 头像。 */
  @Prop targetAvatar: string = '';
  /** 是否展示 leading 头像（私聊 true / 群聊 false）。 */
  @Prop showAvatar: boolean = false;
  /** 返回回调。 */
  onBack: () => void = () => {};
  /** 尾随操作回调（打开会话设置/操作）。 */
  onTrailingAction: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row() {
      // 左侧：返回按钮 + 可选 leading 头像 + 标题
      Row() {
        Button() {
          Text() {
            SymbolSpan($r(AppIcon.BACK))
              .fontSize(DesignConstants.FONT.SIZE.XL)
              .fontColor([this.getPalette().textPrimary])
          }
        }
        .type(ButtonType.Circle)
        .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
        .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
        .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
        .onClick(() => {
          this.onBack();
        })

        if (this.showAvatar && this.targetAvatar.length > 0) {
          SystemAvatar({
            avatarUrl: this.targetAvatar,
            name: this.title,
            avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
            shape: 'circle'
          })
            .margin({ left: DesignConstants.SPACING.XS })
        }

        Text(this.title)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
          .fontColor(this.getPalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
          .layoutWeight(1)
          .margin({ left: DesignConstants.SPACING.SM })
      }
      .layoutWeight(1)

      // 尾随：会话信息/操作按钮
      Button() {
        Text() {
          SymbolSpan($r(AppIcon.INFO))
            .fontSize(DesignConstants.FONT.SIZE.XL)
            .fontColor([this.getPalette().textSecondary])
        }
      }
      .type(ButtonType.Circle)
      .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .backgroundColor(DesignConstants.COLOR.TRANSPARENT)
      .onClick(() => {
        this.onTrailingAction();
      })
    }
    .width('100%')
    .margin({ top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop), bottom: DesignConstants.SPACING.XS })
    .shadow(this.getPalette().elevatedShadow)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: ChatThemeManager.current().titleBarBorder })
    .borderRadius(DesignConstants.RADIUS.XL)
    .zIndex(1)
    .onTouch((event: TouchEvent) => {
      // 点头部空白处收起输入法（与原 ChatPage 行为一致）
      if (event.type === TouchType.Down) {
        KeyboardDismissManager.getInstance().dismiss();
      }
    })
  }
}
