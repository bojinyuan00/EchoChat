<!--
  聊天对话页

  布局规则：
  - 自己的消息：靠右对齐，顺序 [状态图标] [蓝色气泡] [自己头像]
  - 对方的消息：靠左对齐，顺序 [对方头像] [白色气泡]
  - 导航栏：返回 / 对方昵称 / 更多
-->
<template>
  <view class="page-wrapper">
    <!-- 导航栏 -->
    <view class="nav-bar">
      <view class="nav-left" @tap="goBack">
        <uni-icons type="back" size="20" color="#1E293B" />
      </view>
      <view class="nav-center">
        <text class="nav-title">{{ peerName || '聊天' }}</text>
        <text v-if="isTyping" class="nav-typing">正在输入...</text>
      </view>
      <view class="nav-right" @tap="goToSettings">
        <uni-icons type="more-filled" size="20" color="#475569" />
      </view>
    </view>

    <!-- 消息列表 -->
    <scroll-view
      scroll-y
      class="msg-list"
      :scroll-into-view="scrollToId"
      :scroll-with-animation="true"
      @scrolltoupper="onLoadMore"
    >
      <view v-if="hasMore" class="load-more" @tap="onLoadMore">
        <text class="load-more-text">{{ loadingMore ? '加载中...' : '加载更多' }}</text>
      </view>

      <view
        v-for="msg in messages"
        :key="msg.client_msg_id || msg.id"
        :id="'msg-' + (msg.id || msg.client_msg_id)"
        class="msg-row"
        :class="isSelf(msg) ? 'msg-row-self' : 'msg-row-other'"
      >
        <!-- ====== 对方消息（左侧）：[头像] [气泡] ====== -->
        <template v-if="!isSelf(msg)">
          <view class="avatar-wrap">
            <image v-if="peerAvatar" class="avatar-img" :src="peerAvatar" mode="aspectFill" />
            <view v-else class="avatar-img avatar-placeholder avatar-peer">
              <text class="avatar-char">{{ (peerName || '?')[0] }}</text>
            </view>
          </view>
          <view
            class="bubble bubble-other"
            :class="{ 'bubble-recalled': msg.status === 2 }"
            @longpress="onMsgLongPress(msg)"
          >
            <text v-if="msg.status === 2" class="recalled-text">消息已撤回</text>
            <text v-else class="msg-text">{{ msg.content }}</text>
          </view>
        </template>

        <!-- ====== 自己消息（右侧）：[状态] [气泡] [头像] ====== -->
        <template v-else>
          <view class="self-msg-col">
            <view class="self-msg-row">
              <view v-if="msg._sending" class="msg-status">
                <uni-icons type="loop" size="16" color="#94A3B8" />
              </view>
              <view v-if="msg._failed" class="msg-status msg-status-tap" @tap="onResend(msg)">
                <uni-icons type="info-filled" size="18" color="#EF4444" />
              </view>
              <view
                class="bubble bubble-self"
                :class="{ 'bubble-recalled': msg.status === 2 }"
                @longpress="onMsgLongPress(msg)"
              >
                <text v-if="msg.status === 2" class="recalled-text">消息已撤回</text>
                <text v-else class="msg-text msg-text-self">{{ msg.content }}</text>
              </view>
              <view class="avatar-wrap">
                <image v-if="selfAvatar" class="avatar-img" :src="selfAvatar" mode="aspectFill" />
                <view v-else class="avatar-img avatar-placeholder avatar-self">
                  <text class="avatar-char">{{ (selfName || '我')[0] }}</text>
                </view>
              </view>
            </view>
            <text v-if="getReadLabel(msg)" class="read-label" :class="isRead(msg) ? 'read-label-read' : 'read-label-unread'">
              {{ getReadLabel(msg) }}
            </text>
          </view>
        </template>
      </view>

      <view id="msg-bottom" style="height: 2rpx;" />
    </scroll-view>

    <!-- 输入栏 -->
    <view class="input-bar">
      <view class="input-wrap">
        <input
          class="msg-input"
          v-model="inputText"
          placeholder="输入消息..."
          placeholder-style="color: #94A3B8"
          confirm-type="send"
          @confirm="onSend"
          @input="onInputChange"
          :adjust-position="true"
        />
      </view>
      <view class="send-btn" :class="{ 'send-btn-active': inputText.trim() }" @tap="onSend">
        <uni-icons type="paperplane" size="22" color="#FFFFFF" />
      </view>
    </view>
  </view>
</template>

<script>
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { ref, computed, nextTick, watch } from 'vue'
import { useChatStore } from '@/store/chat'
import { useUserStore } from '@/store/user'

export default {
  name: 'ChatConversation',
  setup() {
    const chatStore = useChatStore()
    const userStore = useUserStore()

    const conversationId = ref(0)
    const peerId = ref(0)
    const peerName = ref('')
    const peerAvatar = ref('')
    const inputText = ref('')
    const scrollToId = ref('')
    const loadingMore = ref(false)

    const messages = computed(() => chatStore.currentMessages)
    const hasMore = computed(() => chatStore.hasMoreMap[conversationId.value] !== false)
    const isTyping = computed(() => chatStore.typingMap[conversationId.value] || false)
    const selfAvatar = computed(() => userStore.userInfo?.avatar || '')
    const selfName = computed(() => userStore.userInfo?.nickname || userStore.userInfo?.username || '我')

    const convType = ref(1)

    const isSelf = (msg) => {
      const myId = Number(userStore.userInfo?.id) || 0
      return msg.sender_id === myId || msg._sending === true || msg._failed === true
    }

    const isRead = (msg) => {
      if (convType.value === 2) return false
      if (msg.status === 2 || msg._sending || msg._failed || !msg.id) return false
      const lastReadId = chatStore.readStatusMap[conversationId.value] || 0
      return msg.id <= lastReadId
    }

    const getReadLabel = (msg) => {
      if (msg.status === 2 || msg._sending || msg._failed || !msg.id) return ''
      if (convType.value === 2) {
        const count = chatStore.groupReadCountMap[msg.id]
        return count > 0 ? `${count}人已读` : ''
      }
      return isRead(msg) ? '已读' : '未读'
    }

    const tryFindExistingConversation = async () => {
      try {
        await chatStore.fetchConversations()
        const existingConv = chatStore.conversationList.find(c => c.peer_user_id === peerId.value)
        if (existingConv) {
          conversationId.value = existingConv.id
          chatStore.setCurrentConversation(existingConv.id)
          await loadInitialMessages()
        }
      } catch (e) {
        console.warn('[Chat] 查找已有会话失败', e)
      }
    }

    onLoad((query) => {
      conversationId.value = parseInt(query.conversationId) || 0
      peerId.value = parseInt(query.peerId) || 0
      peerName.value = decodeURIComponent(query.peerName || '')
      peerAvatar.value = decodeURIComponent(query.peerAvatar || '')
      convType.value = parseInt(query.convType) || 1

      chatStore.initWsListeners()

      if (conversationId.value) {
        chatStore.setCurrentConversation(conversationId.value)
        loadInitialMessages()
      } else if (peerId.value) {
        tryFindExistingConversation()
      }
    })

    onUnload(() => {
      chatStore.setCurrentConversation(null)
    })

    watch(() => chatStore.currentConversationId, (newId) => {
      if (newId && newId !== conversationId.value) {
        conversationId.value = newId
      }
    })

    const loadInitialMessages = async () => {
      if (!chatStore.messagesMap[conversationId.value]?.length) {
        await chatStore.loadHistoryMessages(conversationId.value)
      }
      scrollToBottom()
    }

    const scrollToBottom = () => {
      nextTick(() => {
        scrollToId.value = ''
        nextTick(() => { scrollToId.value = 'msg-bottom' })
      })
    }

    const onSend = () => {
      const content = inputText.value.trim()
      if (!content) return
      chatStore.sendMessage({
        conversationId: conversationId.value || 0,
        targetUserId: conversationId.value ? 0 : peerId.value,
        content,
        type: 1
      })
      inputText.value = ''
      scrollToBottom()
    }

    let typingTimer = null
    const onInputChange = () => {
      if (typingTimer) return
      if (conversationId.value) {
        chatStore.sendTyping(conversationId.value)
      }
      typingTimer = setTimeout(() => { typingTimer = null }, 3000)
    }

    const onLoadMore = async () => {
      if (loadingMore.value || !hasMore.value) return
      loadingMore.value = true
      try { await chatStore.loadHistoryMessages(conversationId.value) }
      finally { loadingMore.value = false }
    }

    const canRecall = (msg) => {
      if (!msg.created_at) return false
      return (Date.now() - new Date(msg.created_at).getTime()) < 2 * 60 * 1000
    }

    const onMsgLongPress = (msg) => {
      if (!isSelf(msg) || msg.status === 2 || !canRecall(msg)) return
      uni.showActionSheet({
        itemList: ['撤回'],
        success: (res) => {
          if (res.tapIndex === 0 && msg.id) chatStore.recallMessage(msg.id)
        }
      })
    }

    const onResend = (msg) => {
      uni.showModal({
        title: '提示',
        content: '是否重新发送？',
        success: (res) => {
          if (res.confirm) {
            chatStore.sendMessage({ conversationId: conversationId.value, content: msg.content, type: msg.type || 1 })
          }
        }
      })
    }

    const goBack = () => {
      if (getCurrentPages().length > 1) {
        uni.navigateBack()
      } else {
        uni.switchTab({ url: '/pages/chat/index' })
      }
    }

    const goToSettings = () => {
      uni.navigateTo({
        url: `/pages/chat/settings?conversationId=${conversationId.value}&peerId=${peerId.value}&peerName=${encodeURIComponent(peerName.value)}&peerAvatar=${encodeURIComponent(peerAvatar.value)}`
      })
    }

    return {
      peerName, peerAvatar, selfAvatar, selfName,
      inputText, scrollToId, loadingMore, convType,
      messages, hasMore, isTyping,
      isSelf, isRead, getReadLabel,
      onSend, onInputChange, onLoadMore,
      onMsgLongPress, onResend, goBack, goToSettings
    }
  }
}
</script>

<style scoped>
.page-wrapper {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: #F1F5F9;
  overflow: hidden;
}

/* ===== 导航栏 ===== */
.nav-bar {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 24rpx;
  padding-top: var(--status-bar-height, 44px);
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #E2E8F0;
}
.nav-left, .nav-right {
  min-width: 88rpx;
  min-height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 150ms ease;
}
.nav-left:active, .nav-right:active { opacity: 0.6; }
.nav-center { flex: 1; text-align: center; }
.nav-title { font-size: 32rpx; font-weight: 600; color: #1E293B; }
.nav-typing { display: block; font-size: 22rpx; color: #2563EB; margin-top: 2rpx; }

/* ===== 消息列表 ===== */
.msg-list {
  flex: 1;
  height: 0;
  min-height: 0;
  padding: 16rpx 24rpx;
  box-sizing: border-box;
  width: 100%;
  overflow: hidden;
}
.load-more { text-align: center; padding: 16rpx 0; }
.load-more-text { font-size: 24rpx; color: #94A3B8; }

/* ===== 消息行 ===== */
.msg-row {
  display: flex;
  align-items: flex-start;
  margin-bottom: 24rpx;
}
.msg-row-other {
  justify-content: flex-start;
}
.msg-row-self {
  justify-content: flex-end;
}

/* ===== 头像 ===== */
.avatar-wrap { flex-shrink: 0; }
.msg-row-other .avatar-wrap { margin-right: 16rpx; }
.msg-row-self .avatar-wrap { margin-left: 16rpx; }

.avatar-img {
  width: 72rpx;
  height: 72rpx;
  border-radius: 18rpx;
}
.avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
}
.avatar-peer { background-color: #2563EB; }
.avatar-self { background-color: #64748B; }
.avatar-char { color: #FFFFFF; font-size: 28rpx; font-weight: 600; }

/* ===== 气泡 ===== */
.bubble {
  max-width: 65vw;
  padding: 20rpx 24rpx;
  border-radius: 24rpx;
  word-break: break-word;
  overflow-wrap: break-word;
}
.bubble-self {
  background-color: #2563EB;
  border-bottom-right-radius: 8rpx;
}
.bubble-other {
  background-color: #FFFFFF;
  border-bottom-left-radius: 8rpx;
}
.bubble-recalled {
  background-color: transparent !important;
  padding: 8rpx 16rpx;
}

.msg-text { font-size: 30rpx; line-height: 42rpx; color: #1E293B; }
.msg-text-self { color: #FFFFFF; }
.recalled-text { font-size: 24rpx; color: #94A3B8; font-style: italic; }

/* ===== 自己消息布局（含已读标记） ===== */
.self-msg-col {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}
.self-msg-row {
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
}
.read-label {
  font-size: 20rpx;
  margin-top: 4rpx;
  margin-right: 88rpx;
}
.read-label-read { color: #2563EB; }
.read-label-unread { color: #94A3B8; }

/* ===== 发送状态 ===== */
.msg-status {
  display: flex;
  align-items: center;
  align-self: center;
  margin: 0 8rpx;
}
.msg-status-tap {
  min-width: 44rpx;
  min-height: 44rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ===== 输入栏 ===== */
.input-bar {
  display: flex;
  align-items: center;
  padding: 16rpx 24rpx;
  padding-bottom: calc(16rpx + env(safe-area-inset-bottom, 0));
  background-color: #FFFFFF;
  border-top: 1rpx solid #E2E8F0;
}
.input-wrap {
  flex: 1;
  background-color: #F1F5F9;
  border-radius: 36rpx;
  padding: 0 28rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
}
.msg-input { flex: 1; font-size: 28rpx; color: #1E293B; }
.send-btn {
  min-width: 72rpx;
  min-height: 72rpx;
  margin-left: 16rpx;
  border-radius: 50%;
  background-color: #CBD5E1;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 200ms ease;
}
.send-btn-active { background-color: #2563EB; }
.send-btn:active { opacity: 0.85; }
</style>
