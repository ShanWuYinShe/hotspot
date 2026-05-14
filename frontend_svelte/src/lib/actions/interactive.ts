import { hoverConfig, type HoverVariant } from '$lib/stores/hover-config.svelte';

export type InteractiveType = 'card' | 'row' | 'btn' | 'action';

export interface InteractiveOptions {
  type?: InteractiveType;
  disabled?: boolean;
}

function getVariant(type: InteractiveType): HoverVariant {
  const config = hoverConfig.config;
  switch (type) {
    case 'card':
      return config!.card;
    case 'row':
      return config!.row;
    case 'action':
      return config!.action;
    default:
      return config!.btn;
  }
}

function applyTransform(node: HTMLElement, variant: HoverVariant) {
  const transforms: string[] = [];
  if (variant.y) transforms.push(`translateY(${variant.y}px)`);
  if (variant.scale !== 1) transforms.push(`scale(${variant.scale})`);
  node.style.transform = transforms.join(' ');
  node.style.transition = hoverConfig.toCssTransition(variant);
}

function clearTransform(node: HTMLElement) {
  node.style.transform = '';
}

export function interactive(node: HTMLElement, options?: InteractiveOptions) {
  let opts = options || { type: 'btn' };

  function onEnter() {
    if (opts.disabled) return;
    applyTransform(node, getVariant(opts.type || 'btn'));
  }

  function onLeave() {
    clearTransform(node);
  }

  function onDown() {
    if (opts.disabled) return;
    const tapVariant = hoverConfig.config!.tap;
    const transforms: string[] = [];
    if (tapVariant.scale !== 1) transforms.push(`scale(${tapVariant.scale})`);
    node.style.transform = transforms.join(' ');
    node.style.transition = hoverConfig.toCssTransition(tapVariant);
  }

  function onUp() {
    onEnter();
  }

  node.addEventListener('mouseenter', onEnter);
  node.addEventListener('mouseleave', onLeave);
  node.addEventListener('mousedown', onDown);
  node.addEventListener('mouseup', onUp);
  node.addEventListener('touchstart', onDown, { passive: true });
  node.addEventListener('touchend', onUp, { passive: true });

  return {
    update(newOptions?: InteractiveOptions) {
      opts = newOptions || { type: 'btn' };
    },
    destroy() {
      node.removeEventListener('mouseenter', onEnter);
      node.removeEventListener('mouseleave', onLeave);
      node.removeEventListener('mousedown', onDown);
      node.removeEventListener('mouseup', onUp);
      node.removeEventListener('touchstart', onDown);
      node.removeEventListener('touchend', onUp);
    }
  };
}
