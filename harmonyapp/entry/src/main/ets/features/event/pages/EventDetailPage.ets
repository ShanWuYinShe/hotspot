/**
 * 活动详情页面
 * Figma Make EventDetail 对齐
 * 包含封面图、活动信息、主办方、参与者、标签、底部操作按钮
 */

import { AppRouter } from '../../../core/router/AppRouter';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';
import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemAvatar } from '../../../components/SystemAvatar';
import { SystemToast } from '../../../components/SystemToast';
import { SystemUiController } from '../../../core/utils/SystemUiController';
import { SafeAreaUtils } from '../../../core/utils/SafeAreaUtils';
import { AppIcon } from '../../../components/SystemIcon';

/** 活动数据结构 */
interface EventItem {
  id: number;
  title: string;
  tag: string;
  tagColor: string;
  coverGradient: string[];
  description: string;
  location: string;
  date: string;
  time: string;
  hostName: string;
  hostAvatar: string;
  participantsCount: number;
  maxParticipants: number;
  distance: string;
  highlights: string[];
}

/** Mock 活动数据 */
const MOCK_EVENTS: EventItem[] = [
  {
    id: 1,
    title: '晨间咖啡会',
    tag: '社交',
    tagColor: DesignConstants.COLOR.WARNING,
    coverGradient: [DesignConstants.COLOR.BRAND_PRIMARY_DARK, DesignConstants.COLOR.BRAND_PRIMARY],
    description: '每周六早上九点，我们聚集在老街咖啡馆，分享一周的所见所闻。无论是设计师、程序员还是艺术家，都欢迎加入！带上你的好奇心就够了。',
    location: '老街咖啡馆 · 徐州市鼓楼区',
    date: '2026年4月18日 周六',
    time: '09:00 - 11:00',
    hostName: 'Aria Sol',
    hostAvatar: '',
    participantsCount: 8,
    maxParticipants: 20,
    distance: '0.3km',
    highlights: ['免费咖啡', '结识新朋友', '分享创意', '每周固定']
  },
  {
    id: 2,
    title: '街头音乐节',
    tag: '音乐',
    tagColor: DesignConstants.COLOR.BRAND_PRIMARY,
    coverGradient: [DesignConstants.COLOR.BRAND_PRIMARY, DesignConstants.COLOR.BRAND_PRIMARY_DARK],
    description: '在城市的中心，用音乐连接每一颗心。本次街头音乐节将邀请本地独立乐队和歌手进行现场表演，免费入场，欢迎所有人来感受音乐的力量。',
    location: '彭城广场 · 徐州市云龙区',
    date: '2026年4月19日 周日',
    time: '14:00 - 20:00',
    hostName: '本地向导',
    hostAvatar: '',
    participantsCount: 45,
    maxParticipants: 200,
    distance: '0.7km',
    highlights: ['免费入场', '现场表演', '多个乐队', '本地艺术家']
  },
  {
    id: 3,
    title: '周末摄影徒步',
    tag: '户外',
    tagColor: DesignConstants.COLOR.SUCCESS,
    coverGradient: [DesignConstants.COLOR.SUCCESS, DesignConstants.MAP_MARKER.GROUP],
    description: '用镜头记录徐州春天的美。从云龙湖到户部山，我们将沿着精心规划的路线，一起捕捉城市与自然的交融。适合所有水平的摄影爱好者。',
    location: '云龙湖风景区 · 徐州市泉山区',
    date: '2026年4月25日 周六',
    time: '08:30 - 12:30',
    hostName: '光影俱乐部',
    hostAvatar: '',
    participantsCount: 12,
    maxParticipants: 30,
    distance: '1.2km',
    highlights: ['专业领队', '器材交流', '后期分享', '免费参加']
  }
];

@Preview
@Component
export struct EventDetailPage {
  pathStack: NavPathStack = new NavPathStack();
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'default_clean_light';
  @StorageLink('safeAreaTop') safeAreaTop: number = 0;
  @StorageLink('safeAreaBottom') safeAreaBottom: number = 0;

  private eventId: number = 1;
  @State joined: boolean = false;
  @State joinAnimating: boolean = false;
  private joinAnimTimer: number = -1;

  aboutToAppear(): void {
    SystemUiController.syncWithCurrentTheme();
  }

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBackground(): string {
    return this.getCurrentThemePalette().surfaceBackground;
  }

  private getSurfaceSubtle(): string {
    return this.getCurrentThemePalette().surfaceSubtle;
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private getCurrentEvent(): EventItem {
    const ev = MOCK_EVENTS.find((e) => e.id === this.eventId);
    return ev ?? MOCK_EVENTS[0];
  }

  private toggleJoin(): void {
    this.joinAnimating = true;
    this.joined = !this.joined;
    SystemToast.show(this.joined ? '已报名参加活动' : '已取消报名', this.getUIContext());
    this.joinAnimTimer = setTimeout(() => {
      this.joinAnimating = false;
      this.joinAnimTimer = -1;
    }, 300);
  }

  private getFillRate(): number {
    const ev = this.getCurrentEvent();
    return Math.round((ev.participantsCount / ev.maxParticipants) * 100);
  }

  aboutToDisappear(): void {
    if (this.joinAnimTimer !== -1) {
      clearTimeout(this.joinAnimTimer);
      this.joinAnimTimer = -1;
    }
  }

  build() {
    NavDestination() {
      Column() {
        // 可滚动内容区
        Scroll() {
          Column({ space: 0 }) {
            this.buildCoverSection()
            this.buildContentSection()
          }
          .width('100%')
        }
        .layoutWeight(1)
        .width('100%')
        .scrollBar(BarState.Auto)

        // 底部固定按钮
        this.buildBottomCta()
      }
      .width('100%')
      .height('100%')
      .backgroundColor(this.getCurrentThemePalette().pageBackground)
    }
    .hideTitleBar(true)
    .onReady((context: NavDestinationContext) => {
      this.pathStack = context.pathStack;
      const params = context.pathInfo?.param as Record<string, Object> | undefined;
      if (params && params.eventId !== undefined) {
        this.eventId = params.eventId as number;
      }
    })
  }

  @Builder
  private buildCoverSection() {
    Stack() {
      // 封面占位（渐变色模拟图片）
      Column()
        .width('100%')
        .height('100%')
        .linearGradient({
          direction: GradientDirection.Bottom,
          colors: [
            [this.getCurrentEvent().coverGradient[0], 0.0],
            [this.getCurrentEvent().coverGradient[1], 1.0]
          ]
        })

      // 底部渐变遮罩
      Column()
        .width('100%')
        .height('40%')
        .linearGradient({
          direction: GradientDirection.Bottom,
          colors: [
            ['rgba(0,0,0,0.0)', 0.0],
            ['rgba(0,0,0,0.45)', 1.0]
          ]
        })
        .align(Alignment.Bottom)

      // 顶部透明导航按钮
      Row({ space: DesignConstants.SPACING.SM }) {
        // 返回按钮
        Button() {
          Text() {
            SymbolSpan($r(AppIcon.BACK))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([this.getCurrentThemePalette().accentText])
          }
        }
        .type(ButtonType.Normal)
        .width(36)
        .height(36)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(DesignConstants.COLOR.OVERLAY_LIGHT)
        .onClick(() => {
          AppRouter.pop();
        })

        // 分享按钮
        Button() {
          Text() {
            SymbolSpan($r(AppIcon.SHARE))
              .fontSize(DesignConstants.FONT.SIZE.MD)
              .fontColor([this.getCurrentThemePalette().accentText])
          }
        }
        .type(ButtonType.Normal)
        .width(36)
        .height(36)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(DesignConstants.COLOR.OVERLAY_LIGHT)
        .onClick(() => {
          SystemToast.show('分享功能开发中', this.getUIContext());
        })
      }
      .width('100%')
      .padding({
        left: DesignConstants.SPACING.LG,
        right: DesignConstants.SPACING.LG,
        top: SafeAreaUtils.resolveTopPadding(this.safeAreaTop)
      })
      .align(Alignment.Top)

      // 标签徽章
      Row() {
        Text(this.getCurrentEvent().tag)
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().accentText)
          .fontWeight(FontWeight.Medium)
          .padding({ left: 10, right: 10, top: 4, bottom: 4 })
          .backgroundColor(this.getCurrentEvent().tagColor)
          .borderRadius(DesignConstants.SHAPE.CHIP)
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG })
      .align(Alignment.BottomStart)
      .margin({ bottom: DesignConstants.SPACING.MD })
    }
    .width('100%')
    .height(220)
  }

  @Builder
  private buildContentSection() {
    Column({ space: DesignConstants.SPACING.LG }) {
      // 标题
      Text(this.getCurrentEvent().title)
        .fontSize(DesignConstants.FONT.SIZE.XXL)
        .fontWeight(FontWeight.Bold)
        .fontColor(this.getCurrentThemePalette().textPrimary)
        .width('100%')

      // 元信息（日期、时间、地点）
      Column({ space: DesignConstants.SPACING.SM }) {
        this.buildMetaRow(AppIcon.TIME, this.getCurrentEvent().date)
        this.buildMetaRow(AppIcon.TIME, this.getCurrentEvent().time)
        this.buildMetaRow(AppIcon.LOCATION, this.getCurrentEvent().location)
      }
      .width('100%')

      // 主办方卡片
      this.buildHostCard()

      // 活动描述
      Text(this.getCurrentEvent().description)
        .fontSize(DesignConstants.FONT.SIZE.MD)
        .fontColor(this.getCurrentThemePalette().textSecondary)
        .lineHeight(24)
        .width('100%')

      // 亮点/特色标签
      Flex({ wrap: FlexWrap.Wrap }) {
        ForEach(this.getCurrentEvent().highlights, (item: string, index: number) => {
          Row({ space: DesignConstants.SPACING.XS }) {
            Text('✓')
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentEvent().tagColor)
              .fontWeight(FontWeight.Bold)

            Text(item)
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor(this.getCurrentThemePalette().textPrimary)
          }
          .padding({ left: 12, right: 14, top: 8, bottom: 8 })
          .backgroundColor(this.getSurfaceBackground())
          .borderRadius(DesignConstants.SHAPE.CHIP)
          .border({ width: 1, color: this.getSurfaceBorder() })
          .margin({ right: DesignConstants.SPACING.SM, bottom: DesignConstants.SPACING.SM })
        }, (item: string, index: number) => `${item}_${index}`)
      }
      .width('100%')

      // 参与者版块
      this.buildParticipantsSection()

      // 距离信息
      Row({ space: DesignConstants.SPACING.SM }) {
        Text() {
          SymbolSpan($r(AppIcon.LOCATION))
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor([this.getCurrentEvent().tagColor])
        }

        Text(`距你 ${this.getCurrentEvent().distance}`)
          .fontSize(DesignConstants.CHAT.BUBBLE_FONT_SIZE)
          .fontColor(this.getCurrentThemePalette().textSecondary)
      }
      .width('100%')
      .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG, top: 10, bottom: 10 })
      .backgroundColor(this.getCurrentEvent().tagColor + '15')
      .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
      .border({ width: 1, color: this.getCurrentEvent().tagColor + '30' })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XXL
    })
  }

  @Builder
  private buildMetaRow(icon: AppIcon, text: string) {
    Row({ space: DesignConstants.SPACING.SM }) {
      Row() {
        Text() {
          SymbolSpan($r(icon))
            .fontSize(DesignConstants.CHAT.BUBBLE_FONT_SIZE)
            .fontColor([this.getCurrentEvent().tagColor])
        }
      }
      .width(28)
      .height(28)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getCurrentEvent().tagColor + '18')
      .justifyContent(FlexAlign.Center)

      Text(text)
        .fontSize(DesignConstants.CHAT.BUBBLE_FONT_SIZE)
        .fontColor(this.getCurrentThemePalette().textSecondary)
    }
    .width('100%')
    .alignItems(VerticalAlign.Center)
  }

  @Builder
  private buildHostCard() {
    Row({ space: DesignConstants.SPACING.MD }) {
      SystemAvatar({
        name: this.getCurrentEvent().hostName,
        avatarSize: 40,
        shape: 'circle'
      })

      Column({ space: 2 }) {
        Text(this.getCurrentEvent().hostName)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .fontWeight(FontWeight.Medium)

        Row({ space: 4 }) {
          Text() {
            SymbolSpan($r(AppIcon.STAR))
              .fontSize(DesignConstants.FONT.SIZE.SM)
              .fontColor([this.getCurrentThemePalette().warning])
          }
          Text('活动发起人')
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentThemePalette().textSecondary)
        }
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      Button('关注')
        .type(ButtonType.Normal)
        .backgroundColor(this.getCurrentEvent().tagColor)
        .fontColor(this.getCurrentThemePalette().accentText)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .height(30)
        .borderRadius(DesignConstants.SHAPE.BUTTON)
        .padding({ left: 12, right: 12 })
    }
    .width('100%')
    .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD, top: 12, bottom: 12 })
    .backgroundColor(this.getSurfaceBackground())
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
    .border({ width: 1, color: this.getSurfaceBorder() })
  }

  @Builder
  private buildParticipantsSection() {
    Column({ space: DesignConstants.SPACING.SM }) {
      Row() {
        Text(`参与者 · ${this.getCurrentEvent().participantsCount}/${this.getCurrentEvent().maxParticipants}`)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .layoutWeight(1)

        Text('查看全部')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().accentBackground)
      }
      .width('100%')

      // 进度条
      Column()
        .width('100%')
        .height(6)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getSurfaceBorder())

      Stack() {
        // 填充条
        Column()
          .width(this.getFillRate().toString() + '%')
          .height(6)
          .borderRadius(DesignConstants.RADIUS.FULL)
          .backgroundColor(this.getCurrentEvent().tagColor)
          .align(Alignment.Start)
      }
      .width('100%')
      .height(6)
      .margin({ top: -6 })

      // 头像行
      Row({ space: 0 }) {
        // 用颜色块模拟参与者头像
        ForEach([0, 1, 2], (i: number) => {
          Row() {
            Column()
              .width(32)
              .height(32)
              .borderRadius(DesignConstants.RADIUS.FULL)
              .backgroundColor(this.getSurfaceSubtle())
              .border({ width: 2, color: this.getSurfaceBackground() })
          }
          .margin({ left: i > 0 ? -8 : 0 })
          .zIndex(10 - i)
        })

        // 剩余名额
        Row() {
          Text('+' + (this.getCurrentEvent().maxParticipants - this.getCurrentEvent().participantsCount).toString())
            .fontSize(DesignConstants.FONT.SIZE.XS)
            .fontColor(this.getCurrentEvent().tagColor)
            .fontWeight(FontWeight.Bold)
        }
        .width(32)
        .height(32)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.getCurrentEvent().tagColor + '25')
        .border({ width: 2, color: this.getSurfaceBackground() })
        .margin({ left: -8 })
        .justifyContent(FlexAlign.Center)

        Text('已报名')
          .fontSize(DesignConstants.FONT.SIZE.SM)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .margin({ left: DesignConstants.SPACING.SM })
      }
      .width('100%')
      .alignItems(VerticalAlign.Center)
    }
    .width('100%')
  }

  @Builder
  private buildBottomCta() {
    Column() {
      Button() {
        Row({ space: DesignConstants.SPACING.SM }) {
          Text() {
            SymbolSpan($r(this.joined ? AppIcon.CHECK : AppIcon.ADD))
              .fontSize(DesignConstants.FONT.SIZE.LG)
              .fontColor([this.joined ? this.getCurrentEvent().tagColor : this.getCurrentThemePalette().accentText])
          }

          Text(this.joined ? '已报名 · 点击取消' : '立即报名参加')
            .fontSize(DesignConstants.FONT.SIZE.LG)
            .fontColor(this.joined ? this.getCurrentEvent().tagColor : this.getCurrentThemePalette().accentText)
            .fontWeight(FontWeight.Medium)
        }
      }
      .type(ButtonType.Normal)
      .width('100%')
      .height(52)
      .borderRadius(DesignConstants.SHAPE.BUTTON)
      .backgroundColor(this.joined ? this.getSurfaceBackground() : this.getCurrentEvent().tagColor)
      .border({ width: this.joined ? 2 : 0, color: this.getCurrentEvent().tagColor })
      .shadow(this.getCurrentThemePalette().accentShadow)
      .onClick(() => this.toggleJoin())
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG,
      right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD,
      bottom: SafeAreaUtils.resolveBottomPadding(this.safeAreaBottom, DesignConstants.SPACING.MD, 0)
    })
    .backgroundColor(this.getSurfaceBackground())
    .border({ width: { top: 1 }, color: this.getSurfaceBorder() })
  }
}

@Builder
export function EventDetailPageBuilder() {
  EventDetailPage()
}
