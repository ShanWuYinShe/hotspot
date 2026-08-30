/**
 * ChatMessageItem —— 聊天页单条消息气泡（系统消息 / 文本 / 图片 + 时间与已读勾选）。
 * 从 ChatPage.buildMessageItem 抽取；主题对象与群/私聊上下文经 props 注入，
 * 无本地状态，保持纯展示。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { ChatThemeColors, ChatThemeManager } from '../../../core/theme/ChatTheme';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { MessageType } from '../../../core/types/api';
import { LucideIcon } from '../../../components/LucideIcon';

/** 消息展示视图模型（与 ChatPage 的 ChatMessage 结构一致） */
export interface ChatMessageView {
  id: string;
  content: string;
  senderId: string;
  senderName: string;
  senderAvatar?: string;
  type: MessageType;
  sentAt: string;
  isSelf: boolean;
  isSystem: boolean;
  rawData?: Object;
}

@Component
export struct ChatMessageItem {
  @Prop msg: ChatMessageView;
  @Prop isGroupChat: boolean = false;
  @Prop isLastSent: boolean = false;
  /** 预格式化时间文本（HH:mm），由页面统一格式化 */
  @Prop timeText: string = '';
  palette: AppThemePalette = AppThemeManager.getPaletteById('vibrant_light');
  chatTheme: ChatThemeColors = ChatThemeManager.current();

  private getBubbleRadius(isSelf: boolean): BorderRadiuses {
    const big: number = DesignConstants.CHAT.BUBBLE_RADIUS;
    const notch: number = DesignConstants.SPACING.XS;
    if (this.isGroupChat) {
      // 群聊：缺角在底部
      return isSelf
        ? { topLeft: big, topRight: big, bottomLeft: big, bottomRight: notch }
        : { topLeft: big, topRight: big, bottomLeft: notch, bottomRight: big };
    }
    // 私聊：缺角在顶部
    return isSelf
      ? { topLeft: big, topRight: notch, bottomLeft: big, bottomRight: big }
      : { topLeft: notch, topRight: big, bottomLeft: big, bottomRight: big };
  }

  build() {
    if (this.msg.isSystem) {
      Row() {
        Text(this.msg.content)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.palette.textSecondary)
          .backgroundColor(this.chatTheme?.bubbleSystem)
          .padding({
            left: DesignConstants.CHAT.IMAGE_BUBBLE_PADDING_H,
            right: DesignConstants.CHAT.IMAGE_BUBBLE_PADDING_H,
            top: DesignConstants.CHAT.IMAGE_BUBBLE_PADDING_V,
            bottom: DesignConstants.CHAT.IMAGE_BUBBLE_PADDING_V
          })
          .borderRadius(DesignConstants.CHAT.BUBBLE_RADIUS)
          .shadow(this.palette.controlShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.chatTheme?.divider })
      }
      .width('100%')
      .justifyContent(FlexAlign.Center)
      .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    } else {
      Row() {
        if (!this.msg.isSelf) {
          SystemAvatar({
            avatarUrl: this.msg.senderAvatar,
            name: this.msg.senderName,
            avatarSize: DesignConstants.CHAT.AVATAR_SIZE
          })
          .margin({ right: DesignConstants.SPACING.SM })
        }

        Column() {
          if (!this.msg.isSelf && this.isGroupChat) {
            Text(this.msg.senderName)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.palette.textSecondary)
              .margin({ bottom: DesignConstants.SPACING.XS, left: DesignConstants.SPACING.XS })
          }

          Row() {
            if (this.msg.type === 'image') {
              Image(MediaUrlUtil.normalize(this.msg.content))
                .width(DesignConstants.CHAT.IMAGE_MAX_WIDTH)
                .objectFit(ImageFit.Cover)
                .backgroundColor(this.chatTheme?.listBackground)
                .borderRadius(this.getBubbleRadius(this.msg.isSelf))
                .shadow(this.palette.controlShadow)
                .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.chatTheme?.divider })
                .margin(DesignConstants.SPACING.XS)
                .accessibilityText('[图片消息]')
            } else {
              Text(this.msg.content)
                .fontSize(DesignConstants.CHAT.BUBBLE_FONT_SIZE)
                .fontColor(this.msg.isSelf ? this.chatTheme?.sendButtonActiveText : this.palette.textPrimary)
                .backgroundColor(this.msg.isSelf ? this.chatTheme?.bubbleSelf : this.chatTheme?.bubblePeer)
                .padding({
                  left: DesignConstants.CHAT.BUBBLE_PADDING_H,
                  right: DesignConstants.CHAT.BUBBLE_PADDING_H,
                  top: DesignConstants.CHAT.BUBBLE_PADDING_V,
                  bottom: DesignConstants.CHAT.BUBBLE_PADDING_V
                })
                .borderRadius(this.getBubbleRadius(this.msg.isSelf))
                .constraintSize({ maxWidth: DesignConstants.CHAT.BUBBLE_MAX_WIDTH })
                .shadow(this.msg.isSelf ? this.palette.accentShadow : this.palette.controlShadow)
            }
          }
          .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)

          Row({ space: DesignConstants.SPACING.XS }) {
            Text(this.timeText)
              .fontSize(DesignConstants.FONT.SIZE.XS)
              .fontColor(this.msg.isSelf ? this.palette.textTertiary : this.palette.textSecondary)
              .opacity(DesignConstants.OPACITY.SUBTLE)
            if (this.msg.isSelf && this.isLastSent) {
              LucideIcon({
                  icon: AppIcon.CHECK_ALL,
                  size: DesignConstants.FONT.SIZE.XS,
                  color: this.palette.online
                })
            }
          }
          .width('100%')
          .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)
          .margin({ top: DesignConstants.SPACING.XS })
        }
        .alignItems(this.msg.isSelf ? HorizontalAlign.End : HorizontalAlign.Start)
        .layoutWeight(1)

        if (this.msg.isSelf) {
          SystemAvatar({
            avatarUrl: this.msg.senderAvatar,
            name: this.msg.senderName,
            avatarSize: DesignConstants.CHAT.AVATAR_SIZE
          })
          .margin({ left: DesignConstants.SPACING.SM })
        }
      }
      .width('100%')
      .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)
      .alignItems(VerticalAlign.Top)
      .margin({ bottom: DesignConstants.CHAT.MESSAGE_GAP })
    }
  }
}
