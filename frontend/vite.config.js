import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'
import basicSsl from '@vitejs/plugin-basic-ssl'

// https://vitejs.dev/config/
//
// HTTPS 说明（WebRTC 必需）：
// 浏览器仅在 secure context 下开放 navigator.mediaDevices（getUserMedia 等 API），
// 手机通过局域网 IP http://192.168.x.x:5173 访问时 mediaDevices 会是 undefined。
// 因此 dev 默认开启自签 HTTPS，同时把 /api 与 /ws 走同源代理转回后端 HTTP，
// 这样浏览器只与 vite HTTPS 通信，前端代码无需关心后端是否 TLS。
//
// 代理目标由环境变量 VITE_DEV_BACKEND 控制，默认 http://localhost:8085，
// 若后端跑在其他主机/端口（如 docker 主机）可自行覆盖。
const BACKEND = process.env.VITE_DEV_BACKEND || 'http://localhost:8085'
const BACKEND_WS = BACKEND.replace(/^http/, 'ws')

export default defineConfig({
  plugins: [
    uni(),
    basicSsl(),
  ],
  server: {
    host: '0.0.0.0',
    https: true,
    proxy: {
      '/api': {
        target: BACKEND,
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: BACKEND_WS,
        changeOrigin: true,
        ws: true,
        secure: false,
      },
    },
  },
})
