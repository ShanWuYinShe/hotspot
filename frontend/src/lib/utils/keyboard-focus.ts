const KEYBOARD_DISMISS_EVENT = 'keyboard:dismiss';

const registeredInputs = new WeakSet<HTMLElement>();
let managedWindow: Window | null = null;
let initialized = false;
let pendingCheck = false;
const TEXT_INPUT_TYPES = new Set([
  '',
  'text',
  'email',
  'password',
  'search',
  'tel',
  'url',
  'number'
]);

function isRegisteredInput(element: Element | null): element is HTMLElement {
  return !!element && registeredInputs.has(element as HTMLElement);
}

function dispatchKeyboardDismiss() {
  if (!managedWindow) return;
  managedWindow.dispatchEvent(new CustomEvent(KEYBOARD_DISMISS_EVENT));
}

function scheduleDismissCheck() {
  if (!managedWindow || pendingCheck) return;

  pendingCheck = true;
  managedWindow.requestAnimationFrame(() => {
    pendingCheck = false;
    const active = managedWindow?.document.activeElement as HTMLElement | null;
    if (isRegisteredInput(active)) {
      return;
    }

    dispatchKeyboardDismiss();
  });
}

function handlePointerDown(event: PointerEvent | TouchEvent) {
  if (!managedWindow) return;

  const active = managedWindow.document.activeElement as HTMLElement | null;
  if (!isRegisteredInput(active)) {
    return;
  }

  const targetNode = event.target as Node | null;
  if (targetNode && active.contains(targetNode)) {
    return;
  }

  active.blur();
  scheduleDismissCheck();
}

function handleFocusOut() {
  scheduleDismissCheck();
}

export function dismissActiveKeyboardFocus(targetWindow?: Window): boolean {
  const nextWindow = targetWindow ?? managedWindow;
  if (!nextWindow) {
    return false;
  }

  const active = nextWindow.document.activeElement as HTMLElement | null;
  if (!isRegisteredInput(active)) {
    return false;
  }

  active.blur();
  scheduleDismissCheck();
  return true;
}

export function initKeyboardFocusManager(targetWindow: Window) {
  if (initialized || !targetWindow) {
    return;
  }

  initialized = true;
  managedWindow = targetWindow;

  const doc = targetWindow.document;
  doc.addEventListener('focusout', handleFocusOut, true);
  targetWindow.addEventListener('pointerdown', handlePointerDown, true);
  targetWindow.addEventListener('touchstart', handlePointerDown, true);
}

function isTextInputElement(node: HTMLElement): boolean {
  if (node instanceof HTMLTextAreaElement) return true;
  if (node instanceof HTMLInputElement) {
    return TEXT_INPUT_TYPES.has((node.type ?? '').toLowerCase());
  }
  return node.isContentEditable;
}

export function useKeyboardFocus(): (element: HTMLElement) => void | (() => void) {
  return (node: HTMLElement): void | (() => void) => {
    if (!isTextInputElement(node)) {
      return;
    }
    registeredInputs.add(node);
    return () => {
      registeredInputs.delete(node);
    };
  };
}

export { KEYBOARD_DISMISS_EVENT };
