import Root from './button.svelte';

import { tv, type VariantProps } from 'tailwind-variants';
import { cn } from '$lib/utils';

const pressFeedback =
  'active:shadow-[inset_5px_5px_12px_var(--shadow-dark),inset_-5px_-5px_12px_var(--shadow-light)]';
const lift = 'hover:-translate-y-0.5 active:translate-y-0';

export const buttonVariants = tv({
  base: cn(
    'inline-flex items-center justify-center gap-2',
    'whitespace-nowrap font-semibold select-none',
    'rounded-[28px] border',
    'transition-all duration-200 ease-[cubic-bezier(0.22,1,0.36,1)]',
    'focus-visible:outline-none focus-visible:ring-[2px] focus-visible:ring-[var(--glow)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-bg)]',
    'disabled:pointer-events-none disabled:opacity-50',
    'cursor-pointer',
    '[&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0'
  ),
  variants: {
    variant: {
      // Filled primary CTA: brand background + readable on-accent text.
      default: cn(
        'border-transparent bg-[var(--brand)] text-on-accent button-glow',
        lift,
        pressFeedback
      ),
      accent: cn(
        'border-transparent bg-[var(--brand-secondary)] text-on-accent button-glow',
        lift,
        pressFeedback
      ),
      destructive: cn(
        'border-transparent bg-[var(--theme-error)] text-on-accent',
        lift,
        pressFeedback
      ),
      outline: cn(pressFeedback),
      secondary: cn('surface-flat', pressFeedback),
      ghost: cn('shadow-none'),
      link: cn('shadow-none', 'underline-offset-4 hover:underline'),
      // Neumorphic raised / pressed surfaces (used for tactile, low-emphasis controls).
      flat: cn('surface-raised border-transparent', lift, 'active:scale-[0.97]'),
      pressed: cn('surface-pressed', 'active:scale-[0.97]'),
      glow: cn('button-glow', lift)
    },
    size: {
      default: 'h-11 px-6 py-3 text-sm',
      sm: 'h-9 px-4 py-2 text-xs',
      lg: 'h-13 px-8 py-4 text-base rounded-[24px]',
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

export { Root, Root as Button };
