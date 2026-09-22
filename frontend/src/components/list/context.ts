import { getContext, setContext } from 'svelte';

const LIST_CONTEXT_KEY = Symbol('list-container');

/**
 * 列表容器（ListContainer）在初始化时调用：向子树声明「当前处于列表容器内」。
 * 子组件据此自动补足 listitem 语义（axe aria-required-children：role=list 的
 * 直接子元素必须为 listitem），调用方无需任何额外 props。
 */
export function setInListContext(): void {
	setContext(LIST_CONTEXT_KEY, true);
}

/** 行/哨兵组件在初始化时调用：是否处于 ListContainer 内。 */
export function isInListContext(): boolean {
	return getContext<boolean>(LIST_CONTEXT_KEY) === true;
}
