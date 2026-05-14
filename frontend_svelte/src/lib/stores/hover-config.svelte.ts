import { themeStore } from './theme.svelte';
import type { ThemeFamily } from './theme.svelte';

export interface HoverVariant {
  y?: number;
  scale: number;
  duration: number;
  easing: string;
}

export interface HoverConfig {
  card: HoverVariant;
  row: HoverVariant;
  btn: HoverVariant;
  action: HoverVariant;
  tap: HoverVariant;
}

const EASE_OUT = 'cubic-bezier(0.22, 1, 0.36, 1)';

const configs: Record<ThemeFamily, HoverConfig> = {
  neu: {
    card: { y: -7, scale: 1.018, duration: 260, easing: EASE_OUT },
    row: { y: -5, scale: 1.01, duration: 240, easing: EASE_OUT },
    btn: { scale: 1.12, duration: 220, easing: EASE_OUT },
    action: { scale: 1.22, duration: 200, easing: EASE_OUT },
    tap: { scale: 0.94, duration: 180, easing: EASE_OUT }
  },
  vibrant: {
    card: { y: -5, scale: 1.014, duration: 220, easing: EASE_OUT },
    row: { y: -3, scale: 1.009, duration: 180, easing: EASE_OUT },
    btn: { scale: 1.1, duration: 170, easing: EASE_OUT },
    action: { scale: 1.2, duration: 140, easing: EASE_OUT },
    tap: { scale: 0.94, duration: 100, easing: EASE_OUT }
  },
  forest: {
    card: { scale: 1.016, duration: 200, easing: EASE_OUT },
    row: { scale: 1.01, duration: 160, easing: EASE_OUT },
    btn: { scale: 1.08, duration: 160, easing: EASE_OUT },
    action: { scale: 1.18, duration: 140, easing: EASE_OUT },
    tap: { scale: 0.97, duration: 100, easing: EASE_OUT }
  }
};

const defaultConfig = configs.vibrant;

class HoverConfigStore {
  config = $derived(configs[themeStore.themeFamily] ?? defaultConfig);

  toCssTransition(variant: HoverVariant): string {
    return `transform ${variant.duration}ms ${variant.easing}`;
  }
}

export const hoverConfig = new HoverConfigStore();
