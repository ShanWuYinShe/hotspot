import type { Component } from 'svelte';

/** 分区筛选 pill 定义（ProfileUserSection 的 filters prop 元素类型）。 */
export type SectionFilter = { id: string; label: string; icon?: Component };
