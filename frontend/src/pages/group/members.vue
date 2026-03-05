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
          :style="{ backgroundColor: getAvatarColor(member.nickname || member.user_nickname) }"
        >
          <text class="avatar-char">{{ getInitial(member.nickname || member.user_nickname) }}</text>
        </view>

        <view class="member-info">
          <view class="member-name-row">
            <text class="member-name">{{ member.nickname || member.user_nickname }}</text>
            <text v-if="member.role === GROUP_ROLE.OWNER" class="role-tag role-tag-owner">🔑 群主</text>
            <text v-else-if="member.role === GROUP_ROLE.ADMIN" class="role-tag role-tag-admin">🛡 管理员</text>
            <text v-else class="role-tag role-tag-member">👤 成员</text>
          </view>
          <view v-if="member.is_muted" class="muted-row">
            <text class="muted-tag">已禁言</text>
          </view>
        </view>

        <view
          v-if="canManage(member)"
          class="manage-btn"
          @tap.stop="onMemberLongPress(member)"
        >
          <uni-icons type="more-filled" size="18" color="#94A3B8" />
        </view>
      </view>
    </view>

    <view v-else class="empty-state">
      <text class="empty-text">{{ searchKeyword ? '未找到匹配的成员' : '暂无成员' }}</text>
    </view>

    <!-- 右上角邀请按钮 -->
    <view class="fab-btn" @tap="goToInvite">
      <uni-icons type="plusempty" size="24" color="#FFFFFF" />
    </view>

    <!-- 自定义操作弹窗 -->
    <view v-if="actionTarget" class="action-overlay" @tap="actionTarget = null">
      <view class="action-sheet" @tap.stop>
        <view class="action-sheet-header">
          <view class="action-sheet-avatar" :style="{ backgroundColor: getAvatarColor(actionTarget.nickname || actionTarget.user_nickname) }">
            <text class="action-sheet-avatar-text">{{ getInitial(actionTarget.nickname || actionTarget.user_nickname) }}</text>
          </view>
          <view class="action-sheet-info">
            <text class="action-sheet-name">{{ actionTarget.nickname || actionTarget.user_nickname }}</text>
            <text class="action-sheet-role">{{ GROUP_ROLE_LABEL[actionTarget.role] || '成员' }}</text>
          </view>
        </view>
        <view class="action-sheet-divider"></view>
        <view class="action-sheet-actions">
          <view
            v-for="action in actionList"
            :key="action.label"
            class="action-sheet-item"
            :class="{ 'action-sheet-item--danger': action.danger }"
            @tap="doAction(action.key)"
          >
            <text class="action-sheet-item-icon">{{ action.icon }}</text>
            <text class="action-sheet-item-label">{{ action.label }}</text>
          </view>
        </view>
        <view class="action-sheet-divider"></view>
        <view class="action-sheet-cancel" @tap="actionTarget = null">
          <text class="action-sheet-cancel-text">取消</text>
        </view>
      </view>
    </view>
  </view>
</template>

<script>
import { onLoad } from '@dcloudio/uni-app'
import { ref, computed } from 'vue'
import { useGroupStore } from '@/store/group'
import { useUserStore } from '@/store/user'
import { getAvatarColor, getInitial } from '@/utils/avatar'
import { GROUP_ROLE, GROUP_ROLE_LABEL } from '@/constants/group'

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
        const userNickname = (m.user_nickname || '').toLowerCase()
        return nickname.includes(kw) || userNickname.includes(kw)
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

    const actionTarget = ref(null)

    /** 构建当前操作目标的可用操作列表 */
    const actionList = computed(() => {
      const member = actionTarget.value
      if (!member) return []

      const list = []
      if (myRole.value === GROUP_ROLE.OWNER) {
        if (member.role === GROUP_ROLE.ADMIN) {
          list.push({ key: 'unsetAdmin', icon: '👤', label: '取消管理员' })
        } else {
          list.push({ key: 'setAdmin', icon: '🛡', label: '设为管理员' })
        }
      }
      if (member.is_muted) {
        list.push({ key: 'unmute', icon: '🔊', label: '解除禁言' })
      } else {
        list.push({ key: 'mute', icon: '🔇', label: '禁言' })
      }
      list.push({ key: 'kick', icon: '🚫', label: '踢出群聊', danger: true })
      return list
    })

    const onMemberLongPress = (member) => {
      if (!canManage(member)) return
      actionTarget.value = member
    }

    const doAction = (key) => {
      const member = actionTarget.value
      if (!member) return
      actionTarget.value = null

      switch (key) {
        case 'setAdmin':
          confirmSetRole(member, GROUP_ROLE.ADMIN)
          break
        case 'unsetAdmin':
          confirmSetRole(member, GROUP_ROLE.MEMBER)
          break
        case 'mute':
          confirmMute(member, true)
          break
        case 'unmute':
          confirmMute(member, false)
          break
        case 'kick':
          confirmKick(member)
          break
      }
    }

    const confirmSetRole = (member, role) => {
      const roleName = role === GROUP_ROLE.ADMIN ? '管理员' : '普通成员'
      const name = member.nickname || member.user_nickname
      uni.showModal({
        title: '确认操作',
        content: `确定将 ${name} 设为${roleName}吗？`,
        success: async (res) => {
          if (res.confirm) {
            try {
              await groupStore.setMemberRole(groupId.value, member.user_id, role)
              uni.showToast({ title: '操作成功', icon: 'success' })
            } catch (e) {
              uni.showToast({ title: e?.message || '操作失败', icon: 'none' })
            }
          }
        }
      })
    }

    const confirmMute = (member, isMuted) => {
      const name = member.nickname || member.user_nickname
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
              uni.showToast({ title: e?.message || '操作失败', icon: 'none' })
            }
          }
        }
      })
    }

    const confirmKick = (member) => {
      const name = member.nickname || member.user_nickname
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
              uni.showToast({ title: e?.message || '操作失败', icon: 'none' })
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
      GROUP_ROLE_LABEL,
      searchKeyword,
      filteredMembers,
      canManage,
      onMemberLongPress,
      actionTarget,
      actionList,
      doAction,
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

.role-tag-member {
  color: #64748B;
  background-color: rgba(100, 116, 139, 0.08);
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

/* ===== 管理按钮 ===== */
.manage-btn {
  flex-shrink: 0;
  width: 64rpx;
  height: 64rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12rpx;
  cursor: pointer;
  transition: background-color 150ms ease;
}

.manage-btn:active {
  background-color: #F1F5F9;
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

/* ===== 自定义操作弹窗 ===== */
.action-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
  animation: fadeIn 200ms ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.action-sheet {
  width: 560rpx;
  max-width: 90vw;
  background-color: #FFFFFF;
  border-radius: 24rpx;
  overflow: hidden;
  box-shadow: 0 16rpx 48rpx rgba(0, 0, 0, 0.15);
  animation: slideUp 200ms ease;
}

@keyframes slideUp {
  from { transform: translateY(40rpx); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

.action-sheet-header {
  display: flex;
  align-items: center;
  padding: 28rpx 32rpx;
  gap: 20rpx;
}

.action-sheet-avatar {
  width: 72rpx;
  height: 72rpx;
  border-radius: 18rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.action-sheet-avatar-text {
  font-size: 28rpx;
  color: #FFFFFF;
  font-weight: 600;
}

.action-sheet-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.action-sheet-name {
  font-size: 30rpx;
  color: #1E293B;
  font-weight: 600;
}

.action-sheet-role {
  font-size: 24rpx;
  color: #94A3B8;
}

.action-sheet-divider {
  height: 1rpx;
  background-color: #F1F5F9;
  margin: 0 24rpx;
}

.action-sheet-actions {
  padding: 8rpx 0;
}

.action-sheet-item {
  display: flex;
  align-items: center;
  padding: 24rpx 32rpx;
  gap: 16rpx;
  cursor: pointer;
  transition: background-color 150ms ease;
}

.action-sheet-item:active {
  background-color: #F8FAFC;
}

.action-sheet-item-icon {
  font-size: 32rpx;
  width: 40rpx;
  text-align: center;
  flex-shrink: 0;
}

.action-sheet-item-label {
  font-size: 30rpx;
  color: #334155;
  font-weight: 500;
}

.action-sheet-item--danger .action-sheet-item-label {
  color: #EF4444;
}

.action-sheet-cancel {
  padding: 24rpx 32rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background-color 150ms ease;
}

.action-sheet-cancel:active {
  background-color: #F8FAFC;
}

.action-sheet-cancel-text {
  font-size: 30rpx;
  color: #94A3B8;
  font-weight: 500;
}
</style>
