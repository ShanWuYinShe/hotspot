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

		/**
		 * 服务端 Locals，通过 hooks.server.ts 注入
		 */
		interface Locals {
			/** 用户是否有有效的 refresh_token cookie */
			hasAuth: boolean;
		}

		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
