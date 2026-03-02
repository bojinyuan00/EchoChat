<!--
  搜索添加好友页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B
  ui-ux-pro-max 规范：防重复提交 / loading 状态 / 触摸目标 >= 88rpx
-->
<template>
  <view class="page-wrapper">
    <!-- 搜索栏 -->
    <view class="search-bar">
      <view class="search-input-wrap">
        <input
          class="search-input"
          v-model="keyword"
          placeholder="输入用户名或昵称搜索"
          placeholder-style="color: #94A3B8"
          confirm-type="search"
          @confirm="doSearch"
        />
      </view>
      <view class="search-btn" @tap="doSearch">
        <text class="search-btn-text">搜索</text>
      </view>
    </view>

    <!-- 搜索结果 -->
    <view v-if="searched" class="result-section">
      <view class="section-header">
        <text class="section-title">搜索结果</text>
      </view>

      <view v-if="searchLoading" class="loading-wrap">
        <text class="loading-text">搜索中...</text>
      </view>

      <view v-else-if="searchResults.length > 0" class="result-list">
        <view
          v-for="user in searchResults"
          :key="user.user_id"
          class="user-item"
        >
          <view class="avatar" :style="{ backgroundColor: getAvatarColor(user.nickname || user.username) }">
            <text class="avatar-text">{{ getInitial(user.nickname || user.username) }}</text>
          </view>
          <view class="user-info">
            <text class="user-name">{{ user.nickname || user.username }}</text>
            <text class="user-account">@{{ user.username }}</text>
          </view>
          <view v-if="user.is_friend" class="status-tag status-tag--friend">
            <text class="status-tag-text">已是好友</text>
          </view>
          <view
            v-else
            class="add-btn"
            :class="{ 'add-btn--disabled': addingMap[user.user_id] }"
            @tap="showAddDialog(user)"
          >
            <text class="add-btn-text">{{ addingMap[user.user_id] ? '已发送' : '添加' }}</text>
          </view>
        </view>
      </view>

      <view v-else class="empty-state">
        <text class="empty-text">未找到相关用户</text>
      </view>
    </view>

    <!-- 好友推荐 -->
    <view v-if="!searched" class="recommend-section">
      <view class="section-header">
        <text class="section-title">好友推荐</text>
      </view>

      <view v-if="recommendLoading" class="loading-wrap">
        <text class="loading-text">加载中...</text>
      </view>

      <view v-else-if="recommendations.length > 0" class="result-list">
        <view
          v-for="user in recommendations"
          :key="user.user_id"
          class="user-item"
        >
          <view class="avatar" :style="{ backgroundColor: getAvatarColor(user.nickname || user.username) }">
            <text class="avatar-text">{{ getInitial(user.nickname || user.username) }}</text>
          </view>
          <view class="user-info">
            <text class="user-name">{{ user.nickname || user.username }}</text>
            <text class="user-desc" v-if="user.common_count">{{ user.common_count }} 位共同好友</text>
          </view>
          <view
            class="add-btn"
            :class="{ 'add-btn--disabled': addingMap[user.user_id] }"
            @tap="showAddDialog(user)"
          >
            <text class="add-btn-text">{{ addingMap[user.user_id] ? '已发送' : '添加' }}</text>
          </view>
        </view>
      </view>

      <view v-else class="empty-state">
        <text class="empty-text">暂无推荐好友</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import contactApi from '@/api/contact'
import { useContactStore } from '@/store/contact'

const contactStore = useContactStore()

const keyword = ref('')
const searched = ref(false)
const searchLoading = ref(false)
const searchResults = ref([])
const recommendLoading = ref(true)
const recommendations = ref([])
const addingMap = reactive({})

const AVATAR_COLORS = ['#7C3AED', '#2563EB', '#0891B2', '#059669', '#D97706', '#DC2626', '#4F46E5']

const getAvatarColor = (name) => {
  if (!name) return AVATAR_COLORS[0]
  return AVATAR_COLORS[name.charCodeAt(0) % AVATAR_COLORS.length]
}

const getInitial = (name) => {
  if (!name) return '?'
  return name.charAt(0).toUpperCase()
}

const doSearch = async () => {
  const kw = keyword.value.trim()
  if (!kw) {
    uni.showToast({ title: '请输入搜索关键词', icon: 'none' })
    return
  }
  searched.value = true
  searchLoading.value = true
  try {
    const res = await contactApi.searchUsers(kw)
    searchResults.value = res.data || []
  } catch (e) {
    console.error('搜索用户失败', e)
    searchResults.value = []
  }
  searchLoading.value = false
}

const showAddDialog = (user) => {
  if (addingMap[user.user_id]) return
  uni.showModal({
    title: '添加好友',
    editable: true,
    placeholderText: '请输入验证消息（可选）',
    content: '',
    success: async (res) => {
      if (res.confirm) {
        addingMap[user.user_id] = true
        try {
          await contactStore.sendRequest(user.user_id, res.content || '')
          uni.showToast({ title: '申请已发送', icon: 'success' })
        } catch (e) {
          console.error('发送好友申请失败', e)
          addingMap[user.user_id] = false
        }
      }
    }
  })
}

onMounted(async () => {
  try {
    const res = await contactApi.getRecommendFriends()
    recommendations.value = res.data || []
  } catch (e) {
    console.error('获取推荐好友失败', e)
  }
  recommendLoading.value = false
})
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
}

/* 搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  padding: 16rpx 24rpx;
  background-color: #FFFFFF;
  gap: 16rpx;
}

.search-input-wrap {
  flex: 1;
  background-color: #F1F5F9;
  border-radius: 16rpx;
  padding: 0 24rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
}

.search-input {
  width: 100%;
  font-size: 28rpx;
  color: #1E293B;
  height: 72rpx;
}

.search-btn {
  padding: 0 28rpx;
  height: 72rpx;
  background-color: #2563EB;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: opacity 200ms ease;
}

.search-btn:active {
  opacity: 0.8;
}

.search-btn-text {
  font-size: 28rpx;
  color: #FFFFFF;
  font-weight: 500;
}

/* section */
.section-header {
  padding: 20rpx 32rpx;
}

.section-title {
  font-size: 26rpx;
  color: #94A3B8;
  font-weight: 500;
}

/* loading */
.loading-wrap {
  padding: 80rpx 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.loading-text {
  font-size: 28rpx;
  color: #94A3B8;
}

/* 结果列表 */
.result-list {
  background-color: #FFFFFF;
}

.user-item {
  display: flex;
  align-items: center;
  padding: 20rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  gap: 20rpx;
}

.user-item:last-child {
  border-bottom: none;
}

.avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.avatar-text {
  font-size: 32rpx;
  color: #FFFFFF;
  font-weight: 600;
}

.user-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}

.user-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
}

.user-account {
  font-size: 24rpx;
  color: #94A3B8;
}

.user-desc {
  font-size: 24rpx;
  color: #64748B;
}

.status-tag {
  padding: 8rpx 20rpx;
  border-radius: 12rpx;
  flex-shrink: 0;
}

.status-tag--friend {
  background-color: #F1F5F9;
}

.status-tag-text {
  font-size: 24rpx;
  color: #94A3B8;
}

.add-btn {
  padding: 12rpx 28rpx;
  background-color: #2563EB;
  border-radius: 12rpx;
  flex-shrink: 0;
  cursor: pointer;
  transition: opacity 200ms ease;
}

.add-btn:active {
  opacity: 0.8;
}

.add-btn--disabled {
  background-color: #CBD5E1;
  pointer-events: none;
}

.add-btn-text {
  font-size: 24rpx;
  color: #FFFFFF;
  font-weight: 500;
}

/* 空状态 */
.empty-state {
  padding: 80rpx 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-text {
  font-size: 28rpx;
  color: #94A3B8;
}
</style>
