// Composer store — 发布器全局开关（FAB/页内 CTA 共用唯一入口）。
// ComposerContext（runes 模块单例）。

let composerOpen = $state(false);

export function openComposer(): void {
  composerOpen = true;
}

export function closeComposer(): void {
  composerOpen = false;
}

export function toggleComposer(): void {
  composerOpen = !composerOpen;
}

export const composer = {
  get composerOpen() { return composerOpen; },
};
