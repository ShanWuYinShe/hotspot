/**
 * NearbyPage — 附近门户页（对齐前端 /nearby 顶级导航 tab）
 *
 * 结构：搜索入口 + 三分区入口卡（附近的人 / 附近群组 / 附近动态）。
 * 内容流由各专区页承担（首页为大纲页），本页不拉取数据，
 * 避免与首页重复展示同一组附近的人/群组/动态（对齐前端信息架构）。
 */

import { SystemPageHeader } from '../../../components/SystemPageHeader';
import { SurfaceIconButton } from '../../../components/SurfaceIconButton';
import { LucideIcon } from '../../../components/LucideIcon';
import { curves } from '@kit.ArkUI';
import { PageShellContainer } from '../../../components/PageShellContainer';
import type { BreakpointType } from '../../../core/responsive/Breakpoint';
import { AppRouter } from '../../../core/router/AppRouter';
import { PageNames } from '../../../constants/AppConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { KEY_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

interface NearbySectionEntry {
  key: string;
  icon: AppIcon;
  title: string;
  description: string;
}

@Preview
@Component
export struct NearbyPage {
  @StorageProp('currentBreakpoint') currentBreakpoint: BreakpointType = 'sm';
  @StorageLink(KEY_RESOLVED_THEME_ID) resolvedThemeId: string = 'vibrant_light';
  @State private searchEntryPressed: boolean = false;
  @State private pressedSectionKey: string = '';

  private static readonly SECTION_ENTRIES: NearbySectionEntry[] = [
    { key: 'people', icon: AppIcon.PERSON, title: '附近的人', description: '认识 5km 内的新朋友' },
    { key: 'groups', icon: AppIcon.GROUP, title: '附近群组', description: '加入身边的兴趣小组' },
    { key: 'moments', icon: AppIcon.SPARKLES, title: '附近动态', description: '看看周边正在发生什么' }
  ];

  private getCurrentThemePalette() {
    return AppThemeManager.getPaletteById(this.resolvedThemeId);
  }

  private getSurfaceBorder(): string {
    return this.getCurrentThemePalette().borderColor;
  }

  private routeForSection(key: string): string {
    if (key === 'people') return PageNames.NEARBY_USERS;
    if (key === 'groups') return PageNames.NEARBY_GROUPS;
    return PageNames.NEARBY_MOMENTS;
  }

  build() {
    PageShellContainer({ variant: 'default', currentBreakpoint: this.currentBreakpoint, noHorizontalPadding: true }) {
      Column() {
        SystemPageHeader({
          title: '附近',
          subtitle: '发现周边有趣的人和事',
          titleSize: DesignConstants.FONT.SIZE.XXL,
          titleWeight: DesignConstants.FONT.WEIGHT.MEDIUM,
          backgroundVariant: 'transparent'
        }) {
          // trailing: 地图入口（对齐前端 PageHeader trailing）
          SurfaceIconButton({
            icon: AppIcon.MAP,
            kind: 'neutral',
            onPress: () => AppRouter.push(PageNames.LOCATION_PICKER)
          })
        }

        Scroll() {
          Column() {
            // 搜索入口（门户第一入口，对齐前端 shadow-raised-md 搜索卡片）
            Row({ space: DesignConstants.SPACING.MD2 }) {
              LucideIcon({
                icon: AppIcon.SEARCH,
                iconSize: DesignConstants.FONT.SIZE.LG,
                paletteRole: 'accent'
              })
              Text('搜索用户、群组、动态、地点…')
                .fontSize(DesignConstants.FONT.SIZE.SM)
                .fontColor(this.getCurrentThemePalette().textSecondary)
                .layoutWeight(1)
                .maxLines(1)
                .textOverflow({ overflow: TextOverflow.Ellipsis })
            }
            .height(DesignConstants.COMPONENT.INPUT_HEIGHT)
            .padding({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
            .margin({ left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG })
            .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
            .borderRadius(DesignConstants.SHAPE.INPUT)
            .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
            .shadow(this.getCurrentThemePalette().surfaceShadow)
            // 按压反馈（对齐 tap-97 语义）
            .scale(this.searchEntryPressed ?
              { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
              { x: 1, y: 1 })
            .animation({ curve: curves.springMotion(
              DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
              DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
            ) })
            .onTouch((event: TouchEvent) => {
              if (event.type === TouchType.Down) {
                this.searchEntryPressed = true;
              } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
                this.searchEntryPressed = false;
              }
            })
            .onClick(() => { AppRouter.push(PageNames.SEARCH); })

            // 三分区入口卡（内容在各专区页，这里只做导航门户）
            Column({ space: DesignConstants.SPACING.MD }) {
              ForEach(NearbyPage.SECTION_ENTRIES, (entry: NearbySectionEntry) => {
                this.buildSectionCard(entry)
              }, (entry: NearbySectionEntry) => entry.key)
            }
            .width('100%')
            .padding({
              left: DesignConstants.SPACING.LG,
              right: DesignConstants.SPACING.LG,
              top: DesignConstants.SPACING.XL
            })

            Row()
              .height(DesignConstants.SPACING.XXL)
          }
          .width('100%')
        }
        .width('100%')
        .layoutWeight(1)
        .scrollBar(BarState.Off)
        .edgeEffect(EdgeEffect.Spring)
        .align(Alignment.Top)
      }
      .width('100%')
      .height('100%')
    }
  }

  @Builder
  private buildSectionCard(entry: NearbySectionEntry) {
    Row({ space: DesignConstants.SPACING.MD3 }) {
      // 图标徽章圆 tile（对齐前端 icon-badge：surface 底 + hairline 描边 + raised-sm 阴影）
      Column() {
        LucideIcon({
          icon: entry.icon,
          iconSize: DesignConstants.FONT.SIZE.LG,
          paletteRole: 'accent'
        })
      }
      .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
      .shadow(this.getCurrentThemePalette().surfaceShadow)
      .justifyContent(FlexAlign.Center)

      Column({ space: DesignConstants.SPACING.HALF }) {
        Text(entry.title)
          .fontSize(DesignConstants.FONT.SIZE.MD)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(this.getCurrentThemePalette().textPrimary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
        Text(entry.description)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(this.getCurrentThemePalette().textSecondary)
          .maxLines(1)
          .textOverflow({ overflow: TextOverflow.Ellipsis })
      }
      .alignItems(HorizontalAlign.Start)
      .layoutWeight(1)

      LucideIcon({
        icon: AppIcon.ARROW_RIGHT,
        iconSize: DesignConstants.FONT.SIZE.MD,
        paletteRole: 'textSecondary'
      })
    }
    .width('100%')
    .padding(DesignConstants.SPACING.LG)
    .backgroundColor(this.getCurrentThemePalette().surfaceBackground)
    .borderRadius(DesignConstants.SHAPE.CARD)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.getSurfaceBorder() })
    .shadow(this.getCurrentThemePalette().surfaceShadow)
    // 整卡按压反馈（对齐 tap-97 语义）
    .scale(this.pressedSectionKey === entry.key ?
      { x: DesignConstants.INTERACTION.SCALE_PRESSED_CARD, y: DesignConstants.INTERACTION.SCALE_PRESSED_CARD } :
      { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressedSectionKey = entry.key;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressedSectionKey = '';
      }
    })
    .onClick(() => { AppRouter.push(this.routeForSection(entry.key)); })
  }
}

@Builder
export function NearbyPageBuilder() {
  NearbyPage()
}
