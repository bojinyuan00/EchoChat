<!--
  群成员列表页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Surface #F1F5F9 / Text #1E293B
  功能：成员搜索、角色标识、禁言标识、管理操作（设管理员/禁言/踢人）
-->
<template>
  <view class="page-wrapper">
    <!-- 搜索栏 -->
    <view class="search-section">
      <view class="search-bar">
        <uni-icons type="search" size="18" color="#94A3B8" />
        <input
          class="search-input"
          v-model="searchKeyword"
          placeholder="搜索成员昵称"
          placeholder-style="color: #94A3B8"
        />
        <view v-if="searchKeyword" class="search-clear" @tap="searchKeyword = ''">
          <uni-icons type="clear" size="16" color="#94A3B8" />
        </view>
      </view>
    </view>

    <!-- 成员数量标题 -->
    <view class="section-header">
      <text class="section-title">全部成员（{{ filteredMembers.length }}）</text>
    </view>

    <!-- 成员列表 -->
    <view v-if="filteredMembers.length > 0" class="member-list">
      <view
        v-for="member in filteredMembers"
        :key="member.user_id"
        class="member-item"
        @longpress="onMemberLongPress(member)"
      >
        <image
          v-if="member.avatar"
          class="member-avatar"
          :src="member.avatar"
          mode="aspectFill"
        />
        <view
          v-else
          class="member-avatar member-avatar-placeholder"
          :style="{ backgroundColor: getAvatarColor(member.nickname || member.username) }"
        >
          <text class="avatar-char">{{ getInitial(member.nickname || member.username) }}</text>
        </view>

        <view class="member-info">
          <view class="member-name-row">
            <text class="member-name">{{ member.nickname || member.username }}</text>
            <text v-if="member.role === GROUP_ROLE.OWNER" class="role-tag role-tag-owner">🔑 群主</text>
            <text v-else-if="member.role === GROUP_ROLE.ADMIN" class="role-tag role-tag-admin">🛡 管理员</text>
          </view>
          <view v-if="member.is_muted" class="muted-row">
            <text class="muted-tag">已禁言</text>
          </view>
        </view>

        <uni-icons
          v-if="canManage(member)"
          type="more-filled"
          size="18"
          color="#94A3B8"
        />
      </view>
    </view>

    <view v-else class="empty-state">
      <text class="empty-text">{{ searchKeyword ? '未找到匹配的成员' : '暂无成员' }}</text>
    </view>

    <!-- 右上角邀请按钮 -->
    <view class="fab-btn" @tap="goToInvite">
      <uni-icons type="plusempty" size="24" color="#FFFFFF" />
    </view>
  </view>
</template>

<script>
import { onLoad } from '@dcloudio/uni-app'
import { ref, computed } from 'vue'
import { useGroupStore } from '@/store/group'
import { useUserStore } from '@/store/user'
import { getAvatarColor, getInitial } from '@/utils/avatar'
import { GROUP_ROLE } from '@/constants/group'

export default {
  name: 'GroupMembers',
  setup() {
    const groupStore = useGroupStore()
    const userStore = useUserStore()

    const groupId = ref(0)
    const searchKeyword = ref('')

    const currentMembers = computed(() => groupStore.currentMembers)
    const myId = computed(() => Number(userStore.userInfo?.id) || 0)

    /** 当前用户在群中的角色 */
    const myRole = computed(() => {
      const me = currentMembers.value.find(m => m.user_id === myId.value)
      return me ? me.role : 0
    })

    /** 搜索过滤后的成员列表 */
    const filteredMembers = computed(() => {
      const kw = searchKeyword.value.trim().toLowerCase()
      if (!kw) return currentMembers.value
      return currentMembers.value.filter(m => {
        const nickname = (m.nickname || '').toLowerCase()
        const username = (m.username || '').toLowerCase()
        return nickname.includes(kw) || username.includes(kw)
      })
    })

    /**
     * 判断当前用户是否可以管理某个成员
     * - 群主可管理所有非群主成员
     * - 管理员仅可管理普通成员
     * - 普通成员无管理权限
     * - 不能管理自己
     */
    const canManage = (member) => {
      if (member.user_id === myId.value) return false
      if (myRole.value === GROUP_ROLE.OWNER) return member.role !== GROUP_ROLE.OWNER
      if (myRole.value === GROUP_ROLE.ADMIN) return member.role === GROUP_ROLE.MEMBER
      return false
    }

    // ==================== 生命周期 ====================

    onLoad((query) => {
      groupId.value = parseInt(query.groupId) || 0
      if (groupId.value) {
        groupStore.fetchMembers(groupId.value)
      }
    })

    // ==================== 管理操作 ====================

    const onMemberLongPress = (member) => {
      if (!canManage(member)) return

      const actions = []

      if (myRole.value === GROUP_ROLE.OWNER) {
        if (member.role === GROUP_ROLE.ADMIN) {
          actions.push('取消管理员')
        } else {
          actions.push('设为管理员')
        }
      }

      if (member.is_muted) {
        actions.push('解除禁言')
      } else {
        actions.push('禁言')
      }

      actions.push('踢出群聊')

      uni.showActionSheet({
        itemList: actions,
        success: (res) => {
          const action = actions[res.tapIndex]
          handleAction(action, member)
        }
      })
    }

    const handleAction = (action, member) => {
      switch (action) {
        case '设为管理员':
          confirmSetRole(member, GROUP_ROLE.ADMIN)
          break
        case '取消管理员':
          confirmSetRole(member, GROUP_ROLE.MEMBER)
          break
        case '禁言':
          confirmMute(member, true)
          break
        case '解除禁言':
          confirmMute(member, false)
          break
        case '踢出群聊':
          confirmKick(member)
          break
      }
    }

    const confirmSetRole = (member, role) => {
      const roleName = role === GROUP_ROLE.ADMIN ? '管理员' : '普通成员'
      const name = member.nickname || member.username
      uni.showModal({
        title: '确认操作',
        content: `确定将 ${name} 设为${roleName}吗？`,
        success: async (res) => {
          if (res.confirm) {
            try {
              await groupStore.setMemberRole(groupId.value, member.user_id, role)
              uni.showToast({ title: '操作成功', icon: 'success' })
            } catch (e) {
              uni.showToast({ title: e?.data?.message || '操作失败', icon: 'none' })
            }
          }
        }
      })
    }

    const confirmMute = (member, isMuted) => {
      const name = member.nickname || member.username
      const tip = isMuted ? `确定禁言 ${name} 吗？` : `确定解除 ${name} 的禁言吗？`
      uni.showModal({
        title: '确认操作',
        content: tip,
        success: async (res) => {
          if (res.confirm) {
            try {
              await groupStore.muteMember(groupId.value, member.user_id, isMuted)
              uni.showToast({ title: '操作成功', icon: 'success' })
            } catch (e) {
              uni.showToast({ title: e?.data?.message || '操作失败', icon: 'none' })
            }
          }
        }
      })
    }

    const confirmKick = (member) => {
      const name = member.nickname || member.username
      uni.showModal({
        title: '踢出群聊',
        content: `确定将 ${name} 踢出群聊吗？`,
        confirmColor: '#EF4444',
        success: async (res) => {
          if (res.confirm) {
            try {
              await groupStore.kickMember(groupId.value, member.user_id)
              uni.showToast({ title: '已踢出', icon: 'success' })
            } catch (e) {
              uni.showToast({ title: e?.data?.message || '操作失败', icon: 'none' })
            }
          }
        }
      })
    }

    // ==================== 导航 ====================

    const goToInvite = () => {
      uni.navigateTo({
        url: `/pages/group/invite?groupId=${groupId.value}`
      })
    }

    return {
      GROUP_ROLE,
      searchKeyword,
      filteredMembers,
      canManage,
      onMemberLongPress,
      goToInvite,
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
.search-section {
  background-color: #FFFFFF;
  padding: 16rpx 24rpx;
  border-bottom: 1rpx solid #E2E8F0;
}

.search-bar {
  display: flex;
  align-items: center;
  background-color: #F1F5F9;
  border-radius: 36rpx;
  padding: 0 24rpx;
  height: 72rpx;
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

/* ===== 区块标题 ===== */
.section-header {
  padding: 20rpx 32rpx 12rpx;
}

.section-title {
  font-size: 26rpx;
  color: #94A3B8;
  font-weight: 500;
}

/* ===== 成员列表 ===== */
.member-list {
  background-color: #FFFFFF;
}

.member-item {
  display: flex;
  align-items: center;
  padding: 20rpx 32rpx;
  border-bottom: 1rpx solid #F1F5F9;
  gap: 20rpx;
  transition: background-color 150ms ease;
}

.member-item:active {
  background-color: #F8FAFC;
}

.member-item:last-child {
  border-bottom: none;
}

/* ===== 头像 ===== */
.member-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 20rpx;
  flex-shrink: 0;
}

.member-avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-char {
  color: #FFFFFF;
  font-size: 28rpx;
  font-weight: 600;
}

/* ===== 成员信息 ===== */
.member-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6rpx;
  min-width: 0;
}

.member-name-row {
  display: flex;
  align-items: center;
  gap: 12rpx;
}

.member-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ===== 角色标签 ===== */
.role-tag {
  font-size: 22rpx;
  padding: 4rpx 12rpx;
  border-radius: 8rpx;
  flex-shrink: 0;
  line-height: 1.4;
}

.role-tag-owner {
  color: #D97706;
  background-color: rgba(217, 119, 6, 0.08);
}

.role-tag-admin {
  color: #2563EB;
  background-color: rgba(37, 99, 235, 0.08);
}

/* ===== 禁言标签 ===== */
.muted-row {
  display: flex;
  align-items: center;
}

.muted-tag {
  font-size: 22rpx;
  color: #EF4444;
  background-color: rgba(239, 68, 68, 0.08);
  padding: 2rpx 12rpx;
  border-radius: 6rpx;
}

/* ===== 空状态 ===== */
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

/* ===== 悬浮按钮 ===== */
.fab-btn {
  position: fixed;
  right: 32rpx;
  bottom: calc(64rpx + env(safe-area-inset-bottom));
  width: 96rpx;
  height: 96rpx;
  border-radius: 50%;
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8rpx 24rpx rgba(37, 99, 235, 0.3);
  transition: opacity 200ms ease;
}

.fab-btn:active {
  opacity: 0.85;
}
</style>
