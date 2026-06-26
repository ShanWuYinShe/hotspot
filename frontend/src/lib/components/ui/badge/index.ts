import Root from './badge.svelte';
import { tv, type VariantProps } from 'tailwind-variants';
import { cn } from '$lib/utils';

export const badgeVariants = tv({
  base: cn(
    'inline-flex items-center gap-1.5',
    'rounded-md',
    'border',
    'px-3 py-1 text-xs font-semibold',
    'transition-all duration-200',
    'shadow-sm shadow-black/10',
    'dark:shadow-md dark:shadow-black/20'
  ),
  variants: {
    variant: {
      default: cn(
        'bg-transparent border-transparent',
        'dark:bg-transparent dark:border-transparent',
        'text-primary dark:text-primary-foreground',
        'hover:bg-primary/10 hover:border-primary/30',
        'dark:hover:bg-primary/20 dark:hover:border-primary/60'
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
        'text-accent dark:text-on-accent',
        'hover:bg-accent/20 hover:border-accent/40',
        'dark:hover:bg-accent/30 dark:hover:border-accent/80'
      ),
      success: cn(
        'bg-success/10 border-success/30',
        'dark:bg-success/20 dark:border-success/60',
        'text-success',
        'hover:bg-success/20',
        'dark:hover:bg-success/30 dark:hover:border-success/70'
      ),
      destructive: cn(
        'bg-destructive/10 border-destructive/30',
        'dark:bg-destructive/20 dark:border-destructive/60',
        'text-destructive dark:text-destructive-foreground',
        'hover:bg-destructive/20 hover:border-destructive/40',
        'dark:hover:bg-destructive/30 dark:hover:border-destructive/70'
      ),
      outline: cn(
        /* surface-bg 替代 bg-white —— 在暗色/非默认主题族下不再失效 */
        'bg-[var(--surface-bg)]/80',
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
