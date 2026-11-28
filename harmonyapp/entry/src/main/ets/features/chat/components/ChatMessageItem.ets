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
import { Reaction } from '../../../core/types/api';
import { MotionUtil } from '../../../core/accessibility/MotionUtil';
import { ChatMessageListItem, TypingMessageSource } from './ChatMessageListUtils';
import { LucideIcon } from '../../../components/LucideIcon';
import { AppRouter, PhotoPreviewRouteParam } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';

/** 消息展示视图模型（extends 列表项最小集 + 页面共享扩展，替代三处重复字段声明） */
export interface ChatMessageView extends ChatMessageListItem, TypingMessageSource {
  senderAvatar?: string;
  /** 已编辑（时间行追加「· 已编辑」标记，对齐前端 editedAt） */
  edited?: boolean;
  /** 表情反应（spec types.md → Reaction[]；H-03 消息点赞） */
  reactions?: Reaction[];
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
  /** 表情反应切换（emoji + active）；未注入时不渲染反应条（H-03） */
  onToggleReaction?: (emoji: string, active: boolean) => void;
  /** 气泡点击上抛（唤起消息操作面板） */
  onMessageClick?: (msg: ChatMessageView) => void = undefined;

  /** 图片加载失败标记（加载占位显示） */
  @State imageFailed: boolean = false;
  /** 图片加载完成标记（onComplete 后淡入；reduced-motion 直接显示） */
  @State imageLoaded: boolean = false;
  /** 图片宽高比（onComplete 读取真实尺寸）：图片消息按原始比例铺满气泡，避免固定框留白/裁切 */
  @State imageAspectRatio: number = 0;

  // 图片长按：onTouch 计时器（List 内 LongPressGesture 与滚动手势竞争不稳定）
  private longPressTimer: number = -1;
  private longPressFired: boolean = false;
  private longPressStartX: number = 0;
  private longPressStartY: number = 0;

  private beginMessageLongPress(): void {
    this.cancelMessageLongPress();
    this.longPressFired = false;
    this.longPressTimer = setTimeout(() => {
      this.longPressTimer = -1;
      this.longPressFired = true;
      if (this.hasActions && this.onMessageClick) {
        this.onMessageClick(this.msg);
      }
    }, 500);
  }

  private cancelMessageLongPress(): void {
    if (this.longPressTimer !== -1) {
      clearTimeout(this.longPressTimer);
      this.longPressTimer = -1;
    }
  }

  aboutToDisappear(): void {
    this.cancelMessageLongPress();
  }

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

  /** 图片消息框宽度：按原始比例在 MAX_W/MAX_H 内缩放到最小可用框（loading 阶段用默认 3:4） */
  private imageBoxWidth(): number {
    const ratio = this.imageAspectRatio;
    if (ratio <= 0) {
      return DesignConstants.CHAT.IMAGE_MAX_WIDTH;
    }
    return Math.min(DesignConstants.CHAT.IMAGE_MAX_WIDTH, Math.round(DesignConstants.CHAT.IMAGE_MAX_HEIGHT * ratio));
  }

  private imageBoxRatio(): number {
    return this.imageAspectRatio > 0 ? this.imageAspectRatio : 0.75;
  }

  /** 图片消息内容：无气泡外壳（不再有蓝色大边框），图片自身圆角贴齐页面背景，时间/已读在图下 */
  @Builder
  private buildImageMessage(): void {
    Column() {
      Stack({ alignContent: Alignment.Center }) {
        if (this.imageFailed) {
          // 加载失败占位（对齐前端图片占位：surface-subtle 底 + 图标）
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
          .width(this.imageBoxWidth())
          .aspectRatio(this.imageBoxRatio())
          .justifyContent(FlexAlign.Center)
          .backgroundColor(this.palette.surfaceSubtle)
          .borderRadius(DesignConstants.RADIUS.MD)
        } else {
          Image(ImageUrlUtil.optimized(MediaUrlUtil.normalize(this.msg.content), 720))
            .width('100%')
            .height('100%')
            .objectFit(ImageFit.Cover)
            .borderRadius(DesignConstants.RADIUS.MD)
            .backgroundColor(this.imageLoaded ? DesignConstants.COLOR.TRANSPARENT : this.palette.surfaceSubtle)
            // 慢网淡入（对齐 NearbyImageCard/SystemAvatar 口径）：加载中保留占位底，
            // onComplete 后 200ms 淡入；失败保持占位并出「图片加载失败」；reduced 直接终态。
            .opacity(this.imageLoaded || MotionUtil.reduced() ? 1 : 0)
            .animation({
              duration: MotionUtil.reduced() ? 0 : DesignConstants.ANIMATION.DURATION_SHORT,
              curve: DesignConstants.ANIMATION.CURVE_EASE_OUT
            })
            .onComplete((event) => {
              // 读取真实像素尺寸按比例布局（ImageCompleteEvent 类型由 ArkUI 全局声明推导）
              if (event && event.contentWidth > 0 && event.contentHeight > 0) {
                this.imageAspectRatio = event.contentWidth / event.contentHeight;
              }
              this.imageLoaded = true;
            })
            .onError(() => { this.imageFailed = true; })
            .accessibilityText('[图片消息]')
        }
      }
      .width(this.imageBoxWidth())
      .aspectRatio(this.imageBoxRatio())
      .borderRadius(DesignConstants.RADIUS.MD)
      .clip(true)
      // 对齐 Web/Android：图片点按始终预览大图；消息操作（撤回/举报）改长按唤起，
      // 不再让「点击=操作面板」占用预览入口（自己刚发的图此前只能通过操作面板、无法看大图）。
      // 长按用 onTouch 计时器实现：List 内嵌图片上 LongPressGesture 与列表手势竞争不稳定，
      // 计时器在 Down 起 500ms 触发、移动超过 10vp 或抬手取消，行为与系统长按一致。
      .onClick(() => {
        if (!this.longPressFired) {
          this.previewImage();
        }
        this.longPressFired = false;
      })
      .onTouch((event: TouchEvent) => {
        if (event.type === TouchType.Down) {
          const downTouch = event.touches.length > 0 ? event.touches[0] : undefined;
          if (downTouch) {
            this.longPressStartX = downTouch.screenX;
            this.longPressStartY = downTouch.screenY;
          }
          this.beginMessageLongPress();
        } else if (event.type === TouchType.Move) {
          const touch = event.touches.length > 0 ? event.touches[0] : undefined;
          if (touch && (Math.abs(touch.screenX - this.longPressStartX) > 10 ||
            Math.abs(touch.screenY - this.longPressStartY) > 10)) {
            this.cancelMessageLongPress();
          }
        } else {
          this.cancelMessageLongPress();
        }
      })
      .accessibilityText(this.hasActions ? '点击预览图片，长按查看消息操作' : '点击预览图片')

      // 图片不套气泡：时间/已读图下对齐（次色，深浅主题均可读）
      Row({ space: DesignConstants.SPACING.XS }) {
        Text(this.timeText)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.palette.textSecondary)
        if (this.msg.isSelf && this.isLastSent) {
          LucideIcon({
            icon: AppIcon.CHECK_ALL,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.palette.textSecondary
          })
            .opacity(DesignConstants.OPACITY.EMPHASIZED)
        }
      }
      .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)
      .margin({ top: DesignConstants.SPACING.XS })
    }
    .alignItems(this.msg.isSelf ? HorizontalAlign.End : HorizontalAlign.Start)
    .margin({ top: DesignConstants.SPACING.HALF })
  }

  /** 视频消息：原生 Video 播放器（controls，不自动播放），宽度与图片气泡一致（对齐 Web <video>）。 */
  @Builder
  private buildVideoMessage(): void {
    Column() {
      Video({
        src: MediaUrlUtil.normalize(this.msg.content)
      })
        .controls(true)
        .width(DesignConstants.CHAT.IMAGE_MAX_WIDTH)
        .aspectRatio(16 / 9)
        .borderRadius(DesignConstants.RADIUS.MD)
        .objectFit(ImageFit.Contain)
        .backgroundColor(this.palette.surfaceSubtle)
        .accessibilityText('[视频消息]')

      Row({ space: DesignConstants.SPACING.XS }) {
        Text(this.timeText)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.palette.textSecondary)
        if (this.msg.isSelf && this.isLastSent) {
          LucideIcon({
            icon: AppIcon.CHECK_ALL,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_2XS,
            color: this.palette.textSecondary
          })
            .opacity(DesignConstants.OPACITY.EMPHASIZED)
        }
      }
      .justifyContent(this.msg.isSelf ? FlexAlign.End : FlexAlign.Start)
      .margin({ top: DesignConstants.SPACING.XS })
    }
    .alignItems(this.msg.isSelf ? HorizontalAlign.End : HorizontalAlign.Start)
    .margin({ top: DesignConstants.SPACING.HALF })
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

          if (this.msg.type === 'image') {
            this.buildImageMessage()
          } else if (this.msg.type === 'video') {
            this.buildVideoMessage()
          } else {
            Column() {
              // 已撤回：斜体「消息已撤回」（对齐前端 MessageBubble recalled 态）
              Text(this.msg.recalled ? '消息已撤回' : this.msg.content)
                .fontSize(DesignConstants.FONT.SIZE.MD)
                // 气泡正文行高对齐前端 MessageBubble 的 leading-chat(1.43)
                .lineHeight(lineHeightFor(DesignConstants.FONT.SIZE.MD, DesignConstants.FONT.LINE_RATIO.CHAT))
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
              // 不能写 width('100%')：气泡宽度由内容决定，百分比子节点会把气泡撑到 maxWidth，
              // 短消息变成整条长条（时间/已读行只需随内容并靠齐）
              .margin({ top: DesignConstants.SPACING.XS })
            }
            .constraintSize({ maxWidth: DesignConstants.CHAT.BUBBLE_MAX_WIDTH })
            .alignItems(this.msg.isSelf ? HorizontalAlign.End : HorizontalAlign.Start)
            // 内距对齐 web px-3 py-1.5（12/6）
            .padding({
              left: DesignConstants.CHAT.BUBBLE_PADDING_H,
              right: DesignConstants.CHAT.BUBBLE_PADDING_H,
              top: DesignConstants.CHAT.BUBBLE_PADDING_V,
              bottom: DesignConstants.CHAT.BUBBLE_PADDING_V
            })
            .backgroundColor(this.getBubbleBackground())
            .borderRadius(DesignConstants.CHAT.BUBBLE_RADIUS)
            // 气泡纯色底无边框无阴影（对齐前端 MessageBubble 与主流 IM；
            // 描边+阴影会让气泡呈现「浮肿的框」）
            .margin({ top: DesignConstants.SPACING.HALF })
            .onClick(() => {
              if (this.hasActions && this.onMessageClick) {
                this.onMessageClick(this.msg);
              }
            })
          }
          if (!this.msg.recalled) {
            ForEach((this.msg.reactions ?? []).filter((r: Reaction) => r.count > 0), (r: Reaction) => {
              Row({ space: DesignConstants.SPACING.HALF }) {
                Text(r.emoji).fontSize(DesignConstants.FONT.SIZE.XS)
                Text(`${r.count}`)
                  .fontSize(DesignConstants.FONT.SIZE.XS)
                  .fontColor(r.reacted_by_me ? this.palette.textAccentOnSoft : this.palette.textSecondary)
              }
              .padding({
                left: DesignConstants.SPACING.SM,
                right: DesignConstants.SPACING.SM,
                top: DesignConstants.SPACING.HALF,
                bottom: DesignConstants.SPACING.HALF
              })
              .backgroundColor(r.reacted_by_me ? this.palette.accentMuted : this.palette.surfaceBackground)
              .border({ width: 1, color: r.reacted_by_me ? this.palette.accentBackground : this.palette.borderColor })
              .borderRadius(DesignConstants.RADIUS.FULL)
              .margin({ top: DesignConstants.SPACING.XS, right: DesignConstants.SPACING.XS })
              .onClick(() => {
                if (this.onToggleReaction) this.onToggleReaction(r.emoji, !r.reacted_by_me);
              })
              .accessibilityText(`表情反应 ${r.emoji}，${r.count} 个`)
            }, (r: Reaction) => `${r.emoji}_${r.count}_${r.reacted_by_me ? 1 : 0}`)
          }
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
