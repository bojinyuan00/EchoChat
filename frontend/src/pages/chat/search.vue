<!--
  消息搜索页

  设计系统：design-system/echochat/MASTER.md
  页面覆盖：design-system/echochat/pages/chat-search.md
  图标方案：@dcloudio/uni-ui uni-icons
  功能：全局消息搜索，结果按会话分组，点击跳转到聊天页
-->
<template>
  <view class="page-wrapper">
    <!-- 搜索栏 -->
    <view class="search-bar">
      <view class="search-input-wrap">
        <uni-icons type="search" size="18" color="#94A3B8" />
        <input
          class="search-input"
          v-model="keyword"
          placeholder="搜索聊天记录"
          placeholder-style="color: #94A3B8"
          confirm-type="search"
          focus
          @confirm="onSearch"
        />
        <view v-if="keyword" class="search-clear" @tap="keyword = ''; groups = []; searched = false">
          <uni-icons type="clear" size="18" color="#94A3B8" />
        </view>
      </view>
      <view class="search-cancel" @tap="goBack">
        <text class="search-cancel-text">取消</text>
      </view>
    </view>

    <!-- 搜索结果 -->
    <scroll-view scroll-y class="result-list">
      <view v-if="searching" class="status-state">
        <text class="status-text">搜索中...</text>
      </view>

      <view v-else-if="searched && groups.length === 0" class="status-state">
        <text class="status-text">未找到相关消息</text>
      </view>

      <!-- 按会话分组展示 -->
      <view v-for="group in groups" :key="group.conversation_id" class="result-group">
        <view class="group-header" @tap="openConversation(group)">
          <view class="group-avatar-wrap">
            <image v-if="group.peer_avatar" class="group-avatar" :src="group.peer_avatar" mode="aspectFill" />
            <view v-else class="group-avatar group-avatar-placeholder">
              <text class="avatar-text">{{ (group.peer_name || '?')[0] }}</text>
            </view>
          </view>
          <text class="group-name">{{ group.peer_name || '未知用户' }}</text>
          <text class="group-count">{{ group.messages.length }} 条相关</text>
        </view>
        <view
          v-for="msg in group.messages"
          :key="msg.id"
          class="result-item"
          @tap="openConversation(group)"
        >
          <text class="result-sender">{{ msg.sender_nickname || '未知' }}：</text>
          <text class="result-content">{{ msg.content }}</text>
          <text class="result-time">{{ msg.created_at }}</text>
        </view>
      </view>
    </scroll-view>
  </view>
</template>

<script>
import { ref, watch } from 'vue'
import imApi from '@/api/im'
import { useChatStore } from '@/store/chat'

export default {
  name: 'ChatSearch',
  setup() {
    const chatStore = useChatStore()
    const keyword = ref('')
    const groups = ref([])
    const searched = ref(false)
    const searching = ref(false)

    let debounceTimer = null
    watch(keyword, (val) => {
      clearTimeout(debounceTimer)
      if (!val.trim()) { groups.value = []; searched.value = false; return }
      debounceTimer = setTimeout(() => onSearch(), 300)
    })

    const onSearch = async () => {
      const kw = keyword.value.trim()
      if (!kw) return

      searching.value = true
      searched.value = true
      try {
        const res = await imApi.searchMessages(kw)
        const list = (res.data && res.data.list) || []
        groups.value = _groupByConversation(list)
      } catch {
        groups.value = []
      } finally {
        searching.value = false
      }
    }

    /** 将扁平消息列表按 conversation_id 分组，附加会话的 peer 信息 */
    const _groupByConversation = (messages) => {
      const map = {}
      for (const msg of messages) {
        const convId = msg.conversation_id
        if (!map[convId]) {
          const conv = chatStore.conversationList.find(c => c.id === convId)
          map[convId] = {
            conversation_id: convId,
            peer_name: conv?.peer_nickname || msg.sender_nickname || '未知',
            peer_avatar: conv?.peer_avatar || msg.sender_avatar || '',
            peer_id: conv?.peer_user_id || 0,
            messages: []
          }
        }
        map[convId].messages.push(msg)
      }
      return Object.values(map)
    }

    const openConversation = (group) => {
      uni.navigateTo({
        url: `/pages/chat/conversation?conversationId=${group.conversation_id}&peerId=${group.peer_id}&peerName=${encodeURIComponent(group.peer_name)}&peerAvatar=${encodeURIComponent(group.peer_avatar)}`
      })
    }

    const goBack = () => {
      uni.navigateBack()
    }

    return { keyword, groups, searched, searching, onSearch, openConversation, goBack }
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
  gap: 12rpx;
}
.search-input { flex: 1; font-size: 28rpx; color: #1E293B; }
.search-clear { min-width: 44rpx; min-height: 44rpx; display: flex; align-items: center; justify-content: center; }

.search-cancel {
  margin-left: 20rpx;
  min-width: 88rpx;
  min-height: 68rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}
.search-cancel-text { font-size: 28rpx; color: #2563EB; }
.search-cancel:active { opacity: 0.6; }

.result-list { height: calc(100vh - 100rpx); }

/* 分组 */
.result-group { margin-bottom: 16rpx; }
.group-header {
  display: flex;
  align-items: center;
  padding: 20rpx 32rpx;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #E2E8F0;
}
.group-header:active { background-color: #F1F5F9; }
.group-avatar-wrap { flex-shrink: 0; margin-right: 16rpx; }
.group-avatar { width: 64rpx; height: 64rpx; border-radius: 16rpx; }
.group-avatar-placeholder { background-color: #2563EB; display: flex; align-items: center; justify-content: center; }
.avatar-text { color: #FFFFFF; font-size: 26rpx; font-weight: 600; }
.group-name { flex: 1; font-size: 30rpx; font-weight: 500; color: #1E293B; }
.group-count { font-size: 24rpx; color: #94A3B8; }

/* 消息条目 */
.result-item {
  display: flex;
  align-items: center;
  padding: 16rpx 32rpx 16rpx 112rpx;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #F8FAFC;
}
.result-item:active { background-color: #F1F5F9; }
.result-sender { font-size: 26rpx; color: #64748B; flex-shrink: 0; }
.result-content { flex: 1; font-size: 26rpx; color: #1E293B; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.result-time { font-size: 22rpx; color: #94A3B8; margin-left: 12rpx; flex-shrink: 0; }

/* 状态 */
.status-state { display: flex; align-items: center; justify-content: center; padding-top: 200rpx; }
.status-text { font-size: 28rpx; color: #94A3B8; }
</style>
