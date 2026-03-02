/**
 * 应用入口文件
 *
 * 初始化 Vue 应用实例，集成：
 * - Pinia 状态管理（含持久化插件）
 *
 * uni-app 要求导出 createApp 工厂函数，由框架负责调用
 */
import { createSSRApp } from "vue"
import * as Pinia from "pinia"
import piniaPluginPersistedstate from "pinia-plugin-persistedstate"
import App from "./App.vue"

export function createApp() {
	const app = createSSRApp(App)

	// 创建 Pinia 实例并注册持久化插件
	const pinia = Pinia.createPinia()
	pinia.use(piniaPluginPersistedstate)
	app.use(pinia)

	return {
		app,
		// uni-app 要求返回 Pinia 实例，用于 SSR 场景
		Pinia,
	}
}
