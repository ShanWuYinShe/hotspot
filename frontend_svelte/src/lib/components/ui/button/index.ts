import Root from './button.svelte';

import { tv, type VariantProps } from 'tailwind-variants';
import { cn } from '$lib/utils';

export const buttonVariants = tv({
	base: cn(
		'inline-flex items-center justify-center gap-2',
		'whitespace-nowrap font-semibold',
		'rounded-[1.35rem] border',
		'transition-all duration-180 ease-[cubic-bezier(0.4,0,0.2,1)]',
		'focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-primary/25 focus-visible:ring-offset-2',
		'disabled:pointer-events-none disabled:opacity-50',
		'cursor-pointer',
		'[&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0'
	),
	variants: {
		variant: {
			default: cn(
				'bg-primary border-primary/80 text-primary-foreground',
				'shadow-[8px_8px_18px_var(--shadow-dark),-8px_-8px_18px_var(--shadow-light),0_6px_16px_var(--glow-primary)]',
				'hover:brightness-[1.03]',
				'active:shadow-[inset_5px_5px_12px_rgba(0,0,0,0.16),inset_-5px_-5px_12px_rgba(255,255,255,0.14)]'
			),
			accent: cn(
				'bg-accent border-border text-accent-foreground',
				'shadow-[8px_8px_18px_var(--shadow-dark),-8px_-8px_18px_var(--shadow-light)]',
				'hover:bg-accent/90',
				'active:shadow-[inset_5px_5px_12px_var(--shadow-dark),inset_-5px_-5px_12px_var(--shadow-light)]'
			),
			destructive: cn(
				'bg-destructive border-destructive/80 text-destructive-foreground',
				'shadow-[8px_8px_18px_var(--shadow-dark),-8px_-8px_18px_var(--shadow-light)]',
				'hover:brightness-[1.03]',
				'active:shadow-[inset_5px_5px_12px_rgba(0,0,0,0.18),inset_-5px_-5px_12px_rgba(255,255,255,0.12)]'
			),
			outline: cn(
				'bg-card border-border text-primary',
				'shadow-[8px_8px_18px_var(--shadow-dark),-8px_-8px_18px_var(--shadow-light)]',
				'hover:bg-primary/6',
				'active:shadow-[inset_5px_5px_12px_var(--shadow-dark),inset_-5px_-5px_12px_var(--shadow-light)]'
			),
			secondary: cn(
				'bg-secondary border-border text-secondary-foreground',
				'shadow-[8px_8px_18px_var(--shadow-dark),-8px_-8px_18px_var(--shadow-light)]',
				'hover:bg-secondary/88',
				'active:shadow-[inset_5px_5px_12px_var(--shadow-dark),inset_-5px_-5px_12px_var(--shadow-light)]'
			),
			ghost: cn(
				'border-transparent bg-transparent text-foreground shadow-none',
				'hover:bg-muted/65 hover:text-foreground',
				'active:bg-muted/90'
			),
			link: cn(
				'border-transparent bg-transparent text-primary shadow-none',
				'underline-offset-4 hover:underline'
			)
		},
		size: {
			default: 'h-11 px-6 py-3 text-sm',
			sm: 'h-9 px-4 py-2 text-xs',
			lg: 'h-13 px-8 py-4 text-base rounded-[1.5rem]',
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
