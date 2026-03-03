<!--
  消息搜索页

  设计系统：design-system/echochat/MASTER.md
  功能：全局消息搜索，结果按会话分组，点击跳转到聊天页
-->
<template>
  <view class="page-wrapper">
    <!-- 搜索栏 -->
    <view class="search-bar">
      <view class="search-input-wrap">
        <text class="search-icon">&#128269;</text>
        <input
          class="search-input"
          v-model="keyword"
          placeholder="搜索聊天记录"
          placeholder-style="color: #94A3B8"
          confirm-type="search"
          focus
          @confirm="onSearch"
        />
      </view>
      <text class="search-cancel" @tap="goBack">取消</text>
    </view>

    <!-- 搜索结果 -->
    <scroll-view scroll-y class="result-list">
      <!-- 空状态 -->
      <view v-if="searched && results.length === 0" class="empty-state">
        <text class="empty-text">未找到相关消息</text>
      </view>

      <!-- 结果列表 -->
      <view
        v-for="item in results"
        :key="item.id"
        class="result-item"
        @tap="openChat(item)"
      >
        <view class="result-avatar-wrap">
          <image v-if="item.sender_avatar" class="result-avatar" :src="item.sender_avatar" mode="aspectFill" />
          <view v-else class="result-avatar result-avatar-placeholder">
            <text class="avatar-text">{{ (item.sender_nickname || '?')[0] }}</text>
          </view>
        </view>
        <view class="result-info">
          <text class="result-name">{{ item.sender_nickname || '未知用户' }}</text>
          <text class="result-content">{{ item.content }}</text>
          <text class="result-time">{{ item.created_at }}</text>
        </view>
      </view>
    </scroll-view>
  </view>
</template>

<script>
import { ref } from 'vue'
import imApi from '@/api/im'

export default {
  name: 'ChatSearch',
  setup() {
    const keyword = ref('')
    const results = ref([])
    const searched = ref(false)

    const onSearch = async () => {
      const kw = keyword.value.trim()
      if (!kw) return

      searched.value = true
      try {
        const res = await imApi.searchMessages(kw)
        results.value = (res.data && res.data.list) || []
      } catch {
        results.value = []
      }
    }

    const openChat = (item) => {
      uni.navigateTo({
        url: `/pages/chat/conversation?conversationId=${item.conversation_id}&peerId=${item.sender_id}&peerName=${encodeURIComponent(item.sender_nickname || '')}&peerAvatar=${encodeURIComponent(item.sender_avatar || '')}`
      })
    }

    const goBack = () => {
      uni.navigateBack()
    }

    return {
      keyword,
      results,
      searched,
      onSearch,
      openChat,
      goBack
    }
  }
}
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
}

.search-bar {
  display: flex;
  align-items: center;
  padding: 16rpx 24rpx;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #E2E8F0;
}

.search-input-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  background-color: #F1F5F9;
  border-radius: 36rpx;
  padding: 0 24rpx;
  height: 68rpx;
}

.search-icon {
  font-size: 28rpx;
  margin-right: 12rpx;
  color: #94A3B8;
}

.search-input {
  flex: 1;
  font-size: 28rpx;
  color: #1E293B;
}

.search-cancel {
  margin-left: 20rpx;
  font-size: 28rpx;
  color: #2563EB;
}

.result-list {
  height: calc(100vh - 100rpx);
}

.result-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #F1F5F9;
}

.result-avatar-wrap {
  flex-shrink: 0;
  margin-right: 20rpx;
}

.result-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 20rpx;
}

.result-avatar-placeholder {
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-text {
  color: #FFFFFF;
  font-size: 28rpx;
  font-weight: 600;
}

.result-info {
  flex: 1;
  overflow: hidden;
}

.result-name {
  font-size: 28rpx;
  font-weight: 500;
  color: #1E293B;
  margin-bottom: 4rpx;
}

.result-content {
  font-size: 26rpx;
  color: #64748B;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 4rpx;
}

.result-time {
  font-size: 22rpx;
  color: #94A3B8;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding-top: 200rpx;
}

.empty-text {
  font-size: 28rpx;
  color: #94A3B8;
}
</style>
