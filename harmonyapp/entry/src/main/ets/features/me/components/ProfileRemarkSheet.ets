/**
 * ProfileRemarkSheet —— 备注编辑半模态（对齐前端备注名称输入，仅已关注可保存）。
 * 从 ProfilePage.buildRemarkSheet 抽取；草稿/保存态经 props 注入，动作回调上抛；
 * 主题按 resolvedThemeId 自解析（与 ProfileContentAtoms 同规）；无本地状态。
 */

import { DesignConstants } from '../../../constants/DesignConstants';
import { SystemInput } from '../../../components/SystemInput';
import { SystemButton } from '../../../components/SystemButton';
import { AppThemeManager } from '../../../core/theme/AppThemeManager';
import { DEFAULT_RESOLVED_THEME_ID } from '../../../core/theme/AppThemeTypes';

// 备注长度上限（对齐前端 remarkDraft.slice(0, 24)）
const REMARK_MAX: number = 24;

@Component
export struct ProfileRemarkSheet {
  @Prop resolvedThemeId: string = DEFAULT_RESOLVED_THEME_ID;
  @Prop nickname: string = '';
  @Prop draft: string = '';
  @Prop isSaving: boolean = false;
  @Prop canSave: boolean = false;
  onDraftChange: (value: string) => void = () => {};
  onCancel: () => void = () => {};
  onSave: () => void = () => {};

  build() {
    Column({ space: DesignConstants.SPACING.MD }) {
      Text('设置备注')
        .fontSize(DesignConstants.FONT.SIZE.XL)
        .fontWeight(DesignConstants.FONT.WEIGHT.BOLD)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textPrimary)

      Text(`备注${this.nickname}，只对你可见`)
        .fontSize(DesignConstants.FONT.SIZE.SM)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textSecondary)

      SystemInput({
        value: this.draft,
        placeholder: '备注名称',
        fontSize: DesignConstants.FONT.SIZE.MD,
        inputHeight: DesignConstants.COMPONENT.BUTTON_HEIGHT,
        maxLength: REMARK_MAX,
        onChange: (value: string) => { this.onDraftChange(value); }
      })
        .backgroundColor(AppThemeManager.getPaletteById(this.resolvedThemeId).surfaceSubtle)
        .shadow(AppThemeManager.getPaletteById(this.resolvedThemeId).controlShadow)
        .borderRadius(DesignConstants.RADIUS.MD)
        .padding({ left: DesignConstants.SPACING.MD, right: DesignConstants.SPACING.MD })

      Text(`${this.draft.trim().length}/${REMARK_MAX}`)
        .fontSize(DesignConstants.FONT.SIZE.XS)
        .fontColor(AppThemeManager.getPaletteById(this.resolvedThemeId).textTertiary)
        .width('100%')
        .textAlign(TextAlign.End)

      Row({ space: DesignConstants.SPACING.SM }) {
        SystemButton({
          label: '取消',
          variant: 'surface',
          buttonSize: 'sheet',
          fullWidth: true,
          onPress: () => { this.onCancel(); }
        })
        SystemButton({
          label: '保存',
          variant: 'primary',
          buttonSize: 'sheet',
          fullWidth: true,
          loading: this.isSaving,
          disabled: !this.canSave,
          onPress: () => { this.onSave(); }
        })
      }
      .width('100%')
    }
    .width('100%')
    .padding({
      left: DesignConstants.SPACING.XXL,
      right: DesignConstants.SPACING.XXL,
      top: DesignConstants.SPACING.LG,
      bottom: DesignConstants.SPACING.XXL
    })
  }
}
