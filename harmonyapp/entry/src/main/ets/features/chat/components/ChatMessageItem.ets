/**
 * ChatMessageItem —— 聊天页单条消息气泡（系统消息 / 文本 / 图片 + 时间与已读勾选）。
 * 从 ChatPage.buildMessageItem 抽取；主题对象与群/私聊上下文经 props 注入，
 * 无本地状态，保持纯展示。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { ChatThemeColors, ChatThemeManager } from '../../../core/theme/ChatTheme';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { MessageType } from '../../../core/types/api';
import { LucideIcon } from '../../../components/LucideIcon';
import { AppRouter, PhotoPreviewRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';

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

  /** 图片加载失败标记（加载占位显示） */
  @State imageFailed: boolean = false;

  /** 图片点击预览（走 PhotoPreviewPage，与动态详情图片预览同一入口） */
  private previewImage(): void {
    const previewParams = new PhotoPreviewRouteParam();
    previewParams.momentId = '';
    previewParams.images = [MediaUrlUtil.normalize(this.msg.content)];
    previewParams.index = 0;
    AppRouter.push(PageNames.PHOTO_PREVIEW, previewParams);
  }

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
      // 对齐前端 MessageBubble：无侧边头像；时间 + 已读勾在气泡内部底行
      Row() {
        Column() {
          if (!this.msg.isSelf && this.isGroupChat) {
            Text(this.msg.senderName)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.palette.textSecondary)
              .margin({ bottom: DesignConstants.SPACING.XS, left: DesignConstants.SPACING.XS })
          }

          Column() {
            if (this.msg.type === 'image') {
              // 图片消息：限高（超高图防撑爆列表）+ 加载占位 + 点击预览
              Stack({ alignContent: Alignment.Center }) {
                if (this.imageFailed) {
                  // 加载失败占位（对齐前端图片占位：accent-soft 底 + 图标）
                  Column({ space: DesignConstants.SPACING.SM }) {
                    LucideIcon({
                      icon: AppIcon.PHOTO,
                      iconSize: DesignConstants.FONT.SIZE.LG,
                      paletteRole: 'textSecondary'
                    })
                    Text('图片加载失败')
                      .fontSize(DesignConstants.FONT.SIZE.XS)
                      .fontColor(this.palette.textSecondary)
                  }
                  .width(DesignConstants.CHAT.IMAGE_MAX_WIDTH)
                  .height(DesignConstants.CHAT.IMAGE_MAX_HEIGHT)
                  .justifyContent(FlexAlign.Center)
                  .backgroundColor(this.chatTheme?.listBackground)
                  .borderRadius(this.getBubbleRadius(this.msg.isSelf))
                } else {
                  Image(MediaUrlUtil.normalize(this.msg.content))
                    .width(DesignConstants.CHAT.IMAGE_MAX_WIDTH)
                    .height(DesignConstants.CHAT.IMAGE_MAX_HEIGHT)
                    .objectFit(ImageFit.Contain)
                    .backgroundColor(this.chatTheme?.listBackground)
                    .borderRadius(this.getBubbleRadius(this.msg.isSelf))
                    .shadow(this.palette.controlShadow)
                    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.chatTheme?.divider })
                    .margin(DesignConstants.SPACING.XS)
                    .onError(() => { this.imageFailed = true; })
                    .accessibilityText('[图片消息]')
                }
              }
              .width(DesignConstants.CHAT.IMAGE_MAX_WIDTH)
              .height(DesignConstants.CHAT.IMAGE_MAX_HEIGHT)
              .onClick(() => { this.previewImage(); })
              .accessibilityText('点击预览图片')
            } else {
              Text(this.msg.content)
                .fontSize(DesignConstants.CHAT.BUBBLE_FONT_SIZE)
                .fontColor(this.msg.isSelf ? this.chatTheme?.sendButtonActiveText : this.palette.textPrimary)
                .maxLines(200)

              // 气泡内底行：时间（自己=on-accent 70%；对方=textSecondary）+ 已读勾
              Row({ space: DesignConstants.SPACING.XS }) {
                Text(this.timeText)
                  .fontSize(DesignConstants.FONT.SIZE.XS2)
                  .fontColor(this.msg.isSelf ? this.chatTheme?.sendButtonActiveText : this.palette.textSecondary)
                  .opacity(this.msg.isSelf ? DesignConstants.OPACITY.PRESSED : 1)
                if (this.msg.isSelf && this.isLastSent) {
                  LucideIcon({
                      icon: AppIcon.CHECK_ALL,
                      iconSize: DesignConstants.FONT.SIZE.XS2,
                      color: this.chatTheme?.sendButtonActiveText
                    })
                    .opacity(DesignConstants.OPACITY.EMPHASIZED)
                }
              }
              .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)
              .width('100%')
              .margin({ top: DesignConstants.SPACING.HALF })
            }
          }
          .constraintSize({ maxWidth: DesignConstants.CHAT.BUBBLE_MAX_WIDTH })
          .alignItems(this.msg.isSelf ? HorizontalAlign.End : HorizontalAlign.Start)
          .padding({
            left: DesignConstants.CHAT.BUBBLE_PADDING_H,
            right: DesignConstants.CHAT.BUBBLE_PADDING_H,
            top: DesignConstants.CHAT.BUBBLE_PADDING_V,
            bottom: DesignConstants.CHAT.BUBBLE_PADDING_V
          })
          .backgroundColor(this.msg.isSelf ? this.chatTheme?.bubbleSelf : this.chatTheme?.bubblePeer)
          .borderRadius(this.getBubbleRadius(this.msg.isSelf))
          .shadow(this.msg.isSelf ? this.palette.accentShadow : this.palette.surfaceShadow)
          .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.msg.isSelf ? DesignConstants.COLOR.TRANSPARENT : this.chatTheme?.divider })
          .margin({ top: DesignConstants.SPACING.HALF })
        }
        .alignItems(this.msg.isSelf ? HorizontalAlign.End : HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')
      .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)
      .alignItems(VerticalAlign.Top)
      .margin({ bottom: DesignConstants.CHAT.MESSAGE_GAP })
    }
  }
}
