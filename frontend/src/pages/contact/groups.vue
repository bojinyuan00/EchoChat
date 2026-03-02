<!--
  好友分组管理页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Text #1E293B
  ui-ux-pro-max 规范：防重复提交 / 确认弹窗 / 过渡动画
-->
<template>
  <view class="page-wrapper">
    <!-- 新建分组 -->
    <view class="create-bar" @tap="handleCreate">
      <text class="create-icon">+</text>
      <text class="create-text">新建分组</text>
    </view>

    <!-- 骨架屏 -->
    <view v-if="loading" class="skeleton-list">
      <view v-for="i in 3" :key="i" class="skeleton-item">
        <view class="skeleton-info">
          <view class="skeleton-line skeleton-line--name"></view>
          <view class="skeleton-line skeleton-line--count"></view>
        </view>
      </view>
    </view>

    <!-- 默认分组 -->
    <view v-else class="group-list">
      <view class="group-item group-item--default">
        <view class="group-left">
          <view class="group-icon group-icon--default">
            <text class="group-icon-text">&#9733;</text>
          </view>
          <view class="group-info">
            <text class="group-name">默认分组</text>
            <text class="group-count">系统默认</text>
          </view>
        </view>
      </view>

      <!-- 自定义分组 -->
      <view
        v-for="group in contactStore.groups"
        :key="group.id"
        class="group-item"
      >
        <view class="group-left">
          <view class="group-icon">
            <text class="group-icon-text">&#9776;</text>
          </view>
          <view class="group-info">
            <text class="group-name">{{ group.name }}</text>
            <text class="group-count">{{ group.friend_count || 0 }} 位好友</text>
          </view>
        </view>
        <view class="group-actions">
          <view class="icon-btn" @tap="handleEdit(group)">
            <text class="icon-btn-text">&#9998;</text>
          </view>
          <view class="icon-btn icon-btn--danger" @tap="handleDelete(group)">
            <text class="icon-btn-text icon-btn-text--danger">&#10005;</text>
          </view>
        </view>
      </view>

      <!-- 空分组提示 -->
      <view v-if="contactStore.groups.length === 0" class="empty-hint">
        <text class="empty-hint-text">还没有自定义分组，点击上方按钮创建</text>
      </view>
    </view>
  </view>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useContactStore } from '@/store/contact'

const contactStore = useContactStore()
const loading = ref(true)
const processing = ref(false)

onMounted(async () => {
  try {
    await contactStore.fetchGroups()
  } catch (e) {
    console.error('获取分组失败', e)
  }
  loading.value = false
})

const handleCreate = () => {
  if (processing.value) return
  uni.showModal({
    title: '新建分组',
    editable: true,
    placeholderText: '请输入分组名称',
    content: '',
    success: async (res) => {
      if (res.confirm && res.content?.trim()) {
        processing.value = true
        try {
          await contactStore.createGroup(res.content.trim())
          uni.showToast({ title: '创建成功', icon: 'success' })
        } catch (e) {
          console.error('创建分组失败', e)
        } finally {
          processing.value = false
        }
      }
    }
  })
}

const handleEdit = (group) => {
  if (processing.value) return
  uni.showModal({
    title: '修改分组名',
    editable: true,
    placeholderText: '请输入新的分组名称',
    content: group.name,
    success: async (res) => {
      if (res.confirm && res.content?.trim() && res.content.trim() !== group.name) {
        processing.value = true
        try {
          await contactStore.updateGroup(group.id, res.content.trim())
          uni.showToast({ title: '已修改', icon: 'success' })
        } catch (e) {
          console.error('修改分组失败', e)
        } finally {
          processing.value = false
        }
      }
    }
  })
}

const handleDelete = (group) => {
  if (processing.value) return
  uni.showModal({
    title: '确认删除',
    content: `确定要删除分组"${group.name}"吗？分组内的好友将移至默认分组。`,
    confirmColor: '#EF4444',
    success: async (res) => {
      if (res.confirm) {
        processing.value = true
        try {
          await contactStore.deleteGroup(group.id)
          uni.showToast({ title: '已删除', icon: 'none' })
        } catch (e) {
          console.error('删除分组失败', e)
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

/* 新建分组 */
.create-bar {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12rpx;
  padding: 28rpx 32rpx;
  background-color: #FFFFFF;
  margin-bottom: 16rpx;
  cursor: pointer;
  transition: background-color 200ms ease;
}

.create-bar:active {
  background-color: #F8FAFC;
}

.create-icon {
  font-size: 36rpx;
  color: #2563EB;
  font-weight: 300;
}

.create-text {
  font-size: 30rpx;
  color: #2563EB;
  font-weight: 500;
}

/* 骨架屏 */
.skeleton-list {
  background-color: #FFFFFF;
}

.skeleton-item {
  padding: 24rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
}

.skeleton-info {
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

.skeleton-line--count {
  width: 25%;
  height: 22rpx;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* 分组列表 */
.group-list {
  background-color: #FFFFFF;
}

.group-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
}

.group-item:last-child {
  border-bottom: none;
}

.group-left {
  display: flex;
  align-items: center;
  gap: 20rpx;
}

.group-icon {
  width: 64rpx;
  height: 64rpx;
  border-radius: 16rpx;
  background-color: #DBEAFE;
  display: flex;
  align-items: center;
  justify-content: center;
}

.group-icon--default {
  background-color: #FEF3C7;
}

.group-icon-text {
  font-size: 28rpx;
}

.group-info {
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.group-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
}

.group-count {
  font-size: 24rpx;
  color: #94A3B8;
}

.group-actions {
  display: flex;
  gap: 16rpx;
}

.icon-btn {
  width: 64rpx;
  height: 64rpx;
  border-radius: 50%;
  background-color: #F1F5F9;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background-color 200ms ease;
}

.icon-btn:active {
  background-color: #E2E8F0;
}

.icon-btn--danger:active {
  background-color: #FEE2E2;
}

.icon-btn-text {
  font-size: 24rpx;
  color: #64748B;
}

.icon-btn-text--danger {
  color: #EF4444;
}

/* 空提示 */
.empty-hint {
  padding: 40rpx 32rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-hint-text {
  font-size: 26rpx;
  color: #94A3B8;
}
</style>
