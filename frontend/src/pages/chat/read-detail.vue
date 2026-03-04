<!--
  已读详情页

  展示群消息的已读/未读用户列表
  Tab 切换：已读 / 未读
-->
<template>
  <view class="page-wrapper">
    <!-- Tab 切换 -->
    <view class="tab-bar">
      <view
        class="tab-item"
        :class="{ 'tab-active': activeTab === 'read' }"
        @tap="activeTab = 'read'"
      >
        <text class="tab-text">已读 ({{ readCount }})</text>
      </view>
      <view
        class="tab-item"
        :class="{ 'tab-active': activeTab === 'unread' }"
        @tap="activeTab = 'unread'"
      >
        <text class="tab-text">未读 ({{ unreadCount }})</text>
      </view>
    </view>

    <!-- 用户列表 -->
    <scroll-view scroll-y class="user-list">
      <view v-if="loading" class="empty-state">
        <text class="empty-text">加载中...</text>
      </view>
      <view v-else-if="displayList.length === 0" class="empty-state">
        <text class="empty-text">暂无数据</text>
      </view>
      <view
        v-for="user in displayList"
        :key="user.user_id"
        class="user-item"
      >
        <view class="user-avatar-wrap">
          <image v-if="user.user_avatar" class="user-avatar" :src="user.user_avatar" mode="aspectFill" />
          <view v-else class="user-avatar user-avatar-placeholder">
            <text class="avatar-char">{{ (user.user_nickname || '?')[0] }}</text>
          </view>
        </view>
        <view class="user-info">
          <text class="user-name">{{ user.user_nickname || '未知用户' }}</text>
          <text v-if="user.read_at && activeTab === 'read'" class="user-time">{{ user.read_at }}</text>
        </view>
      </view>
    </scroll-view>
  </view>
</template>

<script>
import { onLoad } from '@dcloudio/uni-app'
import { ref, computed } from 'vue'
import imApi from '@/api/im'

export default {
  name: 'ReadDetail',
  setup() {
    const messageId = ref(0)
    const activeTab = ref('read')
    const loading = ref(false)
    const readList = ref([])
    const unreadList = ref([])
    const readCount = ref(0)
    const totalCount = ref(0)

    const unreadCount = computed(() => totalCount.value - readCount.value)

    const displayList = computed(() => {
      return activeTab.value === 'read' ? readList.value : unreadList.value
    })

    const fetchReadDetail = async () => {
      if (!messageId.value) return
      loading.value = true
      try {
        const res = await imApi.getMessageReadDetail(messageId.value)
        if (res.data) {
          readList.value = res.data.read_list || []
          unreadList.value = res.data.unread_list || []
          readCount.value = res.data.read_count || 0
          totalCount.value = res.data.total_count || 0
        }
      } catch (e) {
        console.warn('[ReadDetail] 获取已读详情失败', e)
        uni.showToast({ title: e?.data?.message || '获取已读详情失败', icon: 'none' })
      } finally {
        loading.value = false
      }
    }

    onLoad((query) => {
      messageId.value = parseInt(query.messageId) || 0
      fetchReadDetail()
    })

    return {
      activeTab, loading, readCount, unreadCount,
      displayList
    }
  }
}
</script>

<style scoped>
.page-wrapper {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: #F8FAFC;
}

/* Tab 切换 */
.tab-bar {
  display: flex;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #E2E8F0;
}
.tab-item {
  flex: 1;
  text-align: center;
  padding: 24rpx 0;
  position: relative;
}
.tab-active {
  border-bottom: 4rpx solid #2563EB;
}
.tab-text {
  font-size: 28rpx;
  color: #475569;
}
.tab-active .tab-text {
  color: #2563EB;
  font-weight: 600;
}

/* 用户列表 */
.user-list {
  flex: 1;
  height: 0;
}
.user-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #F1F5F9;
}
.user-avatar-wrap { flex-shrink: 0; margin-right: 20rpx; }
.user-avatar {
  width: 72rpx;
  height: 72rpx;
  border-radius: 18rpx;
}
.user-avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #2563EB;
}
.avatar-char { color: #FFFFFF; font-size: 28rpx; font-weight: 600; }
.user-info { flex: 1; }
.user-name { font-size: 28rpx; color: #1E293B; }
.user-time { display: block; font-size: 22rpx; color: #94A3B8; margin-top: 4rpx; }

/* 空状态 */
.empty-state { padding: 80rpx 0; text-align: center; }
.empty-text { font-size: 28rpx; color: #94A3B8; }
</style>
