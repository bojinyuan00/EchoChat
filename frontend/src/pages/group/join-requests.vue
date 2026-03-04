<!--
  入群申请审批页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Surface #F1F5F9 / Text #1E293B
  功能：查看待审批/已处理的入群申请，通过或拒绝操作
-->
<template>
  <view class="page-wrapper">
    <!-- 骨架屏 -->
    <view v-if="loading" class="skeleton-list">
      <view v-for="i in 4" :key="i" class="skeleton-item">
        <view class="skeleton-avatar"></view>
        <view class="skeleton-info">
          <view class="skeleton-line skeleton-line--name"></view>
          <view class="skeleton-line skeleton-line--msg"></view>
        </view>
        <view class="skeleton-actions">
          <view class="skeleton-btn"></view>
          <view class="skeleton-btn"></view>
        </view>
      </view>
    </view>

    <!-- 内容区域 -->
    <view v-else>
      <!-- 待审批区域 -->
      <view v-if="pendingList.length > 0" class="section">
        <view class="section-header">
          <text class="section-title">待审批（{{ pendingList.length }}）</text>
        </view>
        <view class="request-list">
          <view
            v-for="req in pendingList"
            :key="req.id"
            class="request-item"
          >
            <image
              v-if="req.avatar"
              class="request-avatar"
              :src="req.avatar"
              mode="aspectFill"
            />
            <view
              v-else
              class="request-avatar request-avatar-placeholder"
              :style="{ backgroundColor: getAvatarColor(req.nickname || req.username) }"
            >
              <text class="avatar-char">{{ getInitial(req.nickname || req.username) }}</text>
            </view>

            <view class="request-info">
              <text class="request-name">{{ req.nickname || req.username }}</text>
              <text class="request-msg" v-if="req.message">{{ req.message }}</text>
              <text class="request-msg" v-else>申请加入群聊</text>
              <text class="request-time">{{ formatTime(req.created_at) }}</text>
            </view>

            <view class="request-actions">
              <view
                class="btn btn--approve"
                :class="{ 'btn--disabled': processingMap[req.id] }"
                @tap="handleApprove(req)"
              >
                <text class="btn-text">通过</text>
              </view>
              <view
                class="btn btn--reject"
                :class="{ 'btn--disabled': processingMap[req.id] }"
                @tap="handleReject(req)"
              >
                <text class="btn-text btn-text--reject">拒绝</text>
              </view>
            </view>
          </view>
        </view>
      </view>

      <!-- 已处理区域 -->
      <view v-if="processedList.length > 0" class="section">
        <view class="section-header">
          <text class="section-title">已处理（{{ processedList.length }}）</text>
        </view>
        <view class="request-list">
          <view
            v-for="req in processedList"
            :key="req.id"
            class="request-item"
          >
            <image
              v-if="req.avatar"
              class="request-avatar"
              :src="req.avatar"
              mode="aspectFill"
            />
            <view
              v-else
              class="request-avatar request-avatar-placeholder"
              :style="{ backgroundColor: getAvatarColor(req.nickname || req.username) }"
            >
              <text class="avatar-char">{{ getInitial(req.nickname || req.username) }}</text>
            </view>

            <view class="request-info">
              <text class="request-name">{{ req.nickname || req.username }}</text>
              <text class="request-msg" v-if="req.message">{{ req.message }}</text>
              <text class="request-msg" v-else>申请加入群聊</text>
              <text class="request-time">{{ formatTime(req.created_at) }}</text>
            </view>

            <view class="status-tag" :class="req.status === 1 ? 'status-tag--approved' : 'status-tag--rejected'">
              <text class="status-tag-text" :class="req.status === 1 ? 'status-tag-text--approved' : 'status-tag-text--rejected'">
                {{ req.status === 1 ? '已通过' : '已拒绝' }}
              </text>
            </view>
          </view>
        </view>
      </view>

      <!-- 空状态 -->
      <view v-if="pendingList.length === 0 && processedList.length === 0" class="empty-state">
        <uni-icons type="person" size="48" color="#CBD5E1" />
        <text class="empty-title">暂无入群申请</text>
        <text class="empty-desc">当有人申请加入群聊时，会在这里显示</text>
      </view>
    </view>
  </view>
</template>

<script>
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { ref, computed, reactive } from 'vue'
import { useGroupStore } from '@/store/group'
import { getAvatarColor, getInitial } from '@/utils/avatar'
import wsService from '@/services/websocket'

export default {
  name: 'GroupJoinRequests',
  setup() {
    const groupStore = useGroupStore()

    const groupId = ref(0)
    const loading = ref(true)
    const processingMap = reactive({})
    /** 本地维护的已处理申请（审批后从 store 移除但需要页面展示） */
    const localProcessed = ref([])

    /** 待审批列表：status === 0 */
    const pendingList = computed(() => {
      return groupStore.joinRequests.filter(r => r.status === 0)
    })

    /**
     * 已处理列表：合并后端返回的已处理申请 + 本地刚审批的
     * 后端 fetch 可能已返回 status !== 0 的，加上本地审批后保存的
     */
    const processedList = computed(() => {
      const fromStore = groupStore.joinRequests.filter(r => r.status !== 0)
      const processedIds = new Set(fromStore.map(r => r.id))
      const fromLocal = localProcessed.value.filter(r => !processedIds.has(r.id))
      return [...fromStore, ...fromLocal]
    })

    /** 格式化时间 */
    const formatTime = (dateStr) => {
      if (!dateStr) return ''
      const date = new Date(dateStr)
      const now = new Date()
      const diff = now - date
      const minutes = Math.floor(diff / 60000)
      if (minutes < 1) return '刚刚'
      if (minutes < 60) return `${minutes} 分钟前`
      const hours = Math.floor(minutes / 60)
      if (hours < 24) return `${hours} 小时前`
      const days = Math.floor(hours / 24)
      if (days < 7) return `${days} 天前`
      return `${date.getMonth() + 1}/${date.getDate()}`
    }

    // ==================== 生命周期 ====================

    /** 新入群申请到达时自动刷新列表 */
    const _onNewJoinRequest = () => {
      if (groupId.value) {
        loadRequests()
      }
    }

    onLoad((query) => {
      groupId.value = parseInt(query.groupId) || 0
      if (groupId.value) {
        loadRequests()
      } else {
        loading.value = false
      }
      wsService.on('group.join.request', _onNewJoinRequest)
    })

    onUnload(() => {
      wsService.off('group.join.request', _onNewJoinRequest)
    })

    /** 加载入群申请列表 */
    const loadRequests = async () => {
      loading.value = true
      try {
        await groupStore.fetchJoinRequests(groupId.value)
      } catch (e) {
        console.error('获取入群申请失败', e)
        uni.showToast({ title: e?.data?.message || '获取申请列表失败', icon: 'none' })
      }
      loading.value = false
    }

    // ==================== 审批操作 ====================

    /** 通过申请 */
    const handleApprove = async (req) => {
      if (processingMap[req.id]) return
      processingMap[req.id] = true
      try {
        await groupStore.reviewJoinRequest(groupId.value, req.id, 'approve')
        localProcessed.value.unshift({ ...req, status: 1 })
        uni.showToast({ title: '已通过', icon: 'success' })
      } catch (e) {
        console.error('审批通过失败', e)
        uni.showToast({ title: e?.data?.message || '操作失败', icon: 'none' })
      } finally {
        processingMap[req.id] = false
      }
    }

    /** 拒绝申请 */
    const handleReject = async (req) => {
      if (processingMap[req.id]) return
      processingMap[req.id] = true
      try {
        await groupStore.reviewJoinRequest(groupId.value, req.id, 'reject')
        localProcessed.value.unshift({ ...req, status: 2 })
        uni.showToast({ title: '已拒绝', icon: 'none' })
      } catch (e) {
        console.error('审批拒绝失败', e)
        uni.showToast({ title: e?.data?.message || '操作失败', icon: 'none' })
      } finally {
        processingMap[req.id] = false
      }
    }

    return {
      loading,
      pendingList,
      processedList,
      processingMap,
      formatTime,
      handleApprove,
      handleReject,
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

/* ===== 骨架屏 ===== */
.skeleton-list {
  background-color: #FFFFFF;
}

.skeleton-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  gap: 20rpx;
  border-bottom: 1rpx solid #F1F5F9;
}

.skeleton-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 20rpx;
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

.skeleton-line--msg {
  width: 60%;
  height: 24rpx;
}

.skeleton-actions {
  display: flex;
  gap: 12rpx;
}

.skeleton-btn {
  width: 100rpx;
  height: 56rpx;
  border-radius: 12rpx;
  background: linear-gradient(90deg, #F1F5F9 25%, #E2E8F0 50%, #F1F5F9 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* ===== 区块 ===== */
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

/* ===== 申请列表 ===== */
.request-list {
  background-color: #FFFFFF;
}

.request-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  gap: 20rpx;
}

.request-item:last-child {
  border-bottom: none;
}

/* ===== 头像 ===== */
.request-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 20rpx;
  flex-shrink: 0;
}

.request-avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-char {
  color: #FFFFFF;
  font-size: 28rpx;
  font-weight: 600;
}

/* ===== 申请信息 ===== */
.request-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
}

.request-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.request-msg {
  font-size: 24rpx;
  color: #64748B;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.request-time {
  font-size: 22rpx;
  color: #94A3B8;
}

/* ===== 操作按钮 ===== */
.request-actions {
  display: flex;
  flex-direction: column;
  gap: 12rpx;
  flex-shrink: 0;
}

.btn {
  padding: 12rpx 28rpx;
  border-radius: 12rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: opacity 200ms ease;
}

.btn:active {
  opacity: 0.7;
}

.btn--approve {
  background-color: #22C55E;
}

.btn--reject {
  background-color: #F1F5F9;
}

.btn--disabled {
  opacity: 0.5;
  pointer-events: none;
}

.btn-text {
  font-size: 24rpx;
  font-weight: 500;
  color: #FFFFFF;
}

.btn-text--reject {
  color: #EF4444;
}

/* ===== 状态标签 ===== */
.status-tag {
  padding: 8rpx 20rpx;
  border-radius: 12rpx;
  flex-shrink: 0;
}

.status-tag--approved {
  background-color: rgba(34, 197, 94, 0.08);
}

.status-tag--rejected {
  background-color: rgba(239, 68, 68, 0.08);
}

.status-tag-text {
  font-size: 24rpx;
  font-weight: 500;
}

.status-tag-text--approved {
  color: #22C55E;
}

.status-tag-text--rejected {
  color: #EF4444;
}

/* ===== 空状态 ===== */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: 300rpx;
  gap: 16rpx;
}

.empty-title {
  font-size: 32rpx;
  color: #64748B;
  font-weight: 500;
}

.empty-desc {
  font-size: 26rpx;
  color: #94A3B8;
  text-align: center;
  padding: 0 60rpx;
}
</style>
