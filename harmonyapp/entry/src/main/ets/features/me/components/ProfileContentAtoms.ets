/**
 * ProfileContentAtoms —— 个人页四个参数化内容原子：
 * CompactStatItem（紧凑统计项）/ ProfileQuickAction（快捷操作）/
 * ProfileIntentCard（意图卡）/ ProfileMenuItem（菜单行）。
 * 从 ProfilePage 的同名 @Builder 抽取；主题经 resolvedThemeId 自解析
 * （AppThemeVisuals + AppThemeManager），动作回调上抛，按压态组件自持。
 */

import { curves } from '@kit.ArkUI';
import { DesignConstants } from '../../../constants/DesignConstants';
import { AppIcon } from '../../../components/SystemIcon';
import { LucideIcon } from '../../../components/LucideIcon';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { AppThemeVisuals } from '../../../core/theme/AppThemeVisuals';

@Component
export struct CompactStatItem {
  @Prop icon: AppIcon;
  @Prop value: string = '';
  @Prop label: string = '';
  @Prop resolvedThemeId: string = 'vibrant_light';
  onTap: () => void = () => {};

  build() {
    Button() {
      Column({ space: DesignConstants.SPACING.XS }) {
        LucideIcon({
          icon: this.icon,
          size: DesignConstants.FONT.SIZE.LG,
          color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground,
          overrideThemeId: this.resolvedThemeId
        })
        Text(this.value)
          .fontSize(DesignConstants.FONT.SIZE.LG)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
        Text(this.label)
          .fontSize(DesignConstants.FONT.SIZE.XS)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
      }
      .width('100%')
      .alignItems(HorizontalAlign.Center)
      .justifyContent(FlexAlign.Center)
    }
    .type(ButtonType.Normal)
    .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceBackground)
    .border({ width: DesignConstants.BORDER_WIDTH.HAIRLINE, color: AppThemeManager.getPaletteById(this.resolvedThemeId).borderColor })
    .borderRadius(DesignConstants.SHAPE.COMPACT_CARD)
    .shadow(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).card.shadow)
    .padding({ top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD })
    .layoutWeight(1)
    .stateEffect(true)
    .onClick(() => { this.onTap(); })
  }
}

@Component
export struct ProfileQuickAction {
  @Prop icon: AppIcon;
  @Prop label: string = '';
  @Prop resolvedThemeId: string = 'vibrant_light';
  onTap: () => void = () => {};

  build() {
    Button() {
      Column({ space: DesignConstants.SPACING.SM }) {
        Row() {
          LucideIcon({
            icon: this.icon,
            size: AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.iconSize,
            color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground,
            overrideThemeId: this.resolvedThemeId
          })
        }
        .width(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.iconWrapperSize)
        .height(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.iconWrapperSize)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceSubtle)
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)

        Text(this.label)
          .fontSize(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.labelFontSize)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
      }
      .width('100%')
      .alignItems(HorizontalAlign.Center)
    }
    .type(ButtonType.Normal)
    .backgroundColor(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.backgroundColor)
    .borderRadius(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.borderRadius)
    .border({ width: AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.borderWidth, color: AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.borderColor })
    .shadow(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.shadow)
    .padding(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).quickAction.padding)
    .layoutWeight(1)
    .onClick(() => { this.onTap(); })
  }
}

@Component
export struct ProfileIntentCard {
  @Prop title: string = '';
  @Prop description: string = '';
  @Prop icon: AppIcon;
  @Prop resolvedThemeId: string = 'vibrant_light';
  onTap: () => void = () => {};

  build() {
    Button() {
      Column({ space: DesignConstants.SPACING.MD }) {
        Row() {
          LucideIcon({
            icon: this.icon,
            size: DesignConstants.FONT.SIZE.XL,
            color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground,
            overrideThemeId: this.resolvedThemeId
          })
        }
        .width(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.iconWrapperSize)
        .height(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.iconWrapperSize)
        .borderRadius(DesignConstants.RADIUS.FULL)
        .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceSubtle)
        .justifyContent(FlexAlign.Center)
        .alignItems(VerticalAlign.Center)

        Text(this.title)
          .fontSize(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.titleFontSize)
          .fontWeight(DesignConstants.FONT.WEIGHT.MEDIUM)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
          .width('100%')
          .textAlign(TextAlign.Start)

        Text(this.description)
          .fontSize(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.descriptionFontSize)
          .lineHeight(DesignConstants.FONT.LINE_HEIGHT.LG)
          .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)
          .width('100%')
          .textAlign(TextAlign.Start)
      }
      .width('100%')
      .alignItems(HorizontalAlign.Start)
    }
    .type(ButtonType.Normal)
    .layoutWeight(1)
    .backgroundColor(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.backgroundColor)
    .borderRadius(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.borderRadius)
    .shadow(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.shadow)
    .border({ width: AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.borderWidth, color: AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.borderColor })
    .padding(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.padding)
    .height(AppThemeVisuals.getProfileVisualStyle(this.resolvedThemeId).intentCard.minHeight)
    .onClick(() => { this.onTap(); })
  }
}

@Component
export struct ProfileMenuItem {
  @Prop title: string = '';
  @Prop icon: AppIcon;
  @Prop resolvedThemeId: string = 'vibrant_light';
  @State pressed: boolean = false;
  onTap: () => void = () => {};

  build() {
    Row({ space: DesignConstants.SPACING.MD }) {
      Row() {
        LucideIcon({
          icon: this.icon,
          size: DesignConstants.FONT.SIZE.LG,
          color: AppThemeManager.getPaletteById(this.resolvedThemeId).accentBackground,
          overrideThemeId: this.resolvedThemeId
        })
      }
      .width(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .height(DesignConstants.LAYOUT.ICON_BUTTON_SIZE)
      .borderRadius(DesignConstants.RADIUS.FULL)
      .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceSubtle)
      .justifyContent(FlexAlign.Center)
      .alignItems(VerticalAlign.Center)

      Text(this.title)
        .fontSize(DesignConstants.FONT.SIZE.LG)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)
        .layoutWeight(1)

      LucideIcon({
        icon: AppIcon.FORWARD,
        size: DesignConstants.FONT.SIZE.LG,
        color: AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary,
        overrideThemeId: this.resolvedThemeId
      })
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.LG, right: DesignConstants.SPACING.LG,
      top: DesignConstants.SPACING.MD, bottom: DesignConstants.SPACING.MD
    })
    .scale(this.pressed ? { x: 0.985, y: 0.985 } : { x: 1, y: 1 })
    .animation({ curve: curves.springMotion(
      DesignConstants.INTERACTION.SPRING_RESPONSE_PRESS,
      DesignConstants.INTERACTION.SPRING_DAMPING_PRESS
    ) })
    .onTouch((event: TouchEvent) => {
      if (event.type === TouchType.Down) {
        this.pressed = true;
      } else if (event.type === TouchType.Up || event.type === TouchType.Cancel) {
        this.pressed = false;
      }
    })
    .onClick(() => { this.onTap(); })
  }
}
