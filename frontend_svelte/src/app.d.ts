// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
  namespace App {
    /**
     * 自定义错误类型，用于统一错误展示
     */
    interface Error {
      /** 业务错误码 */
      code?: number;
      /** 错误描述 */
      message: string;
    }

    // interface Locals {}

    // interface PageData {}
    // interface PageState {}
    // interface Platform {}
  }
}

export {};
