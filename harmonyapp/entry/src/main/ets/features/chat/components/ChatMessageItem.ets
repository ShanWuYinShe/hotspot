/**
 * ChatMessageItem —— 聊天页单条消息气泡（系统消息 / 文本 / 图片 + 时间与已读勾选）。
 * 从 ChatPage.buildMessageItem 抽取；主题对象与群/私聊上下文经 props 注入，
 * 无本地状态，保持纯展示。
 *
 * 视觉对齐前端 MessageBubble：
 * - 圆角统一 16（不再做「缺角」气泡，web 已收敛成 rounded-md 单档）
 * - 正文 14 + 行高 1.7（leading-card）、内距 16/8（web px-4 py-2）
 * - 时间戳 11、与正文留 4（mt-1）；自己气泡上的正文/时间一律 full on-accent
 *   （web 明确去掉 opacity——实底上叠透明度会跌破 4.5:1）
 * - 撤回 = 中性底 + 次色斜体（不留在 accent 底上）
 */

import { DesignConstants, lineHeightFor } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemePalette, DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { ChatThemeColors, ChatThemeManager } from '../../../core/theme/ChatTheme';
import { MediaUrlUtil } from '../../../core/utils/MediaUrlUtil';
import { ImageUrlUtil } from '../../../core/utils/ImageUrlUtil';
import { ChatMessageListItem, TypingMessageSource } from './ChatMessageListUtils';
import { LucideIcon } from '../../../components/LucideIcon';
import { AppRouter, PhotoPreviewRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';

/** 消息展示视图模型（extends 列表项最小集 + 页面共享扩展，替代三处重复字段声明） */
export interface ChatMessageView extends ChatMessageListItem, TypingMessageSource {
  senderAvatar?: string;
  /** 已编辑（时间行追加「· 已编辑」标记，对齐前端 editedAt） */
  edited?: boolean;
  rawData?: Object;
}

@Component
export struct ChatMessageItem {
  @Prop msg: ChatMessageView;
  @Prop isGroupChat: boolean = false;
  @Prop isLastSent: boolean = false;
  /** 预格式化时间文本（HH:mm），由页面统一格式化 */
  @Prop timeText: string = '';
  palette: AppThemePalette = AppThemeManager.getPaletteById(DEFAULT_RESOLVED_THEME_ID);
  chatTheme: ChatThemeColors = ChatThemeManager.current();

  /** 气泡是否可唤起操作面板（编辑/撤回；由页面按权限计算） */
  @Prop hasActions: boolean = false;
  /** 气泡点击上抛（唤起消息操作面板） */
  onMessageClick?: (msg: ChatMessageView) => void = undefined;

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

  /** 气泡外皮：撤回 = 中性底（两个方向统一）；自己 = accent 实底；对方 = 白气泡 */
  private getBubbleBackground(): string {
    if (this.msg.recalled) {
      return this.palette.surfaceSubtle;
    }
    return this.msg.isSelf ? this.chatTheme.bubbleSelf : this.chatTheme.bubblePeer;
  }

  /** 气泡正文色：撤回/对方 = 主文字或次色；自己 = on-accent（不叠透明度） */
  private getBubbleTextColor(): string {
    if (this.msg.recalled) {
      return this.palette.textSecondary;
    }
    return this.msg.isSelf ? this.chatTheme.bubbleSelfText : this.palette.textPrimary;
  }

  /** 气泡内元信息（时间/已编辑/已读勾）颜色：跟随气泡底色，一律不叠 opacity */
  private getMetaColor(): string {
    if (this.msg.recalled) {
      return this.palette.textSecondary;
    }
    return this.msg.isSelf ? this.chatTheme.bubbleSelfText : this.palette.textSecondary;
  }

  build() {
    if (this.msg.isSystem) {
      Row() {
        Text(this.msg.content)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.palette.textSecondary)
          .backgroundColor(this.chatTheme.bubbleSystem)
          .padding({
            left: DesignConstants.CHAT.IMAGE_BUBBLE_PADDING_H,
            right: DesignConstants.CHAT.IMAGE_BUBBLE_PADDING_H,
            top: DesignConstants.CHAT.IMAGE_BUBBLE_PADDING_V,
            bottom: DesignConstants.CHAT.IMAGE_BUBBLE_PADDING_V
          })
          .borderRadius(DesignConstants.CHAT.BUBBLE_RADIUS)
          // 系统 pill 纯底色即可，描边+阴影过重
          .border({ width: 0, color: DesignConstants.COLOR.TRANSPARENT })
      }
      .width('100%')
      .justifyContent(FlexAlign.Center)
      .margin({ top: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
    } else {
      // 对齐前端 MessageBubble：无侧边头像；时间 + 已读勾在气泡内部底行
      Row() {
        Column() {
          if (!this.msg.isSelf && this.isGroupChat) {
            // 群聊收到：气泡上方发送者昵称（web mb-1 px-1 text-sm font-medium）
            Text(this.msg.senderName)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
              .maxLines(1)
              .textOverflow({ overflow: TextOverflow.Ellipsis })
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
                      iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
                      paletteRole: 'textSecondary'
                    })
                    Text('图片加载失败')
                      .fontSize(DesignConstants.FONT.SIZE.XS)
                      .fontColor(this.palette.textSecondary)
                  }
                  .width(DesignConstants.CHAT.IMAGE_MAX_WIDTH)
                  .height(DesignConstants.CHAT.IMAGE_MAX_HEIGHT)
                  .justifyContent(FlexAlign.Center)
                  .backgroundColor(this.chatTheme.listBackground)
                  .borderRadius(DesignConstants.CHAT.BUBBLE_RADIUS)
                  } else {
                  Image(ImageUrlUtil.optimized(MediaUrlUtil.normalize(this.msg.content), 720))
                    .width(DesignConstants.CHAT.IMAGE_MAX_WIDTH)
                    .height(DesignConstants.CHAT.IMAGE_MAX_HEIGHT)
                    .objectFit(ImageFit.Contain)
                    .backgroundColor(this.chatTheme.listBackground)
                    .borderRadius(DesignConstants.CHAT.BUBBLE_RADIUS)
                    .margin(DesignConstants.SPACING.XS)
                    .onError(() => { this.imageFailed = true; })
                    .accessibilityText('[图片消息]')
                }
              }
              .width(DesignConstants.CHAT.IMAGE_MAX_WIDTH)
              .height(DesignConstants.CHAT.IMAGE_MAX_HEIGHT)
              .onClick(() => {
                // 有可操作项（撤回）时点击唤起操作面板，否则走图片预览
                if (this.hasActions && this.onMessageClick) {
                  this.onMessageClick(this.msg);
                  return;
                }
                this.previewImage();
              })
              .accessibilityText('点击预览图片')
              // 图片气泡也带时间/已读底行（与文本气泡一致，Telegram 规范）
              Row({ space: DesignConstants.SPACING.XS }) {
                Text(this.timeText)
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .fontColor(this.getMetaColor())
                if (this.msg.isSelf && this.isLastSent) {
                  LucideIcon({
                    icon: AppIcon.CHECK_ALL,
                    iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                    color: this.getMetaColor()
                  })
                    .opacity(DesignConstants.OPACITY.EMPHASIZED)
                }
              }
              .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)
              .width('100%')
              .margin({ top: DesignConstants.SPACING.XS })
            } else {
              // 已撤回：斜体「消息已撤回」（对齐前端 MessageBubble recalled 态）
              Text(this.msg.recalled ? '消息已撤回' : this.msg.content)
                .fontSize(DesignConstants.FONT.SIZE.MD)
                // 气泡正文行高对齐前端 MessageBubble 的 leading-card(1.7)
                .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.MD, DesignConstants.FONT.LINE_RATIO.CARD))
                .fontColor(this.getBubbleTextColor())
                .fontStyle(this.msg.recalled === true ? FontStyle.Italic : FontStyle.Normal)
                .maxLines(200)

              // 气泡内底行：时间（11，mt-1）+ 已编辑 + 已读勾，颜色不叠 opacity
              Row({ space: DesignConstants.SPACING.XS }) {
                Text(this.timeText)
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .fontColor(this.getMetaColor())
                if (this.msg.edited && !this.msg.recalled) {
                  Text('· 已编辑')
                    .fontSize(DesignConstants.FONT.SIZE.XS)
                    .fontColor(this.getMetaColor())
                }
                if (this.msg.isSelf && this.isLastSent && !this.msg.recalled) {
                  LucideIcon({
                      icon: AppIcon.CHECK_ALL,
                      iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
                      color: this.getMetaColor()
                    })
                    .opacity(DesignConstants.OPACITY.EMPHASIZED)
                }
              }
              .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)
              .width('100%')
              .margin({ top: DesignConstants.SPACING.XS })
            }
          }
          .constraintSize({ maxWidth: DesignConstants.CHAT.BUBBLE_MAX_WIDTH })
          .alignItems(this.msg.isSelf ? HorizontalAlign.End : HorizontalAlign.Start)
          // 内距对齐 web px-4 py-2（16/8）
          .padding({
            left: DesignConstants.CHAT.BUBBLE_PADDING_H,
            right: DesignConstants.CHAT.BUBBLE_PADDING_H,
            top: DesignConstants.SPACING.SM,
            bottom: DesignConstants.SPACING.SM
          })
          .backgroundColor(this.getBubbleBackground())
          .borderRadius(DesignConstants.CHAT.BUBBLE_RADIUS)
          // 气泡纯色底无边框无阴影（对齐前端 MessageBubble 与主流 IM；
          // 描边+阴影会让气泡呈现「浮肿的框」）
          .margin({ top: DesignConstants.SPACING.HALF })
          .onClick(() => {
            if (this.msg.type === 'image') {
              return;
            }
            if (this.hasActions && this.onMessageClick) {
              this.onMessageClick(this.msg);
            }
          })
        }
        .alignItems(this.msg.isSelf ? HorizontalAlign.End : HorizontalAlign.Start)
        .layoutWeight(1)
      }
      .width('100%')
      .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)
      .alignItems(VerticalAlign.Top)
      // 消息间距对齐 web gap-2(8)；原 MESSAGE_GAP(16) 会让一轮对话散开、读不出「一问一答」
      .margin({ bottom: DesignConstants.SPACING.SM })
    }
  }
}
