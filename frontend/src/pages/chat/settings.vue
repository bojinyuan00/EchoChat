<!--
  聊天设置页

  设计系统：design-system/echochat/MASTER.md
  功能：对方信息展示、置顶、清空聊天记录、删除会话
-->
<template>
  <view class="page-wrapper">
    <!-- 对方信息 -->
    <view class="user-card">
      <view class="user-avatar-wrap">
        <image v-if="peerAvatar" class="user-avatar" :src="peerAvatar" mode="aspectFill" />
        <view v-else class="user-avatar user-avatar-placeholder">
          <text class="avatar-text">{{ (peerName || '?')[0] }}</text>
        </view>
      </view>
      <text class="user-name">{{ peerName || '未知用户' }}</text>
    </view>

    <!-- 操作项 -->
    <view class="settings-group">
      <view class="settings-item" @tap="togglePin">
        <text class="settings-label">{{ isPinned ? '取消置顶' : '置顶聊天' }}</text>
        <text class="settings-value">{{ isPinned ? '已置顶' : '' }}</text>
      </view>
    </view>

    <view class="settings-group">
      <view class="settings-item settings-danger" @tap="onClearHistory">
        <text class="settings-label-danger">清空聊天记录</text>
      </view>
      <view class="settings-item settings-danger" @tap="onDeleteConversation">
        <text class="settings-label-danger">删除会话</text>
      </view>
    </view>
  </view>
</template>

<script>
import { onLoad } from '@dcloudio/uni-app'
import { ref } from 'vue'
import { useChatStore } from '@/store/chat'

export default {
  name: 'ChatSettings',
  setup() {
    const chatStore = useChatStore()
    const conversationId = ref(0)
    const peerId = ref(0)
    const peerName = ref('')
    const peerAvatar = ref('')
    const isPinned = ref(false)

    onLoad((query) => {
      conversationId.value = parseInt(query.conversationId) || 0
      peerId.value = parseInt(query.peerId) || 0
      peerName.value = decodeURIComponent(query.peerName || '')
      peerAvatar.value = decodeURIComponent(query.peerAvatar || '')

      const conv = chatStore.conversationList.find(c => c.id === conversationId.value)
      if (conv) {
        isPinned.value = conv.is_pinned
      }
    })

    const togglePin = async () => {
      const newVal = !isPinned.value
      await chatStore.pinConversation(conversationId.value, newVal)
      isPinned.value = newVal
      uni.showToast({ title: newVal ? '已置顶' : '已取消置顶', icon: 'none' })
    }

    const onClearHistory = () => {
      uni.showModal({
        title: '提示',
        content: '确定清空聊天记录？此操作不可恢复',
        success: async (res) => {
          if (res.confirm) {
            await chatStore.clearHistory(conversationId.value)
            uni.showToast({ title: '已清空', icon: 'none' })
          }
        }
      })
    }

    const onDeleteConversation = () => {
      uni.showModal({
        title: '提示',
        content: '确定删除该会话？',
        success: async (res) => {
          if (res.confirm) {
            await chatStore.deleteConversation(conversationId.value)
            uni.navigateBack({ delta: 2 })
          }
        }
      })
    }

    return {
      peerName,
      peerAvatar,
      isPinned,
      togglePin,
      onClearHistory,
      onDeleteConversation
    }
  }
}
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F1F5F9;
}

.user-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48rpx 32rpx;
  background-color: #FFFFFF;
  margin-bottom: 16rpx;
}

.user-avatar-wrap {
  margin-bottom: 20rpx;
}

.user-avatar {
  width: 120rpx;
  height: 120rpx;
  border-radius: 30rpx;
}

.user-avatar-placeholder {
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-text {
  color: #FFFFFF;
  font-size: 48rpx;
  font-weight: 600;
}

.user-name {
  font-size: 34rpx;
  font-weight: 600;
  color: #1E293B;
}

.settings-group {
  background-color: #FFFFFF;
  margin-bottom: 16rpx;
}

.settings-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
}

.settings-item:last-child {
  border-bottom: none;
}

.settings-label {
  font-size: 30rpx;
  color: #1E293B;
}

.settings-value {
  font-size: 26rpx;
  color: #94A3B8;
}

.settings-label-danger {
  font-size: 30rpx;
  color: #EF4444;
}
</style>
