<!--
  聊天对话页

  设计系统：design-system/echochat/MASTER.md
  页面覆盖：design-system/echochat/pages/chat-conversation.md
  图标方案：@dcloudio/uni-ui uni-icons
  色板：Primary #2563EB / BG #F1F5F9 / Text #1E293B

  功能：
  - 消息气泡（左侧对方 / 右侧自己）
  - 游标分页加载历史消息
  - 消息发送（三态：sending → sent → failed）
  - 正在输入提示
  - 消息撤回（长按自己的消息）
-->
<template>
  <view class="page-wrapper">
    <!-- 自定义导航栏 -->
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
      <!-- 加载更多提示 -->
      <view v-if="hasMore" class="load-more" @tap="onLoadMore">
        <text class="load-more-text">{{ loadingMore ? '加载中...' : '上拉加载更多' }}</text>
      </view>

      <!-- 消息气泡 -->
      <view
        v-for="msg in messages"
        :key="msg.client_msg_id || msg.id"
        :id="'msg-' + (msg.id || msg.client_msg_id)"
        class="msg-row"
        :class="{ 'msg-row-self': isSelf(msg), 'msg-row-other': !isSelf(msg) }"
      >
        <!-- 对方头像 -->
        <view v-if="!isSelf(msg)" class="msg-avatar-wrap">
          <image v-if="peerAvatar" class="msg-avatar" :src="peerAvatar" mode="aspectFill" />
          <view v-else class="msg-avatar msg-avatar-placeholder">
            <text class="msg-avatar-text">{{ (peerName || '?')[0] }}</text>
          </view>
        </view>

        <!-- 消息内容 -->
        <view
          class="msg-bubble"
          :class="{
            'bubble-self': isSelf(msg),
            'bubble-other': !isSelf(msg),
            'bubble-recalled': msg.status === 2
          }"
          @longpress="onMsgLongPress(msg)"
        >
          <text v-if="msg.status === 2" class="msg-recalled">消息已撤回</text>
          <text v-else class="msg-text">{{ msg.content }}</text>
        </view>

        <!-- 发送状态 -->
        <view v-if="isSelf(msg) && msg._sending" class="msg-status">
          <uni-icons type="loop" size="16" color="#94A3B8" />
        </view>
        <view v-if="isSelf(msg) && msg._failed" class="msg-status msg-status-tap" @tap="onResend(msg)">
          <uni-icons type="info-filled" size="18" color="#EF4444" />
        </view>
      </view>

      <view id="msg-bottom" style="height: 2rpx;" />
    </scroll-view>

    <!-- 底部输入区 -->
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
import { ref, computed, nextTick } from 'vue'
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

    const isSelf = (msg) => {
      return msg.sender_id === (userStore.userInfo?.id || 0) || msg._sending
    }

    onLoad((query) => {
      conversationId.value = parseInt(query.conversationId) || 0
      peerId.value = parseInt(query.peerId) || 0
      peerName.value = decodeURIComponent(query.peerName || '')
      peerAvatar.value = decodeURIComponent(query.peerAvatar || '')

      chatStore.initWsListeners()

      if (conversationId.value) {
        chatStore.setCurrentConversation(conversationId.value)
        loadInitialMessages()
      }
    })

    onUnload(() => {
      chatStore.setCurrentConversation(null)
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
        nextTick(() => {
          scrollToId.value = 'msg-bottom'
        })
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
      typingTimer = setTimeout(() => {
        typingTimer = null
      }, 3000)
    }

    const onLoadMore = async () => {
      if (loadingMore.value || !hasMore.value) return
      loadingMore.value = true
      try {
        await chatStore.loadHistoryMessages(conversationId.value)
      } finally {
        loadingMore.value = false
      }
    }

    const onMsgLongPress = (msg) => {
      if (!isSelf(msg) || msg.status === 2) return
      uni.showActionSheet({
        itemList: ['撤回'],
        success: (res) => {
          if (res.tapIndex === 0 && msg.id) {
            chatStore.recallMessage(msg.id)
          }
        }
      })
    }

    const onResend = (msg) => {
      uni.showModal({
        title: '提示',
        content: '是否重新发送？',
        success: (res) => {
          if (res.confirm) {
            chatStore.sendMessage({
              conversationId: conversationId.value,
              content: msg.content,
              type: msg.type || 1
            })
          }
        }
      })
    }

    const goBack = () => {
      uni.navigateBack()
    }

    const goToSettings = () => {
      uni.navigateTo({
        url: `/pages/chat/settings?conversationId=${conversationId.value}&peerId=${peerId.value}&peerName=${encodeURIComponent(peerName.value)}&peerAvatar=${encodeURIComponent(peerAvatar.value)}`
      })
    }

    return {
      peerName,
      peerAvatar,
      inputText,
      scrollToId,
      loadingMore,
      messages,
      hasMore,
      isTyping,
      isSelf,
      onSend,
      onInputChange,
      onLoadMore,
      onMsgLongPress,
      onResend,
      goBack,
      goToSettings
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
}

/* 导航栏 */
.nav-bar {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 24rpx;
  padding-top: var(--status-bar-height, 44px);
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #E2E8F0;
}

.nav-left {
  min-width: 88rpx;
  min-height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 150ms ease;
}

.nav-left:active {
  opacity: 0.6;
}

.nav-center {
  flex: 1;
  text-align: center;
}

.nav-title {
  font-size: 32rpx;
  font-weight: 600;
  color: #1E293B;
}

.nav-typing {
  display: block;
  font-size: 22rpx;
  color: #2563EB;
  margin-top: 2rpx;
}

.nav-right {
  min-width: 88rpx;
  min-height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 150ms ease;
}

.nav-right:active {
  opacity: 0.6;
}

/* 消息列表 */
.msg-list {
  flex: 1;
  padding: 16rpx 24rpx;
}

.load-more {
  text-align: center;
  padding: 16rpx 0;
}

.load-more-text {
  font-size: 24rpx;
  color: #94A3B8;
}

/* 消息行 */
.msg-row {
  display: flex;
  align-items: flex-start;
  margin-bottom: 24rpx;
}

.msg-row-self {
  flex-direction: row-reverse;
}

.msg-row-other {
  flex-direction: row;
}

/* 头像 */
.msg-avatar-wrap {
  flex-shrink: 0;
  margin-right: 16rpx;
}

.msg-avatar {
  width: 72rpx;
  height: 72rpx;
  border-radius: 18rpx;
}

.msg-avatar-placeholder {
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
}

.msg-avatar-text {
  color: #FFFFFF;
  font-size: 28rpx;
  font-weight: 600;
}

/* 气泡 */
.msg-bubble {
  max-width: 560rpx;
  padding: 20rpx 24rpx;
  border-radius: 24rpx;
  word-break: break-all;
}

.bubble-self {
  background-color: #2563EB;
  border-bottom-right-radius: 8rpx;
  margin-left: 16rpx;
}

.bubble-other {
  background-color: #FFFFFF;
  border-bottom-left-radius: 8rpx;
}

.bubble-recalled {
  background-color: transparent !important;
  padding: 8rpx 16rpx;
}

.msg-text {
  font-size: 30rpx;
  line-height: 42rpx;
}

.bubble-self .msg-text {
  color: #FFFFFF;
}

.bubble-other .msg-text {
  color: #1E293B;
}

.msg-recalled {
  font-size: 24rpx;
  color: #94A3B8;
  font-style: italic;
}

/* 发送状态 */
.msg-status {
  display: flex;
  align-items: center;
  margin: 0 8rpx;
  align-self: center;
}

.msg-status-tap {
  min-width: 44rpx;
  min-height: 44rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 输入栏 */
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

.msg-input {
  flex: 1;
  font-size: 28rpx;
  color: #1E293B;
}

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

.send-btn-active {
  background-color: #2563EB;
}

.send-btn:active {
  opacity: 0.85;
}
</style>
