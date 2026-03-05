<!--
  搜索群聊页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Surface #F1F5F9 / Text #1E293B
  功能：搜索公开群聊、查看群信息、申请加入群聊
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
          placeholder="输入群名称搜索"
          placeholder-style="color: #94A3B8"
          confirm-type="search"
          @confirm="doSearch"
        />
        <view v-if="keyword" class="search-clear" @tap="keyword = ''">
          <uni-icons type="clear" size="16" color="#94A3B8" />
        </view>
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

      <!-- 加载中 -->
      <view v-if="searchLoading && page === 1" class="loading-wrap">
        <text class="loading-text">搜索中...</text>
      </view>

      <!-- 结果列表 -->
      <scroll-view
        v-else-if="resultList.length > 0"
        class="result-scroll"
        scroll-y
        @scrolltolower="loadMore"
      >
        <view class="result-list">
          <view
            v-for="group in resultList"
            :key="group.id"
            class="group-item"
          >
            <image
              v-if="group.avatar"
              class="group-avatar"
              :src="group.avatar"
              mode="aspectFill"
            />
            <view
              v-else
              class="group-avatar group-avatar-placeholder"
              :style="{ backgroundColor: getAvatarColor(group.name) }"
            >
              <text class="avatar-char">{{ getInitial(group.name) }}</text>
            </view>

            <view class="group-info">
              <text class="group-name">{{ group.name }}</text>
              <text class="group-meta">{{ group.member_count || 0 }} 名成员</text>
              <text v-if="group.description" class="group-desc">{{ group.description }}</text>
            </view>

            <view v-if="isInGroup(group.id)" class="action-btn action-btn--joined">
              <text class="action-btn-text action-btn-text--joined">已加入</text>
            </view>
            <view
              v-else
              class="action-btn action-btn--apply"
              :class="{ 'action-btn--disabled': appliedMap[group.id] }"
              @tap="showApplyDialog(group)"
            >
              <text class="action-btn-text action-btn-text--apply">
                {{ appliedMap[group.id] ? '已申请' : '申请加入' }}
              </text>
            </view>
          </view>
        </view>

        <!-- 底部加载状态 -->
        <view class="load-more-wrap">
          <text v-if="loadingMore" class="load-more-text">加载中...</text>
          <text v-else-if="noMore" class="load-more-text">没有更多了</text>
        </view>
      </scroll-view>

      <!-- 无结果 -->
      <view v-else class="empty-state">
        <uni-icons type="search" size="48" color="#CBD5E1" />
        <text class="empty-text">未找到相关群聊</text>
      </view>
    </view>

    <!-- 未搜索提示 -->
    <view v-else class="hint-section">
      <uni-icons type="search" size="48" color="#CBD5E1" />
      <text class="hint-text">输入关键词搜索群聊</text>
    </view>
  </view>
</template>

<script>
import { onLoad } from '@dcloudio/uni-app'
import { ref, computed, reactive } from 'vue'
import { useGroupStore } from '@/store/group'
import { useChatStore } from '@/store/chat'
import { getAvatarColor, getInitial } from '@/utils/avatar'

export default {
  name: 'GroupSearch',
  setup() {
    const groupStore = useGroupStore()
    const chatStore = useChatStore()

    /** 当前用户已加入的群 ID 集合 */
    const myGroupIds = computed(() => {
      const ids = new Set()
      chatStore.conversationList.forEach(conv => {
        if (conv.type === 2 && conv.group_id) {
          ids.add(conv.group_id)
        }
      })
      return ids
    })

    /** 判断是否已在该群内 */
    const isInGroup = (groupId) => myGroupIds.value.has(groupId)

    const keyword = ref('')
    const searched = ref(false)
    const searchLoading = ref(false)
    const loadingMore = ref(false)
    const page = ref(1)
    const pageSize = 20
    const appliedMap = reactive({})

    /** 搜索结果列表 */
    const resultList = computed(() => groupStore.searchResults)

    /** 是否还有更多数据 */
    const noMore = computed(() => {
      return resultList.value.length >= groupStore.searchTotal && groupStore.searchTotal > 0
    })

    // ==================== 生命周期 ====================

    onLoad(() => {
      groupStore.searchResults = []
      groupStore.searchTotal = 0
      if (chatStore.conversationList.length === 0) {
        chatStore.fetchConversations()
      }
    })

    // ==================== 搜索逻辑 ====================

    /** 执行搜索 */
    const doSearch = async () => {
      const kw = keyword.value.trim()
      if (!kw) {
        uni.showToast({ title: '请输入搜索关键词', icon: 'none' })
        return
      }

      searched.value = true
      searchLoading.value = true
      page.value = 1

      try {
        await groupStore.searchGroups(kw, 1, pageSize)
      } catch (e) {
        console.error('搜索群聊失败', e)
        uni.showToast({ title: e?.message || '搜索失败', icon: 'none' })
      }

      searchLoading.value = false
    }

    /** 滚动加载更多 */
    const loadMore = async () => {
      if (loadingMore.value || noMore.value || !searched.value) return

      loadingMore.value = true
      page.value += 1

      try {
        await groupStore.searchGroups(keyword.value.trim(), page.value, pageSize, true)
      } catch (e) {
        console.error('加载更多群聊失败', e)
        page.value -= 1
      }

      loadingMore.value = false
    }

    // ==================== 申请加入 ====================

    /** 弹出申请附言输入框 */
    const showApplyDialog = (group) => {
      if (appliedMap[group.id]) return

      uni.showModal({
        title: `申请加入「${group.name}」`,
        editable: true,
        placeholderText: '请输入申请附言（可选）',
        content: '',
        success: async (res) => {
          if (res.confirm) {
            try {
              await groupStore.submitJoinRequest(group.id, res.content || '')
              appliedMap[group.id] = true
              uni.showToast({ title: '申请已发送', icon: 'success' })
            } catch (e) {
              console.error('发送入群申请失败', e)
              uni.showToast({ title: e?.message || '申请失败', icon: 'none' })
            }
          }
        }
      })
    }

    return {
      keyword,
      searched,
      searchLoading,
      loadingMore,
      page,
      resultList,
      noMore,
      appliedMap,
      isInGroup,
      doSearch,
      loadMore,
      showApplyDialog,
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

/* ===== 搜索栏 ===== */
.search-bar {
  display: flex;
  align-items: center;
  padding: 16rpx 24rpx;
  background-color: #FFFFFF;
  gap: 16rpx;
  border-bottom: 1rpx solid #E2E8F0;
}

.search-input-wrap {
  flex: 1;
  background-color: #F1F5F9;
  border-radius: 36rpx;
  padding: 0 24rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.search-input {
  flex: 1;
  font-size: 28rpx;
  color: #1E293B;
  height: 72rpx;
}

.search-clear {
  flex-shrink: 0;
  padding: 8rpx;
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

/* ===== 区块标题 ===== */
.section-header {
  padding: 20rpx 32rpx;
}

.section-title {
  font-size: 26rpx;
  color: #94A3B8;
  font-weight: 500;
}

/* ===== 加载状态 ===== */
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

/* ===== 搜索结果滚动区域 ===== */
.result-scroll {
  height: calc(100vh - 200rpx);
}

/* ===== 结果列表 ===== */
.result-list {
  background-color: #FFFFFF;
}

.group-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  gap: 20rpx;
}

.group-item:last-child {
  border-bottom: none;
}

/* ===== 群头像 ===== */
.group-avatar {
  width: 88rpx;
  height: 88rpx;
  border-radius: 24rpx;
  flex-shrink: 0;
}

.group-avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-char {
  color: #FFFFFF;
  font-size: 32rpx;
  font-weight: 600;
}

/* ===== 群信息 ===== */
.group-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
}

.group-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-meta {
  font-size: 24rpx;
  color: #94A3B8;
}

.group-desc {
  font-size: 24rpx;
  color: #64748B;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ===== 操作按钮 ===== */
.action-btn {
  padding: 12rpx 24rpx;
  border-radius: 12rpx;
  flex-shrink: 0;
  cursor: pointer;
  transition: opacity 200ms ease;
}

.action-btn:active {
  opacity: 0.8;
}

.action-btn--apply {
  border: 2rpx solid #2563EB;
  background-color: transparent;
}

.action-btn--disabled {
  border-color: #CBD5E1;
  pointer-events: none;
}

.action-btn-text {
  font-size: 24rpx;
  font-weight: 500;
}

.action-btn-text--apply {
  color: #2563EB;
}

.action-btn--disabled .action-btn-text--apply {
  color: #94A3B8;
}

.action-btn--joined {
  border-color: #059669;
  background-color: #ECFDF5;
  pointer-events: none;
}

.action-btn-text--joined {
  color: #059669;
  font-size: 24rpx;
  font-weight: 500;
}

/* ===== 加载更多 ===== */
.load-more-wrap {
  padding: 24rpx 0 40rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

.load-more-text {
  font-size: 24rpx;
  color: #94A3B8;
}

/* ===== 空状态 ===== */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: 200rpx;
  gap: 16rpx;
}

.empty-text {
  font-size: 28rpx;
  color: #94A3B8;
}

/* ===== 未搜索提示 ===== */
.hint-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding-top: 300rpx;
  gap: 16rpx;
}

.hint-text {
  font-size: 28rpx;
  color: #94A3B8;
}
</style>
