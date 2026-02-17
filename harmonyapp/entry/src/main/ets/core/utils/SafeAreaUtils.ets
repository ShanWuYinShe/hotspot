import { DesignConstants } from '../../constants/DesignConstants';

export class SafeAreaUtils {
  static resolveTopPadding(safeAreaTop: number, extra: number = DesignConstants.SPACING.SM,
    fallback: number = DesignConstants.SPACING.LG): number {
    const base = Math.max(safeAreaTop, fallback);
    return base + extra;
  }

  static resolveFlexibleTop(safeAreaTop: number, fallback: number): number {
    return Math.max(safeAreaTop, fallback);
  }

  static resolveBottomPadding(safeAreaBottom: number, extra: number = 0,
    fallback: number = DesignConstants.SPACING.LG): number {
    const base = Math.max(safeAreaBottom, fallback);
    return base + extra;
  }

  static resolveFloatingBottomOffset(safeAreaBottom: number, fallback: number = DesignConstants.SPACING.XL): number {
    return Math.max(safeAreaBottom, fallback);
  }
}
