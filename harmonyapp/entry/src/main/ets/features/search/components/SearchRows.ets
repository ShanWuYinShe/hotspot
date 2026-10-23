/**
 * SearchRows —— 搜索页两个自包含行组件：
 * SearchSuggestRow（联想建议行）/ SearchMomentRow（全部结果中的动态行）。
 * 从 SearchPage 的 buildSuggestRow/buildAllMomentRow 抽取；
 * 文案与动作经 props/回调注入，主题对象由页面传入。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { Pressable } from '../../../components/Pressable';
import { AppThemePalette } from '../../../core/theme/AppThemeTypes';
import { MomentItem } from '../../../core/types/api';
import { cleanPlainText } from '../../../core/utils/FormatUtil';
import { LucideIcon } from '../../../components/LucideIcon';

export type SearchSuggestType = 'tag' | 'user' | 'group' | 'moment';

@Component
export struct SearchSuggestRow {
  @Prop itemType: SearchSuggestType = 'moment';
  @Prop title: string = '';
  @Prop subtitle: string = '';
  palette: AppThemePalette | null = null;
  surfaceBackground: string = '';
  surfaceSubtle: string = '';
  surfaceBorder: string | Resource = '';
  onPress: () => void = () => {};

  build() {
    Pressable({ onPress: () => { this.onPress(); } }) {
      Row({ space: DesignConstants.SPACING.MD }) {
        Row() {
          LucideIcon({
              icon: this.itemType === 'tag' ? AppIcon.CHART : this.itemType === 'user' ? AppIcon.PERSON : this.itemType === 'group' ? AppIcon.GROUP : AppIcon.CAMERA,
              iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
              color: this.palette?.accentBackground
            })
        }
        .width(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
        .height(DesignConstants.COMPONENT.AVATAR_SIZE.SMALL)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(this.surfaceSubtle)
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)

        Column({ space: DesignConstants.SPACING.XS }) {
          Text(this.title)
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
            .fontColor(this.palette?.textPrimary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
          Text(this.subtitle)
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .fontColor(this.palette?.textSecondary)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
        }
        .layoutWeight(1)
        .alignItems(HorizontalAlign.Start)

        LucideIcon({
            icon: AppIcon.ARROW_RIGHT,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            color: this.palette?.textSecondary
          })
      }
      .width('100%')
      .padding(DesignConstants.SPACING.MD)
      .backgroundColor(this.surfaceBackground)
      .borderRadius(DesignConstants.SHAPE.CARD)
      .shadow(this.palette?.surfaceShadow)
      .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.surfaceBorder })
    }
  }
}

@Component
export struct SearchMomentRow {
  @Prop moment: MomentItem;
  palette: AppThemePalette | null = null;
  surfaceBackground: string = '';
  surfaceBorder: string | Resource = '';
  /** true = 自带卡片外框（独立使用）；false = 无框行（嵌在分组卡里，对齐前端
   *  SearchResults 的「一组一张卡 + 组内行无框」结构） */
  @Prop framed: boolean = true;
  onPress: () => void = () => {};

  build() {
    Pressable({ onPress: () => { this.onPress(); } }) {
      Row() {
        Column({ space: DesignConstants.SPACING.XS }) {
          Text(cleanPlainText(this.moment.description))
            .fontSize(DesignConstants.FONT.SIZE.MD)
            .fontColor(this.palette?.textPrimary)
            .maxLines(2)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .width('100%')
          Text(this.moment.location?.location_name ?? '')
            .fontSize(DesignConstants.FONT.SIZE.SM)
            .maxLines(1)
            .textOverflow({ overflow: TextOverflow.Ellipsis })
            .fontColor(this.palette?.textSecondary)
            .width('100%')
        }
        .alignItems(HorizontalAlign.Start)
        .layoutWeight(1)

        LucideIcon({
            icon: AppIcon.ARROW_RIGHT,
            iconSize: DesignConstants.COMPONENT.ICON_SIZE_XS,
            color: this.palette?.textSecondary
          })
      }
      .width('100%')
      .padding(this.framed ? DesignConstants.SPACING.MD : DesignConstants.SPACING.MD3)
      .backgroundColor(this.framed ? this.surfaceBackground : DesignConstants.COLOR.TRANSPARENT)
      .borderRadius(this.framed ? DesignConstants.SHAPE.CARD : 0)
      .shadow(this.framed ? this.palette?.surfaceShadow : DesignConstants.SHADOW.SMALL)
      .border(this.framed ? { width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: this.surfaceBorder } : { width: 0 })
    }
  }
}
