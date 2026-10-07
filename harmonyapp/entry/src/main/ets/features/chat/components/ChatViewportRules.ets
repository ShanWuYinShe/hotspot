/**
 * ChatViewportRules —— 聊天页视口相关纯规则（从 ChatPage 抽取）。
 *
 * 两条规则都只依赖入参，无状态、无副作用：
 * 1) 底部安全区「键盘尖峰」抑制：软键盘弹出会把 safeAreaBottom 从手势条高度
 *    直接顶到键盘高度，若照单全收，输入栏会先被顶起再回落造成跳动；这里在
 *    超过阈值且增幅明显时沿用上一稳定值。
 * 2) 是否「接近底部」：决定新消息到达时是否自动滚动 / 是否显示回到底部浮标。
 */

/** 视为键盘顶起的下界（vp）：低于此值只可能是手势条/系统栏，不是键盘 */
export const SAFE_AREA_KEYBOARD_SPIKE_THRESHOLD: number = 120;
/** 相对上一稳定值的最小增幅，才算键盘尖峰（避免误判小幅系统变化） */
export const SAFE_AREA_SPIKE_DELTA: number = 48;
/** 距离末尾多少条以内算「接近底部」 */
export const NEAR_BOTTOM_TRAIL_COUNT: number = 3;

/**
 * 计算应采用的稳定底部安全区：
 * - 首次（上一稳定值 <= 0）直接采纳当前值；
 * - 非键盘尖峰：采纳当前值；
 * - 键盘尖峰（当前值 >= 阈值 且 比上一稳定值高出 DELTA 以上）：沿用上一稳定值，抑制跳动。
 * 返回值为「本次应采用的安全区」，调用方应同时把上一稳定值更新为该返回值。
 */
export function resolveStableBottomInset(currentInset: number, stableInset: number): number {
  const clamped = Math.max(0, currentInset);
  if (stableInset <= 0) {
    return clamped;
  }
  const isKeyboardSpike = clamped >= SAFE_AREA_KEYBOARD_SPIKE_THRESHOLD &&
    clamped > stableInset + SAFE_AREA_SPIKE_DELTA;
  if (!isKeyboardSpike) {
    return clamped;
  }
  return stableInset;
}

/** 消息列表是否处于「接近底部」状态（空列表视为在底部） */
export function isNearBottom(messageCount: number, firstVisibleIndex: number): boolean {
  return messageCount === 0 || firstVisibleIndex >= messageCount - NEAR_BOTTOM_TRAIL_COUNT;
}
