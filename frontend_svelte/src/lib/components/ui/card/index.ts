import Root from './card.svelte';
import Content from './card-content.svelte';
import Description from './card-description.svelte';
import Footer from './card-footer.svelte';
import Header from './card-header.svelte';
import Title from './card-title.svelte';

import { tv, type VariantProps } from 'tailwind-variants';

export const cardVariants = tv({
  base: 'rounded-[32px] border border-border/80 bg-card/95 backdrop-blur-xl transition-all duration-200 text-card-foreground shadow-[8px_8px_20px_var(--neu-shadow-dark),-8px_-8px_20px_var(--neu-shadow-light)]',
  variants: {
    variant: {
      default: 'border-border/80 bg-card/95',
      primary: 'border-primary/25 bg-primary/6 dark:bg-primary/10',
      accent: 'border-border/80 bg-accent/80',
      secondary: 'border-border/80 bg-secondary/85',
      muted: 'border-border/70 bg-muted/75'
    },
    glow: {
      none: '',
      soft: 'hover:ring-2 hover:ring-primary/12',
      strong: 'ring-2 ring-primary/16 hover:ring-primary/24'
    },
    interactive: {
      true: 'cursor-pointer select-none hover:shadow-[10px_10px_24px_var(--neu-shadow-dark),-10px_-10px_24px_var(--neu-shadow-light)] active:shadow-[inset_6px_6px_12px_var(--neu-shadow-dark),inset_-6px_-6px_12px_var(--neu-shadow-light)] focus-visible:outline-none focus-visible:ring-[2px] focus-visible:ring-[var(--neu-glow)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--neu-bg)]',
      false: ''
    }
  },
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
