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
import { DEFAULT_RESOLVED_THEME_ID, KEY_RESOLVED_THEME_ID } from '../core/theme/AppThemeTypes';
import { ChatThemeManager } from '../core/theme/ChatTheme';
import { AppIcon } from './SystemIcon';
import { SystemAvatar } from './SystemAvatar';
import { KeyboardDismissManager } from '../core/utils/KeyboardDismissManager';
import { SafeAreaUtils } from '../core/utils/SafeAreaUtils';
import { SurfaceIconButton } from './SurfaceIconButton';

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
  /** 在线状态副文本（在线/离线；空串不显示）。 */
  @Prop presenceText: string = '';
  /** 对端是否在线（决定副文本颜色）。 */
  @Prop presenceOnline: boolean = false;
  /**
   * 群聊模式（对齐前端 GroupChatView 头部）：
   * - 头像用群头像（圆角方 shape=squircle，对应 web shape="rounded"）
   * - 副文本 = 「正在输入…」提示（accent 色）/「N 位成员」/「群聊」
   * - 头像 + 标题整体可点进群详情（web avatar-entry button）
   */
  @Prop isGroup: boolean = false;
  @Prop groupAvatar: string = '';
  @Prop membersCount: number = 0;
  /** 群聊「正在输入」提示文本；非空时副文本切 accent 色。 */
  @Prop typingHint: string = '';
  onOpenGroupDetail: () => void = () => {};
  /** 是否显示尾随操作钮（web GroupChatView 无尾随按钮，群聊入口收敛到头部）。 */
  @Prop showTrailing: boolean = true;
  /** 返回回调。 */
  onBack: () => void = () => {};
  /** 尾随操作回调（打开会话设置/操作）。 */
  onTrailingAction: () => void = () => {};
  @StorageProp(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;

  private getPalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  build() {
    Row() {
      // 左侧：返回按钮 + 可选 leading 头像 + 标题
      Row() {
        // 返回钮统一走 SurfaceIconButton（与 SystemPageHeader / SearchPageHeader
        // 同款凸起中性 40vp：a11y + spring 按压 + 44 命中区由组件内聚，审查 中-1/低-4）
        SurfaceIconButton({
          icon: AppIcon.BACK,
          kind: 'surface',
          buttonSize: DesignConstants.LAYOUT.BACK_BUTTON_SIZE,
          glyphSize: DesignConstants.FONT.SIZE.LG_PLUS,
          accessibilityLabel: '返回',
          onPress: () => {
            this.onBack();
          }
        })

        if (this.isGroup || this.showAvatar) {
          SystemAvatar({
            avatarUrl: this.isGroup ? this.groupAvatar : this.targetAvatar,
            name: this.title,
            avatarSize: DesignConstants.COMPONENT.AVATAR_SIZE.SMALL,
            // 群聊头像圆角方（web shape="rounded"）；私聊保持圆形
            shape: this.isGroup ? 'squircle' : 'circle'
          })
            .margin({ left: DesignConstants.SPACING.XS })
        }

        // 标题 + 状态副文本（对齐前端：私聊 = 名字 medium + presence xs；
        // 群聊 = GroupChatView：头像+标题整体可点进群详情，副文本「N 位成员」/输入中提示）
        Column() {
          Text(this.title)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(this.isGroup ? DesignConstants.FONT.WEIGHT.REGULAR : DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.getPalette().textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          if (this.isGroup) {
            Text(this.typingHint.length > 0 ? this.typingHint :
              (this.membersCount > 0 ? `${this.membersCount} 位成员` : '群聊'))
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.typingHint.length > 0 ?
                this.getPalette().accentBackground : this.getPalette().textSecondary)
              .maxLines(1)
          } else if (this.presenceText.length > 0) {
            Text(this.presenceText)
              // 前端 presence 为 text-xs(11)；原 XS2(10) 为近邻档漂移
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.presenceOnline ? this.getPalette().online : this.getPalette().textSecondary)
              .maxLines(1)
          }
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)
        .margin({ left: DesignConstants.SPACING.SM })
        // 群聊：头像+标题整体进群详情（web avatar-entry）
        .onClick(() => {
          if (this.isGroup) {
            this.onOpenGroupDetail();
          }
        })
        .accessibilityText(this.isGroup ? `查看群「${this.title}」详情` : '')
      }
      .layoutWeight(1)

      // 尾随：会话操作 ghost 圆钮（私聊保留；群聊 web 无尾随按钮——入口已收敛到头部）
      if (this.showTrailing) {
        SurfaceIconButton({
          icon: AppIcon.MORE,
          kind: 'surface',
          buttonSize: DesignConstants.COMPONENT.AVATAR_SIZE.XS,
          glyphSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
          accessibilityLabel: '会话操作',
          onPress: () => {
            this.onTrailingAction();
          }
        })
      }
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.MD,
      right: DesignConstants.SPACING.MD,
      top: DesignConstants.SPACING.SM2,
      bottom: DesignConstants.SPACING.SM2
    })
    .margin({ top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop, 0), bottom: DesignConstants.SPACING.XS })
    .border({ width: { bottom: DesignConstants.BORDER_WIDTH.HAIRLINE }, color: this.getPalette().borderColor })
    .zIndex(1)
    .onTouch((event: TouchEvent) => {
      // 点头部空白处收起输入法（与原 ChatPage 行为一致）
      if (event.type === TouchType.Down) {
        KeyboardDismissManager.getInstance().dismiss();
      }
    })
  }
}
