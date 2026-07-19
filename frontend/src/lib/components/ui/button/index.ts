import Root from './button.svelte';

import { tv, type VariantProps } from 'tailwind-variants';
import { cn } from '$lib/utils';

// Press feedback: scale-down (replaces old neumorphic inset shadow, which depended
// on --theme-shadow-dark/light that are now 0 after the flatten refactor).
const pressFeedback = 'active:scale-[0.97]';
const lift = 'hover:-translate-y-0.5 active:translate-y-0';

export const buttonVariants = tv({
  base: cn(
    'inline-flex items-center justify-center gap-2',
    'whitespace-nowrap font-semibold select-none',
    'rounded-[var(--radius-button)] border',
    'transition-all duration-200 ease-[cubic-bezier(0.22,1,0.36,1)]',
    'focus-visible:outline-none focus-visible:ring-[2px] focus-visible:ring-[var(--glow)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-bg)]',
    'disabled:pointer-events-none disabled:opacity-50',
    'cursor-pointer',
    '[&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0'
  ),
  variants: {
    variant: {
      // Filled primary CTA: brand background + readable on-accent text.
      // (No box-shadow — flatten refactor removed the neumorphic glow class;
      // brand bg provides the visual weight.)
      default: cn('border-transparent bg-[var(--brand)] text-on-accent', lift, pressFeedback),
      accent: cn(
        'border-transparent bg-[var(--brand-secondary)] text-on-accent',
        lift,
        pressFeedback
      ),
      destructive: cn('border-transparent bg-[var(--theme-error)] text-on-accent', lift, pressFeedback),
      outline: cn(pressFeedback),
      secondary: cn('surface-flat', pressFeedback),
      ghost: cn('shadow-none'),
      link: cn('shadow-none', 'underline-offset-4 hover:underline'),
      // 扁平变体: border + 略深 bg(surface-flat),不再 neumorphic raised。
      flat: cn('surface-flat', lift, pressFeedback),
      pressed: cn('surface-pressed', pressFeedback),
      glow: cn('border-transparent bg-[var(--brand)] text-on-accent', lift, pressFeedback)
    },
    size: {
      default: 'h-11 px-6 py-3 text-sm',
      sm: 'h-9 px-4 py-2 text-xs',
      lg: 'h-13 px-8 py-4 text-base rounded-[var(--radius-xl)]',
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
