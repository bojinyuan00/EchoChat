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
      chatStore.initWsListeners()
      contactStore.initWsListeners()
      groupStore.initWsListeners()
    }
  }
}
</script>

<style>
/*每个页面公共css */
</style>
