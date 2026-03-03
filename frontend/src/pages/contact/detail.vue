<!--
  好友详情页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B / Danger #EF4444
  ui-ux-pro-max 规范：确认弹窗 / 防重复提交 / 过渡动画
-->
<template>
  <view class="page-wrapper">
    <view v-if="friend" class="detail-content">
      <!-- 头像卡片 -->
      <view class="profile-card">
        <view class="avatar-lg" :style="{ backgroundColor: getAvatarColor(friend.nickname || friend.username) }">
          <text class="avatar-text-lg">{{ getInitial(friend.remark || friend.nickname || friend.username) }}</text>
        </view>
        <text class="profile-name">{{ friend.remark || friend.nickname || friend.username }}</text>
        <text class="profile-username">@{{ friend.username }}</text>
        <view class="online-badge" :class="friend.is_online ? 'online-badge--on' : 'online-badge--off'">
          <view class="online-badge-dot"></view>
          <text class="online-badge-text">{{ friend.is_online ? '在线' : '离线' }}</text>
        </view>
      </view>

      <!-- 信息编辑 -->
      <view class="info-section">
        <view class="info-item" @tap="editRemark">
          <text class="info-label">备注名</text>
          <view class="info-value-wrap">
            <text class="info-value">{{ friend.remark || '未设置' }}</text>
            <text class="arrow">&#8250;</text>
          </view>
        </view>
        <view class="info-item" @tap="selectGroup">
          <text class="info-label">所属分组</text>
          <view class="info-value-wrap">
            <text class="info-value">{{ currentGroupName }}</text>
            <text class="arrow">&#8250;</text>
          </view>
        </view>
      </view>

      <!-- 操作按钮 -->
      <view class="action-section">
        <view class="action-btn action-btn--primary" @tap="sendMessage">
          <text class="action-btn-text action-btn-text--primary">发消息</text>
        </view>
      </view>

      <view class="danger-section">
        <view class="danger-item" @tap="handleBlock">
          <text class="danger-text">拉黑该用户</text>
        </view>
        <view class="danger-item danger-item--last" @tap="handleDelete">
          <text class="danger-text">删除好友</text>
        </view>
      </view>
    </view>

    <!-- 未找到好友 -->
    <view v-else-if="!loading" class="empty-state">
      <text class="empty-title">未找到好友信息</text>
    </view>
  </view>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useContactStore } from '@/store/contact'
import { getAvatarColor, getInitial } from '@/utils/avatar'

const contactStore = useContactStore()
const loading = ref(true)
const friend = ref(null)
const userId = ref(0)
const processing = ref(false)

const currentGroupName = computed(() => {
  if (!friend.value || !friend.value.group_id) return '默认分组'
  const group = contactStore.groups.find(g => g.id === friend.value.group_id)
  return group ? group.name : '默认分组'
})

onMounted(async () => {
  const pages = getCurrentPages()
  const currentPage = pages[pages.length - 1]
  userId.value = Number(currentPage.options?.userId || 0)

  if (contactStore.friendList.length === 0) {
    await contactStore.fetchFriends()
  }
  if (contactStore.groups.length === 0) {
    await contactStore.fetchGroups()
  }

  friend.value = contactStore.friendList.find(f => f.user_id === userId.value) || null
  loading.value = false
})

const editRemark = () => {
  uni.showModal({
    title: '修改备注',
    editable: true,
    placeholderText: '请输入备注名',
    content: friend.value?.remark || '',
    success: async (res) => {
      if (res.confirm && res.content !== undefined) {
        try {
          await contactStore.updateRemark(userId.value, res.content.trim())
          friend.value = contactStore.friendList.find(f => f.user_id === userId.value)
          uni.showToast({ title: '已更新', icon: 'success' })
        } catch (e) {
          console.error('更新备注失败', e)
        }
      }
    }
  })
}

const selectGroup = () => {
  const groups = [{ id: null, name: '默认分组' }, ...contactStore.groups]
  const names = groups.map(g => g.name)

  uni.showActionSheet({
    itemList: names,
    success: async (res) => {
      const selectedGroup = groups[res.tapIndex]
      try {
        await contactStore.moveToGroup(userId.value, selectedGroup.id)
        friend.value = contactStore.friendList.find(f => f.user_id === userId.value)
        uni.showToast({ title: '已移动', icon: 'success' })
      } catch (e) {
        console.error('移动分组失败', e)
      }
    }
  })
}

const sendMessage = () => {
  const f = friend.value
  if (!f) return
  uni.navigateTo({
    url: `/pages/chat/conversation?conversationId=0&peerId=${f.user_id}&peerName=${encodeURIComponent(f.remark || f.nickname || f.username)}&peerAvatar=${encodeURIComponent(f.avatar || '')}`
  })
}

const handleBlock = () => {
  if (processing.value) return
  uni.showModal({
    title: '确认拉黑',
    content: `确定要拉黑 ${friend.value.remark || friend.value.nickname || friend.value.username} 吗？拉黑后将自动解除好友关系。`,
    confirmColor: '#EF4444',
    success: async (res) => {
      if (res.confirm) {
        processing.value = true
        try {
          await contactStore.blockUser(userId.value)
          uni.showToast({ title: '已拉黑', icon: 'none' })
          setTimeout(() => uni.navigateBack(), 800)
        } catch (e) {
          console.error('拉黑失败', e)
        } finally {
          processing.value = false
        }
      }
    }
  })
}

const handleDelete = () => {
  if (processing.value) return
  uni.showModal({
    title: '确认删除',
    content: `确定要删除好友 ${friend.value.remark || friend.value.nickname || friend.value.username} 吗？删除后需要重新申请添加。`,
    confirmColor: '#EF4444',
    success: async (res) => {
      if (res.confirm) {
        processing.value = true
        try {
          await contactStore.deleteFriend(userId.value)
          uni.showToast({ title: '已删除', icon: 'none' })
          setTimeout(() => uni.navigateBack(), 800)
        } catch (e) {
          console.error('删除好友失败', e)
        } finally {
          processing.value = false
        }
      }
    }
  })
}
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F8FAFC;
}

/* 头像卡片 */
.profile-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48rpx 32rpx 40rpx;
  background-color: #FFFFFF;
}

.avatar-lg {
  width: 140rpx;
  height: 140rpx;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 20rpx;
}

.avatar-text-lg {
  font-size: 56rpx;
  color: #FFFFFF;
  font-weight: 600;
}

.profile-name {
  font-size: 36rpx;
  color: #1E293B;
  font-weight: 700;
  margin-bottom: 6rpx;
}

.profile-username {
  font-size: 26rpx;
  color: #94A3B8;
  margin-bottom: 16rpx;
}

.online-badge {
  display: flex;
  align-items: center;
  gap: 8rpx;
  padding: 8rpx 20rpx;
  border-radius: 20rpx;
}

.online-badge--on {
  background-color: #ECFDF5;
}

.online-badge--off {
  background-color: #F1F5F9;
}

.online-badge-dot {
  width: 12rpx;
  height: 12rpx;
  border-radius: 50%;
}

.online-badge--on .online-badge-dot {
  background-color: #10B981;
}

.online-badge--off .online-badge-dot {
  background-color: #CBD5E1;
}

.online-badge-text {
  font-size: 22rpx;
  font-weight: 500;
}

.online-badge--on .online-badge-text {
  color: #059669;
}

.online-badge--off .online-badge-text {
  color: #94A3B8;
}

/* 信息编辑 */
.info-section {
  margin-top: 16rpx;
  background-color: #FFFFFF;
}

.info-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 28rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  cursor: pointer;
  transition: background-color 200ms ease;
}

.info-item:active {
  background-color: #F8FAFC;
}

.info-item:last-child {
  border-bottom: none;
}

.info-label {
  font-size: 30rpx;
  color: #1E293B;
}

.info-value-wrap {
  display: flex;
  align-items: center;
  gap: 8rpx;
}

.info-value {
  font-size: 28rpx;
  color: #94A3B8;
}

.arrow {
  font-size: 28rpx;
  color: #CBD5E1;
}

/* 操作按钮 */
.action-section {
  margin-top: 32rpx;
  padding: 0 32rpx;
}

.action-btn {
  width: 100%;
  height: 88rpx;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: opacity 200ms ease;
}

.action-btn:active {
  opacity: 0.8;
}

.action-btn--primary {
  background-color: #2563EB;
}

.action-btn-text--primary {
  font-size: 30rpx;
  font-weight: 600;
  color: #FFFFFF;
}

/* 危险操作 */
.danger-section {
  margin-top: 32rpx;
  background-color: #FFFFFF;
}

.danger-item {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 28rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  cursor: pointer;
  transition: background-color 200ms ease;
}

.danger-item:active {
  background-color: #FEF2F2;
}

.danger-item--last {
  border-bottom: none;
}

.danger-text {
  font-size: 30rpx;
  color: #EF4444;
  font-weight: 500;
}

/* 空状态 */
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding-top: 300rpx;
}

.empty-title {
  font-size: 32rpx;
  color: #64748B;
}
</style>
