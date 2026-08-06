// Vite raw 导入类型声明（SVG 以字符串内联，用于 currentColor 主题化染色）。
declare module '*.svg?raw' {
	const content: string;
	export default content;
}
