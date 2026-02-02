import Root from './card.svelte';
import Content from './card-content.svelte';
import Description from './card-description.svelte';
import Footer from './card-footer.svelte';
import Header from './card-header.svelte';
import Title from './card-title.svelte';

import { tv, type VariantProps } from 'tailwind-variants';

// Game UI - Neubrutalism card variants with hard shadows
export const cardVariants = tv({
	base: 'rounded-2xl md:rounded-3xl border-[3px] md:border-[3px] lg:border-[4px] backdrop-blur-xl transition-all duration-150 ease-[cubic-bezier(0.4,0,0.2,1)] text-card-foreground',
	variants: {
		variant: {
			default: [
				// Borders - 清晰的边界
				'border-border',
				// Background
				'bg-white/95 dark:bg-card/95',
				// Hard shadow (Neubrutalism)
				'shadow-[3px_3px_0px_theme(colors.border)]',
				'md:shadow-[4px_4px_0px_theme(colors.border)]',
				'lg:shadow-[5px_5px_0px_theme(colors.border)]'
			],
			primary: [
				// Primary themed card
				'border-primary',
				'bg-primary/5 dark:bg-primary/10',
				'shadow-[3px_3px_0px_theme(colors.primary)]',
				'md:shadow-[4px_4px_0px_theme(colors.primary)]',
				'lg:shadow-[5px_5px_0px_theme(colors.primary)]'
			],
			accent: [
				// Accent themed card
				'border-accent',
				'bg-accent/5 dark:bg-accent/10',
				'shadow-[3px_3px_0px_theme(colors.accent)]',
				'md:shadow-[4px_4px_0px_theme(colors.accent)]',
				'lg:shadow-[5px_5px_0px_theme(colors.accent)]'
			],
			secondary: [
				// Secondary themed card
				'border-secondary',
				'bg-secondary/5 dark:bg-secondary/10',
				'shadow-[3px_3px_0px_theme(colors.secondary)]',
				'md:shadow-[4px_4px_0px_theme(colors.secondary)]',
				'lg:shadow-[5px_5px_0px_theme(colors.secondary)]'
			],
			muted: [
				// Muted card
				'border-muted',
				'bg-muted/50 dark:bg-muted/30',
				'shadow-[3px_3px_0px_theme(colors.muted)]',
				'md:shadow-[4px_4px_0px_theme(colors.muted)]',
				'lg:shadow-[5px_5px_0px_theme(colors.muted)]'
			]
		},
		glow: {
			none: '',
			soft: [
				// Subtle glow on hover
				'hover:ring-2 hover:ring-primary/20',
				'dark:hover:ring-primary/40'
			],
			strong: [
				// Strong glow (always visible)
				'ring-2 ring-primary/20',
				'dark:ring-primary/40',
				'hover:ring-primary/30',
				'dark:hover:ring-primary/60'
			]
		},
		interactive: {
			true: [
				'cursor-pointer',
				// Hover - move up-left, increase shadow
				'hover:-translate-x-0.5 hover:-translate-y-0.5',
				'hover:shadow-[5px_5px_0px_theme(colors.border)]',
				'md:hover:shadow-[6px_6px_0px_theme(colors.border)]',
				'lg:hover:shadow-[7px_7px_0px_theme(colors.border)]',
				// Active - press down effect
				'active:translate-x-[1px] active:translate-y-[1px]',
				'active:shadow-[2px_2px_0px_theme(colors.border)]'
			],
			false: ''
		}
	},
	compoundVariants: [
		{
			variant: 'primary',
			interactive: true,
			class: [
				'hover:shadow-[5px_5px_0px_theme(colors.primary)]',
				'md:hover:shadow-[6px_6px_0px_theme(colors.primary)]',
				'lg:hover:shadow-[7px_7px_0px_theme(colors.primary)]',
				'active:shadow-[2px_2px_0px_theme(colors.primary)]'
			]
		},
		{
			variant: 'accent',
			interactive: true,
			class: [
				'hover:shadow-[5px_5px_0px_theme(colors.accent)]',
				'md:hover:shadow-[6px_6px_0px_theme(colors.accent)]',
				'lg:hover:shadow-[7px_7px_0px_theme(colors.accent)]',
				'active:shadow-[2px_2px_0px_theme(colors.accent)]'
			]
		}
	],
	defaultVariants: {
		variant: 'default',
		glow: 'none',
		interactive: false
	}
});

export type CardVariant = VariantProps<typeof cardVariants>['variant'];
export type CardGlow = VariantProps<typeof cardVariants>['glow'];

export {
	Root,
	Content,
	Description,
	Footer,
	Header,
	Title,
	Root as Card,
	Content as CardContent,
	Description as CardDescription,
	Footer as CardFooter,
	Header as CardHeader,
	Title as CardTitle
};
