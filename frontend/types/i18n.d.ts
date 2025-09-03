declare module '#i18n' {
  // Minimal runtime helpers exposed by @nuxtjs/i18n for programmatic control
  export function setLocale(code: string): Promise<void> | void
  export function getLocaleMessage(code: string): any
}
