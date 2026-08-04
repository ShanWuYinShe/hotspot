// 项目 UI 封装层：Dialog（含 Sheet 定位变体）。
// 行为来自 bits-ui，视觉来自项目 token（--surface-* / --z-modal 等）。
export { default as Root } from './root.svelte';
export { default as Content } from './content.svelte';
export { default as Title } from './title.svelte';
export { default as Description } from './description.svelte';
export { default as Close } from './close.svelte';
export type { DialogSide } from './content.svelte';
