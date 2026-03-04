<!--
  创建群聊页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Surface #F1F5F9 / Text #1E293B
  ui-ux-pro-max 规范：防重复提交 / loading 状态 / 触摸目标 >= 88rpx
-->
<template>
  <view class="page-wrapper">
    <!-- 群名称输入区 -->
    <view class="section">
      <view class="section-header">
        <text class="section-title">群名称</text>
      </view>
      <view class="name-input-wrap">
        <input
          class="name-input"
          v-model="groupName"
          placeholder="请输入群聊名称"
          placeholder-style="color: #94A3B8"
          maxlength="30"
        />
        <text class="name-count">{{ groupName.length }}/30</text>
      </view>
    </view>

    <!-- 已选好友展示区 -->
    <view v-if="selectedList.length > 0" class="section">
      <view class="section-header">
        <text class="section-title">已选好友（{{ selectedList.length }}）</text>
      </view>
      <scroll-view class="selected-scroll" scroll-x>
        <view class="selected-list">
          <view
            v-for="friend in selectedList"
            :key="friend.user_id"
            class="selected-item"
            @tap="toggleSelect(friend)"
          >
            <view class="selected-avatar" :style="{ backgroundColor: getAvatarColor(friend.nickname || friend.username) }">
              <text class="selected-avatar-text">{{ getInitial(friend.nickname || friend.username) }}</text>
              <view class="selected-remove">
                <text class="selected-remove-text">×</text>
              </view>
            </view>
            <text class="selected-name">{{ friend.remark || friend.nickname || friend.username }}</text>
          </view>
        </view>
      </scroll-view>
    </view>

    <!-- 搜索好友 -->
    <view class="section">
      <view class="section-header">
        <text class="section-title">选择好友</text>
      </view>
      <view class="search-bar">
        <uni-icons type="search" size="18" color="#94A3B8" />
        <input
          class="search-input"
          v-model="searchKeyword"
          placeholder="搜索好友昵称"
          placeholder-style="color: #94A3B8"
        />
        <view v-if="searchKeyword" class="search-clear" @tap="searchKeyword = ''">
          <uni-icons type="clear" size="16" color="#94A3B8" />
        </view>
      </view>
    </view>

    <!-- 好友列表 -->
    <view v-if="loading" class="skeleton-list">
      <view v-for="i in 5" :key="i" class="skeleton-item">
        <view class="skeleton-check"></view>
        <view class="skeleton-avatar"></view>
        <view class="skeleton-info">
          <view class="skeleton-line skeleton-line--name"></view>
        </view>
      </view>
    </view>

    <view v-else-if="filteredFriends.length > 0" class="friend-list">
      <view
        v-for="friend in filteredFriends"
        :key="friend.user_id"
        class="friend-item"
        @tap="toggleSelect(friend)"
      >
        <view class="checkbox" :class="{ 'checkbox--checked': isSelected(friend.user_id) }">
          <text v-if="isSelected(friend.user_id)" class="checkbox-icon">✓</text>
        </view>
        <view class="avatar" :style="{ backgroundColor: getAvatarColor(friend.nickname || friend.username) }">
          <text class="avatar-text">{{ getInitial(friend.nickname || friend.username) }}</text>
        </view>
        <view class="friend-info">
          <text class="friend-name">{{ friend.remark || friend.nickname || friend.username }}</text>
          <text v-if="friend.remark" class="friend-account">{{ friend.nickname || friend.username }}</text>
        </view>
      </view>
    </view>

    <view v-else class="empty-state">
      <text class="empty-text">{{ searchKeyword ? '未找到匹配的好友' : '暂无好友' }}</text>
    </view>

    <!-- 底部占位 -->
    <view class="bottom-spacer"></view>

    <!-- 底部固定创建按钮 -->
    <view class="bottom-bar">
      <view
        class="create-btn"
        :class="{ 'create-btn--disabled': !canCreate || submitting }"
        @tap="handleCreate"
      >
        <text class="create-btn-text">
          {{ submitting ? '创建中...' : `创建群聊${selectedList.length > 0 ? '（' + selectedList.length + '人）' : ''}` }}
        </text>
      </view>
    </view>
  </view>
</template>

<script>
import { ref, computed } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { useContactStore } from '@/store/contact'
import { useGroupStore } from '@/store/group'
import { getAvatarColor, getInitial } from '@/utils/avatar'

export default {
  name: 'GroupCreate',
  setup() {
    const contactStore = useContactStore()
    const groupStore = useGroupStore()

    const groupName = ref('')
    const searchKeyword = ref('')
    const selectedIds = ref({})
    const loading = ref(true)
    const submitting = ref(false)

    /** 已选中的好友列表 */
    const selectedList = computed(() => {
      return contactStore.friendList.filter(f => selectedIds.value[f.user_id])
    })

    /** 搜索过滤后的好友列表 */
    const filteredFriends = computed(() => {
      const kw = searchKeyword.value.trim().toLowerCase()
      if (!kw) return contactStore.friendList
      return contactStore.friendList.filter(f => {
        const nickname = (f.nickname || '').toLowerCase()
        const username = (f.username || '').toLowerCase()
        const remark = (f.remark || '').toLowerCase()
        return nickname.includes(kw) || username.includes(kw) || remark.includes(kw)
      })
    })

    /** 是否满足创建条件：群名非空 + 至少选2人 */
    const canCreate = computed(() => {
      return groupName.value.trim().length > 0 && selectedList.value.length >= 2
    })

    /** 判断好友是否已选中 */
    const isSelected = (userId) => {
      return !!selectedIds.value[userId]
    }

    /** 切换好友选中状态 */
    const toggleSelect = (friend) => {
      const newMap = { ...selectedIds.value }
      if (newMap[friend.user_id]) {
        delete newMap[friend.user_id]
      } else {
        newMap[friend.user_id] = true
      }
      selectedIds.value = newMap
    }

    /** 提交创建群聊 */
    const handleCreate = async () => {
      if (!canCreate.value || submitting.value) return

      submitting.value = true
      try {
        const memberIds = selectedList.value.map(f => f.user_id)
        const result = await groupStore.createGroup(groupName.value.trim(), memberIds)

        uni.showToast({ title: '群聊创建成功', icon: 'success' })

        setTimeout(() => {
          if (result && result.conversation_id) {
            uni.redirectTo({
              url: `/pages/group/conversation?conversationId=${result.conversation_id}&groupId=${result.group_id || 0}&peerName=${encodeURIComponent(groupName.value.trim())}`
            })
          } else {
            uni.navigateBack()
          }
        }, 500)
      } catch (e) {
        console.error('创建群聊失败', e)
        uni.showToast({
          title: e?.data?.message || '创建群聊失败',
          icon: 'none'
        })
      } finally {
        submitting.value = false
      }
    }

    onLoad(async () => {
      try {
        await contactStore.fetchFriends()
      } catch (e) {
        console.error('获取好友列表失败', e)
      }
      loading.value = false
    })

    return {
      groupName,
      searchKeyword,
      selectedList,
      filteredFriends,
      canCreate,
      loading,
      submitting,
      isSelected,
      toggleSelect,
      handleCreate,
      getAvatarColor,
      getInitial
    }
  }
}
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
  padding-bottom: env(safe-area-inset-bottom);
}

/* 区块通用 */
.section {
  margin-bottom: 16rpx;
}

.section-header {
  padding: 20rpx 32rpx 12rpx;
}

.section-title {
  font-size: 26rpx;
  color: #94A3B8;
  font-weight: 500;
}

/* 群名称输入 */
.name-input-wrap {
  display: flex;
  align-items: center;
  background-color: #FFFFFF;
  padding: 0 32rpx;
  height: 96rpx;
}

.name-input {
  flex: 1;
  font-size: 30rpx;
  color: #1E293B;
  height: 96rpx;
}

.name-count {
  font-size: 24rpx;
  color: #94A3B8;
  flex-shrink: 0;
  margin-left: 16rpx;
}

/* 已选好友区 */
.selected-scroll {
  white-space: nowrap;
  background-color: #FFFFFF;
  padding: 20rpx 32rpx;
}

.selected-list {
  display: flex;
  gap: 24rpx;
}

.selected-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100rpx;
  flex-shrink: 0;
}

.selected-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.selected-avatar-text {
  font-size: 32rpx;
  color: #FFFFFF;
  font-weight: 600;
}

.selected-remove {
  position: absolute;
  top: -6rpx;
  right: -6rpx;
  width: 32rpx;
  height: 32rpx;
  border-radius: 50%;
  background-color: #EF4444;
  display: flex;
  align-items: center;
  justify-content: center;
}

.selected-remove-text {
  font-size: 20rpx;
  color: #FFFFFF;
  font-weight: 600;
  line-height: 1;
}

.selected-name {
  font-size: 22rpx;
  color: #475569;
  margin-top: 8rpx;
  max-width: 100rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: center;
}

/* 搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  background-color: #FFFFFF;
  padding: 0 24rpx;
  height: 80rpx;
  gap: 12rpx;
}

.search-input {
  flex: 1;
  font-size: 28rpx;
  color: #1E293B;
  height: 80rpx;
}

.search-clear {
  flex-shrink: 0;
  padding: 8rpx;
}

/* 骨架屏 */
.skeleton-list {
  background-color: #FFFFFF;
}

.skeleton-item {
  display: flex;
  align-items: center;
  padding: 20rpx 32rpx;
  gap: 20rpx;
  border-bottom: 1rpx solid #F1F5F9;
}

.skeleton-check {
  width: 40rpx;
  height: 40rpx;
  border-radius: 8rpx;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  flex-shrink: 0;
}

.skeleton-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 50%;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  flex-shrink: 0;
}

.skeleton-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10rpx;
}

.skeleton-line {
  border-radius: 8rpx;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

.skeleton-line--name {
  width: 40%;
  height: 28rpx;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* 好友列表 */
.friend-list {
  background-color: #FFFFFF;
}

.friend-item {
  display: flex;
  align-items: center;
  padding: 20rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  gap: 20rpx;
  cursor: pointer;
  transition: background-color 150ms ease;
}

.friend-item:active {
  background-color: #F8FAFC;
}

.friend-item:last-child {
  border-bottom: none;
}

/* 复选框 */
.checkbox {
  width: 40rpx;
  height: 40rpx;
  border-radius: 8rpx;
  border: 2rpx solid #CBD5E1;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: all 200ms ease;
}

.checkbox--checked {
  background-color: #2563EB;
  border-color: #2563EB;
}

.checkbox-icon {
  font-size: 24rpx;
  color: #FFFFFF;
  font-weight: 700;
  line-height: 1;
}

/* 头像 */
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

/* 好友信息 */
.friend-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  min-width: 0;
}

.friend-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.friend-account {
  font-size: 24rpx;
  color: #94A3B8;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

/* 底部占位 */
.bottom-spacer {
  height: 140rpx;
}

/* 底部固定按钮 */
.bottom-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 16rpx 32rpx;
  padding-bottom: calc(16rpx + env(safe-area-inset-bottom));
  background-color: #FFFFFF;
  box-shadow: 0 -2rpx 12rpx rgba(0, 0, 0, 0.04);
}

.create-btn {
  height: 88rpx;
  border-radius: 16rpx;
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: opacity 200ms ease;
}

.create-btn:active {
  opacity: 0.85;
}

.create-btn--disabled {
  background-color: #CBD5E1;
  pointer-events: none;
}

.create-btn-text {
  font-size: 30rpx;
  color: #FFFFFF;
  font-weight: 600;
}
</style>
