<script>
/**
 * 应用入口
 *
 * 全局生命周期：
 * - onLaunch: 初始化 WebSocket 连接（如果用户已登录）和事件监听
 * - onShow: 检查 WebSocket 连接状态并恢复
 */
import { useUserStore } from '@/store/user'
import { useWebSocketStore } from '@/store/websocket'
import { useChatStore } from '@/store/chat'
import { useContactStore } from '@/store/contact'
import { useGroupStore } from '@/store/group'
import { useNotifyStore } from '@/store/notify'

export default {
  onLaunch() {
    console.log('App Launch')
    this._initGlobalWS()
  },
  onShow() {
    console.log('App Show')
    const userStore = useUserStore()
    const wsStore = useWebSocketStore()
    if (userStore.isLoggedIn && !wsStore.isConnected) {
      wsStore.connect()
    }
  },
  onHide() {
    console.log('App Hide')
  },
  methods: {
    _initGlobalWS() {
      const userStore = useUserStore()
      if (!userStore.isLoggedIn) return
      const wsStore = useWebSocketStore()
      wsStore.connect()
      const chatStore = useChatStore()
      const contactStore = useContactStore()
      const groupStore = useGroupStore()
      const notifyStore = useNotifyStore()
      chatStore.initWsListeners()
      contactStore.initWsListeners()
      groupStore.initWsListeners()
      notifyStore.initWsListeners()
      contactStore.fetchPendingRequests().catch(() => {})
      notifyStore.fetchUnreadCount().catch(() => {})
    }
  }
}
</script>

<style>
/* 每个页面公共 css */

/*
 * 基础全屏容器：防止 uni-app H5 在大窗口下祖先容器尺寸小于视口
 * 导致 position: fixed 元素（如会议房间 .room）"偏左上"。
 * 同时让普通页面在桌面宽屏下仍能把内容区（.page / uni-page-body）
 * 拉伸到视口高度，避免下方出现大片浏览器默认底色。
 */
/* #ifdef H5 */
html, body, #app {
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0;
}
uni-app, uni-app > uni-page, uni-page-wrapper, uni-page-body {
  width: 100%;
  min-height: 100%;
}
/* #endif */
</style>
