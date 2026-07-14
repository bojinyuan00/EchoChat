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

// MinIO 对象存储代理目标，默认 http://localhost:9000
// 历史原因：后端 FileService.buildURL 直接返回 http://localhost:9000/{bucket}/{object}。
// PC 端访问 https://localhost:5173 时，localhost:9000 恰好命中本机 MinIO（且 localhost 是
// secure-context 例外，无 mixed-content 拦截）；但手机端访问 https://<lan-ip>:5173 时，
// 手机自身的 localhost 并不是开发机，会导致图片/语音/文件全部加载失败。
// 这里开一条 /minio 同源代理，让前端统一通过 vite HTTPS 终止后转发到 MinIO，
// 配合 utils/file.js::normalizeMediaUrl 把历史消息里的绝对 URL 重写为 /minio 即可。
const MINIO = process.env.VITE_DEV_MINIO || 'http://localhost:9000'

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
      '/minio': {
        target: MINIO,
        changeOrigin: true,
        secure: false,
        rewrite: (path) => path.replace(/^\/minio/, ''),
      },
    },
  },
})
