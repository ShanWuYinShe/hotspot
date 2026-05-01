import Root from './badge.svelte';
import { tv, type VariantProps } from 'tailwind-variants';
import { cn } from '$lib/utils';

export const badgeVariants = tv({
	base: cn(
		'inline-flex items-center gap-1.5',
		'rounded-2xl',
		'border-2',
		'px-3 py-1 text-xs font-semibold',
		'transition-all duration-200',
		'shadow-sm shadow-black/10',
		'dark:shadow-md dark:shadow-black/20'
	),
	variants: {
		variant: {
			default: cn(
				'bg-primary/10 border-primary/30',
				'dark:bg-primary/20 dark:border-primary/60',
				'text-primary dark:text-primary-foreground',
				'hover:bg-primary/20 hover:border-primary/40',
				'dark:hover:bg-primary/30 dark:hover:border-primary/70'
			),
			secondary: cn(
				'bg-secondary/10 border-secondary/30',
				'dark:bg-secondary/20 dark:border-secondary/60',
				'text-secondary dark:text-secondary-foreground',
				'hover:bg-secondary/20 hover:border-secondary/40',
				'dark:hover:bg-secondary/30 dark:hover:border-secondary/70'
			),
			accent: cn(
				'bg-accent/10 border-accent/30',
				'dark:bg-accent/20 dark:border-accent/70',
				'text-accent dark:neu-active-text',
				'hover:bg-accent/20 hover:border-accent/40',
				'dark:hover:bg-accent/30 dark:hover:border-accent/80'
			),
			success: cn(
				'bg-green-500/10 border-green-500/30',
				'dark:bg-green-500/20 dark:border-green-500/60',
				'text-green-600 dark:text-green-400',
				'hover:bg-green-500/20',
				'dark:hover:bg-green-500/30 dark:hover:border-green-500/70'
			),
			destructive: cn(
				'bg-destructive/10 border-destructive/30',
				'dark:bg-destructive/20 dark:border-destructive/60',
				'text-destructive dark:text-destructive-foreground',
				'hover:bg-destructive/20 hover:border-destructive/40',
				'dark:hover:bg-destructive/30 dark:hover:border-destructive/70'
			),
			outline: cn(
				'bg-white/80 dark:bg-card/80',
				'border-muted-foreground/20 dark:border-muted-foreground/40',
				'text-foreground',
				'hover:bg-muted/50',
				'dark:hover:bg-card/90'
			)
		}
	},
	defaultVariants: {
		variant: 'default'
	}
});

export type Variant = VariantProps<typeof badgeVariants>['variant'];

export { Root, Root as Badge };
