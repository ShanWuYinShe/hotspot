import { getContext, setContext } from 'svelte';

const LIST_CONTEXT_KEY = Symbol('list-container');

type ListContext = { island: boolean };

/**
 * 列表容器（ListContainer）在初始化时调用：向子树声明「当前处于列表容器内」。
 * 子组件据此自动补足 listitem 语义（axe aria-required-children：role=list 的
 * 直接子元素必须为 listitem），调用方无需任何额外 props。
 *
 * Round 3：额外携带 island 标记——bordered 软岛容器的行必须透明（bg-transparent），
 * 让软岛表面/阴影透出；否则行自带的不透明 surface-primary 会把软岛盖住（暗色下
 * 岛与画布完全糊平，实测 ratio 1.01）。
 */
export function setInListContext(island = false): void {
	setContext(LIST_CONTEXT_KEY, { island } satisfies ListContext);
}

function getListContext(): ListContext | null {
	return getContext<ListContext | null>(LIST_CONTEXT_KEY);
}

/** 行/哨兵组件在初始化时调用：是否处于 ListContainer 内。 */
export function isInListContext(): boolean {
	// getContext 未设置时返回 undefined（非 null），必须用 != null
	return getListContext() != null;
}

/** 行是否处于 bordered 软岛容器内（行背景应透明，露出软岛表面）。 */
export function isInIslandContext(): boolean {
	return getListContext()?.island === true;
}
