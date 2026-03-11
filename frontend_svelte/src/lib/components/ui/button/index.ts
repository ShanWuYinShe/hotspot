import Root from './button.svelte';

import { tv, type VariantProps } from 'tailwind-variants';
import { cn } from '$lib/utils';

// Game UI - Neubrutalism button variants with hard shadows
export const buttonVariants = tv({
	base: cn(
		'inline-flex items-center justify-center gap-2',
		'whitespace-nowrap font-semibold',
		'rounded-xl border-[2px]',
		'transition-all duration-150 ease-[cubic-bezier(0.4,0,0.2,1)]',
		'focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/40 focus-visible:ring-offset-2',
		'disabled:pointer-events-none disabled:opacity-50',
		'cursor-pointer',
		'[&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0'
	),
	variants: {
		variant: {
			default: cn(
				// Primary button - purple theme
				'bg-primary border-primary text-primary-foreground',
				// Hard shadow using current color
				'shadow-[3px_3px_0px_currentColor]',
				// Hover - move up-left, increase shadow
				'hover:-translate-x-[1px] hover:-translate-y-[1px]',
				'hover:shadow-[4px_4px_0px_currentColor]',
				// Active - press down effect
				'active:translate-x-[2px] active:translate-y-[2px]',
				'active:shadow-[1px_1px_0px_currentColor]',
				// Glow effect in dark mode
				'dark:shadow-[3px_3px_0px_currentColor,0_0_20px_theme(colors.glow-primary)]',
				'dark:hover:shadow-[4px_4px_0px_currentColor,0_0_30px_theme(colors.glow-primary)]'
			),
			accent: cn(
				// Accent button - pink theme
				'bg-accent border-accent text-accent-foreground',
				'shadow-[3px_3px_0px_currentColor]',
				'hover:-translate-x-[1px] hover:-translate-y-[1px]',
				'hover:shadow-[4px_4px_0px_currentColor]',
				'active:translate-x-[2px] active:translate-y-[2px]',
				'active:shadow-[1px_1px_0px_currentColor]',
				'dark:shadow-[3px_3px_0px_currentColor,0_0_20px_theme(colors.glow-accent)]',
				'dark:hover:shadow-[4px_4px_0px_currentColor,0_0_30px_theme(colors.glow-accent)]'
			),
			destructive: cn(
				// Destructive button - red theme
				'bg-destructive border-destructive text-destructive-foreground',
				'shadow-[3px_3px_0px_currentColor]',
				'hover:-translate-x-[1px] hover:-translate-y-[1px]',
				'hover:shadow-[4px_4px_0px_currentColor]',
				'active:translate-x-[2px] active:translate-y-[2px]',
				'active:shadow-[1px_1px_0px_currentColor]'
			),
			outline: cn(
				// Outline button
				'bg-background border-primary text-primary',
				'shadow-[3px_3px_0px_theme(colors.primary)]',
				'hover:-translate-x-[1px] hover:-translate-y-[1px]',
				'hover:bg-primary/5',
				'hover:shadow-[4px_4px_0px_theme(colors.primary)]',
				'active:translate-x-[2px] active:translate-y-[2px]',
				'active:shadow-[1px_1px_0px_theme(colors.primary)]'
			),
			secondary: cn(
				// Secondary button - blue theme
				'bg-secondary border-secondary text-secondary-foreground',
				'shadow-[3px_3px_0px_currentColor]',
				'hover:-translate-x-[1px] hover:-translate-y-[1px]',
				'hover:shadow-[4px_4px_0px_currentColor]',
				'active:translate-x-[2px] active:translate-y-[2px]',
				'active:shadow-[1px_1px_0px_currentColor]'
			),
			ghost: cn(
				// Ghost button - 仅轻微背景高亮，避免按下时出现透明感
				'border-transparent text-foreground',
				'hover:bg-muted/60 hover:text-foreground',
				'active:bg-muted/80'
			),
			link: cn(
				// Link button - no border/shadow
				'border-transparent',
				'text-primary underline-offset-4',
				'hover:underline'
			)
		},
		size: {
			default: 'h-11 px-6 py-3 text-sm',
			sm: 'h-9 px-4 py-2 text-xs',
			lg: 'h-13 px-8 py-4 text-base rounded-2xl',
			icon: 'h-11 w-11'
		}
	},
	defaultVariants: {
		variant: 'default',
		size: 'default'
	}
});

export type Variant = VariantProps<typeof buttonVariants>['variant'];
export type Size = VariantProps<typeof buttonVariants>['size'];

export {
	Root,
	Root as Button
};
