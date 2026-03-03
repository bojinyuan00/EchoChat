<!--
  消息 - 会话列表页（TabBar 页面）

  设计系统：design-system/echochat/MASTER.md
  页面覆盖：design-system/echochat/pages/chat-index.md
  图标方案：@dcloudio/uni-ui uni-icons（跨平台兼容）
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B / Muted #94A3B8
-->
<template>
  <view class="page-wrapper">
    <!-- 顶部栏 -->
    <view class="header">
      <text class="header-title">消息</text>
      <view class="header-actions">
        <view class="action-btn" @tap="goToSearch">
          <uni-icons type="search" size="20" color="#475569" />
        </view>
      </view>
    </view>

    <!-- 会话列表 -->
    <scroll-view scroll-y class="conv-list" @scrolltolower="onScrollToLower">
      <!-- 空状态 -->
      <view v-if="!loading && conversations.length === 0" class="empty-state">
        <uni-icons type="chatbubble" size="64" color="#CBD5E1" />
        <text class="empty-text">暂无消息</text>
        <text class="empty-hint">找好友聊聊天吧</text>
      </view>

      <!-- 会话条目 -->
      <view
        v-for="conv in conversations"
        :key="conv.id"
        class="conv-item"
        :class="{ 'conv-pinned': conv.is_pinned }"
        @tap="openChat(conv)"
        @longpress="onLongPress(conv)"
      >
        <!-- 头像 -->
        <view class="conv-avatar-wrap">
          <image
            v-if="conv.peer_avatar"
            class="conv-avatar"
            :src="conv.peer_avatar"
            mode="aspectFill"
          />
          <view v-else class="conv-avatar conv-avatar-placeholder">
            <text class="avatar-text">{{ (conv.peer_nickname || '?')[0] }}</text>
          </view>
          <view v-if="conv.unread_count > 0" class="conv-badge">
            <text class="conv-badge-text">{{ conv.unread_count > 99 ? '99+' : conv.unread_count }}</text>
          </view>
        </view>

        <!-- 信息区 -->
        <view class="conv-info">
          <view class="conv-top">
            <text class="conv-name">{{ conv.peer_nickname || '未知用户' }}</text>
            <text class="conv-time">{{ formatTime(conv.last_msg_time) }}</text>
          </view>
          <view class="conv-bottom">
            <text v-if="isTyping(conv.id)" class="conv-typing">对方正在输入...</text>
            <text v-else class="conv-preview" :class="{ 'conv-preview-unread': conv.unread_count > 0 }">
              {{ conv.last_msg_content || ' ' }}
            </text>
            <view v-if="conv.is_pinned" class="conv-pin-tag">
              <uni-icons type="top" size="14" color="#94A3B8" />
            </view>
          </view>
        </view>
      </view>
    </scroll-view>

    <CustomTabBar :current="0" />
  </view>
</template>

<script>
import { onShow } from '@dcloudio/uni-app'
import { ref, computed } from 'vue'
import { useChatStore } from '@/store/chat'
import CustomTabBar from '@/components/CustomTabBar.vue'

export default {
  name: 'ChatIndex',
  components: { CustomTabBar },
  setup() {
    const chatStore = useChatStore()
    const loading = ref(false)

    const conversations = computed(() => chatStore.sortedConversations)

    const loadData = async () => {
      loading.value = true
      try {
        await chatStore.fetchConversations()
      } finally {
        loading.value = false
      }
    }

    onShow(() => {
      chatStore.initWsListeners()
      loadData()
    })

    const openChat = (conv) => {
      uni.navigateTo({
        url: `/pages/chat/conversation?conversationId=${conv.id}&peerId=${conv.peer_user_id}&peerName=${encodeURIComponent(conv.peer_nickname || '')}&peerAvatar=${encodeURIComponent(conv.peer_avatar || '')}`
      })
    }

    const goToSearch = () => {
      uni.navigateTo({ url: '/pages/chat/search' })
    }

    const isTyping = (convId) => {
      return chatStore.typingMap[convId] || false
    }

    const formatTime = (timeStr) => {
      if (!timeStr) return ''
      const date = new Date(timeStr)
      const now = new Date()
      const isToday = date.toDateString() === now.toDateString()
      if (isToday) {
        return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
      }
      const yesterday = new Date(now)
      yesterday.setDate(yesterday.getDate() - 1)
      if (date.toDateString() === yesterday.toDateString()) {
        return '昨天'
      }
      return `${date.getMonth() + 1}/${date.getDate()}`
    }

    const onLongPress = (conv) => {
      const items = [conv.is_pinned ? '取消置顶' : '置顶', '删除会话']
      uni.showActionSheet({
        itemList: items,
        success: async (res) => {
          if (res.tapIndex === 0) {
            await chatStore.pinConversation(conv.id, !conv.is_pinned)
          } else if (res.tapIndex === 1) {
            uni.showModal({
              title: '提示',
              content: '确定删除该会话？',
              success: async (r) => {
                if (r.confirm) {
                  await chatStore.deleteConversation(conv.id)
                }
              }
            })
          }
        }
      })
    }

    const onScrollToLower = () => {}

    return {
      loading,
      conversations,
      openChat,
      goToSearch,
      isTyping,
      formatTime,
      onLongPress,
      onScrollToLower
    }
  }
}
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
  padding-bottom: 120rpx;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32rpx;
  height: 88rpx;
  padding-top: var(--status-bar-height, 44px);
  background-color: #FFFFFF;
}

.header-title {
  font-size: 36rpx;
  font-weight: 700;
  color: #1E293B;
}

.header-actions {
  display: flex;
  gap: 16rpx;
}

.action-btn {
  min-width: 88rpx;
  min-height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 16rpx;
  background-color: #F1F5F9;
  transition: background-color 150ms ease;
}

.action-btn:active {
  background-color: #E2E8F0;
}

/* 会话列表 */
.conv-list {
  height: calc(100vh - 88rpx - var(--status-bar-height, 44px) - 120rpx);
}

.conv-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #F1F5F9;
  transition: background-color 150ms ease;
}

.conv-item:active {
  background-color: #F1F5F9;
}

.conv-pinned {
  background-color: #F8FAFC;
}

.conv-pinned:active {
  background-color: #F1F5F9;
}

/* 头像 */
.conv-avatar-wrap {
  position: relative;
  flex-shrink: 0;
  margin-right: 24rpx;
}

.conv-avatar {
  width: 96rpx;
  height: 96rpx;
  border-radius: 24rpx;
}

.conv-avatar-placeholder {
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-text {
  color: #FFFFFF;
  font-size: 36rpx;
  font-weight: 600;
}

.conv-badge {
  position: absolute;
  top: -8rpx;
  right: -8rpx;
  min-width: 36rpx;
  height: 36rpx;
  padding: 0 8rpx;
  background-color: #EF4444;
  border-radius: 18rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.conv-badge-text {
  color: #FFFFFF;
  font-size: 20rpx;
  line-height: 36rpx;
}

/* 信息区 */
.conv-info {
  flex: 1;
  overflow: hidden;
}

.conv-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8rpx;
}

.conv-name {
  font-size: 30rpx;
  font-weight: 500;
  color: #1E293B;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 400rpx;
}

.conv-time {
  font-size: 24rpx;
  color: #94A3B8;
  flex-shrink: 0;
}

.conv-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.conv-preview {
  font-size: 26rpx;
  color: #94A3B8;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 480rpx;
}

.conv-preview-unread {
  color: #64748B;
  font-weight: 500;
}

.conv-typing {
  font-size: 26rpx;
  color: #2563EB;
}

.conv-pin-tag {
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 200rpx;
}

.empty-text {
  font-size: 32rpx;
  font-weight: 600;
  color: #1E293B;
  margin-top: 24rpx;
  margin-bottom: 8rpx;
}

.empty-hint {
  font-size: 26rpx;
  color: #94A3B8;
}
</style>
