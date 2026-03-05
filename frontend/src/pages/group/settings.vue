<!--
  群设置页

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / BG #F8FAFC / Surface #F1F5F9 / Text #1E293B
  功能：群信息展示/编辑、成员概览、免打扰、解散/退出群聊
-->
<template>
  <view class="page-wrapper">
    <!-- 群头像 + 群名称 -->
    <view class="group-card">
      <view class="group-avatar-wrap" @tap="onChangeAvatar">
        <image v-if="groupAvatar" class="group-avatar" :src="groupAvatar" mode="aspectFill" />
        <view v-else class="group-avatar group-avatar-placeholder" :style="{ backgroundColor: getAvatarColor(groupName) }">
          <text class="avatar-text">{{ getInitial(groupName) }}</text>
        </view>
        <view v-if="isOwnerOrAdmin" class="avatar-edit-badge">
          <uni-icons type="camera-filled" size="14" color="#FFFFFF" />
        </view>
      </view>
      <text class="group-name">{{ groupName || '群聊' }}</text>
      <text class="group-id">群ID: {{ groupId }}</text>
    </view>

    <!-- 群公告 -->
    <view class="settings-group">
      <view class="settings-item" @tap="onEditAnnouncement">
        <text class="settings-label">群公告</text>
        <view class="settings-right">
          <text class="settings-value settings-value-ellipsis">{{ currentGroup?.notice || '暂无公告' }}</text>
          <uni-icons v-if="isOwnerOrAdmin" type="forward" size="16" color="#94A3B8" />
        </view>
      </view>
    </view>

    <!-- 成员概览 -->
    <view class="settings-group">
      <view class="section-header">
        <text class="section-title">群成员（{{ currentMembers.length }}）</text>
        <view class="section-action" @tap="goToMembers">
          <text class="section-action-text">查看全部</text>
          <uni-icons type="forward" size="14" color="#2563EB" />
        </view>
      </view>
      <scroll-view class="member-preview-scroll" scroll-x>
        <view class="member-preview-list">
          <view
            v-for="member in previewMembers"
            :key="member.user_id"
            class="member-preview-item"
          >
            <image
              v-if="member.avatar"
              class="member-preview-avatar"
              :src="member.avatar"
              mode="aspectFill"
            />
            <view
              v-else
              class="member-preview-avatar member-preview-avatar-placeholder"
              :style="{ backgroundColor: getAvatarColor(member.nickname || member.user_nickname) }"
            >
              <text class="member-preview-char">{{ getInitial(member.nickname || member.user_nickname) }}</text>
            </view>
            <text class="member-preview-name">{{ member.nickname || member.user_nickname }}</text>
          </view>
        </view>
      </scroll-view>
    </view>

    <!-- 功能区 -->
    <view class="settings-group">
      <view class="settings-item" @tap="onEditGroupName">
        <text class="settings-label">群名称</text>
        <view class="settings-right">
          <text class="settings-value">{{ groupName }}</text>
          <uni-icons v-if="isOwnerOrAdmin" type="forward" size="16" color="#94A3B8" />
        </view>
      </view>
      <view class="settings-item" @tap="onEditNickname">
        <text class="settings-label">我在本群的昵称</text>
        <view class="settings-right">
          <text class="settings-value">{{ myNickname || '未设置' }}</text>
          <uni-icons type="forward" size="16" color="#94A3B8" />
        </view>
      </view>
      <view class="settings-item">
        <text class="settings-label">消息免打扰</text>
        <switch
          :checked="isDoNotDisturb"
          color="#2563EB"
          style="transform: scale(0.8);"
          @change="onToggleDND"
        />
      </view>
    </view>

    <!-- 危险操作区 -->
    <view class="settings-group">
      <view v-if="isOwner" class="settings-item settings-danger" @tap="onDissolve">
        <text class="settings-label-danger">解散群聊</text>
      </view>
      <view v-else class="settings-item settings-danger" @tap="onLeave">
        <text class="settings-label-danger">退出群聊</text>
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
import { GROUP_ROLE } from '@/constants/group'

export default {
  name: 'GroupSettings',
  setup() {
    const groupStore = useGroupStore()
    const userStore = useUserStore()

    const groupId = ref(0)
    const conversationId = ref(0)
    const groupName = ref('')
    const groupAvatar = ref('')
    const isDoNotDisturb = ref(false)

    const currentGroup = computed(() => groupStore.currentGroup)
    const currentMembers = computed(() => groupStore.currentMembers)
    const myId = computed(() => Number(userStore.userInfo?.id) || 0)

    /** 成员概览：最多显示前 5 个 */
    const previewMembers = computed(() => {
      return currentMembers.value.slice(0, 5)
    })

    /** 当前用户在群中的成员信息 */
    const myMember = computed(() => {
      return currentMembers.value.find(m => m.user_id === myId.value)
    })

    /** 我在本群的昵称 */
    const myNickname = computed(() => {
      return myMember.value?.nickname || ''
    })

    /** 当前用户是否为群主 */
    const isOwner = computed(() => {
      return myMember.value?.role === GROUP_ROLE.OWNER
    })

    /** 当前用户是否为群主或管理员 */
    const isOwnerOrAdmin = computed(() => {
      const role = myMember.value?.role
      return role === GROUP_ROLE.OWNER || role === GROUP_ROLE.ADMIN
    })

    // ==================== 生命周期 ====================

    onLoad((query) => {
      groupId.value = parseInt(query.groupId) || 0
      conversationId.value = parseInt(query.conversationId) || 0
      groupName.value = decodeURIComponent(query.peerName || '')
      groupAvatar.value = decodeURIComponent(query.peerAvatar || '')

      if (groupId.value) {
        groupStore.fetchGroupDetail(groupId.value).then((detail) => {
          if (detail) {
            groupName.value = detail.name || groupName.value
            groupAvatar.value = detail.avatar || groupAvatar.value
          }
        })
        groupStore.fetchMembers(groupId.value)
      }
    })

    // ==================== 编辑群名称 ====================

    const onEditGroupName = () => {
      if (!isOwnerOrAdmin.value) return
      uni.showModal({
        title: '修改群名称',
        editable: true,
        placeholderText: '请输入新群名称',
        content: groupName.value,
        success: async (res) => {
          if (res.confirm && res.content && res.content.trim()) {
            try {
              await groupStore.updateGroup(groupId.value, { name: res.content.trim() })
              groupName.value = res.content.trim()
              uni.showToast({ title: '修改成功', icon: 'success' })
            } catch (e) {
              uni.showToast({ title: e?.message || '修改失败', icon: 'none' })
            }
          }
        }
      })
    }

    // ==================== 编辑群公告 ====================

    const onEditAnnouncement = () => {
      if (!isOwnerOrAdmin.value) return
      uni.showModal({
        title: '修改群公告',
        editable: true,
        placeholderText: '请输入群公告',
        content: currentGroup.value?.notice || '',
        success: async (res) => {
          if (res.confirm) {
            try {
              await groupStore.updateGroup(groupId.value, { notice: res.content || '' })
              uni.showToast({ title: '修改成功', icon: 'success' })
            } catch (e) {
              uni.showToast({ title: e?.message || '修改失败', icon: 'none' })
            }
          }
        }
      })
    }

    // ==================== 修改群头像（占位，仅群主/管理员） ====================

    const onChangeAvatar = () => {
      if (!isOwnerOrAdmin.value) return
      uni.showToast({ title: '头像修改功能即将开放', icon: 'none' })
    }

    // ==================== 修改我的群昵称 ====================

    const onEditNickname = () => {
      uni.showModal({
        title: '修改群昵称',
        editable: true,
        placeholderText: '请输入群昵称',
        content: myNickname.value,
        success: async (res) => {
          if (res.confirm && res.content !== undefined) {
            try {
              await groupStore.updateNickname(groupId.value, res.content.trim())
              groupStore.fetchMembers(groupId.value)
              uni.showToast({ title: '修改成功', icon: 'success' })
            } catch (e) {
              uni.showToast({ title: e?.message || '修改失败', icon: 'none' })
            }
          }
        }
      })
    }

    // ==================== 免打扰 ====================

    const onToggleDND = async (e) => {
      const newVal = e.detail.value
      try {
        await groupStore.setDoNotDisturb(conversationId.value, newVal)
        isDoNotDisturb.value = newVal
      } catch {
        isDoNotDisturb.value = !newVal
        uni.showToast({ title: '设置失败', icon: 'none' })
      }
    }

    // ==================== 解散群聊（仅群主） ====================

    const onDissolve = () => {
      uni.showModal({
        title: '解散群聊',
        content: '确定要解散该群聊吗？此操作不可撤回，所有群成员将被移除。',
        confirmColor: '#EF4444',
        success: async (res) => {
          if (res.confirm) {
            try {
              await groupStore.dissolveGroup(groupId.value)
              uni.showToast({ title: '群聊已解散', icon: 'success' })
              setTimeout(() => {
                uni.switchTab({ url: '/pages/chat/index' })
              }, 500)
            } catch (e) {
              uni.showToast({ title: e?.message || '解散失败', icon: 'none' })
            }
          }
        }
      })
    }

    // ==================== 退出群聊（非群主） ====================

    const onLeave = () => {
      uni.showModal({
        title: '退出群聊',
        content: '确定要退出该群聊吗？退出后将不再接收该群消息。',
        confirmColor: '#EF4444',
        success: async (res) => {
          if (res.confirm) {
            try {
              await groupStore.leaveGroup(groupId.value)
              uni.showToast({ title: '已退出群聊', icon: 'success' })
              setTimeout(() => {
                uni.switchTab({ url: '/pages/chat/index' })
              }, 500)
            } catch (e) {
              uni.showToast({ title: e?.message || '退出失败', icon: 'none' })
            }
          }
        }
      })
    }

    // ==================== 导航 ====================

    const goToMembers = () => {
      uni.navigateTo({
        url: `/pages/group/members?groupId=${groupId.value}`
      })
    }

    return {
      groupId,
      groupName,
      groupAvatar,
      isDoNotDisturb,
      currentGroup,
      currentMembers,
      previewMembers,
      myNickname,
      isOwner,
      isOwnerOrAdmin,
      onEditGroupName,
      onEditAnnouncement,
      onChangeAvatar,
      onEditNickname,
      onToggleDND,
      onDissolve,
      onLeave,
      goToMembers,
      getAvatarColor,
      getInitial
    }
  }
}
</script>

<style scoped>
.page-wrapper {
  min-height: 100vh;
  background-color: #F1F5F9;
}

/* ===== 群信息卡片 ===== */
.group-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 48rpx 32rpx;
  background-color: #FFFFFF;
  margin-bottom: 16rpx;
}

.group-avatar-wrap {
  position: relative;
  margin-bottom: 20rpx;
}

.group-avatar {
  width: 120rpx;
  height: 120rpx;
  border-radius: 30rpx;
}

.group-avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar-text {
  color: #FFFFFF;
  font-size: 48rpx;
  font-weight: 600;
}

.avatar-edit-badge {
  position: absolute;
  bottom: -4rpx;
  right: -4rpx;
  width: 40rpx;
  height: 40rpx;
  border-radius: 50%;
  background-color: #2563EB;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 3rpx solid #FFFFFF;
}

.group-name {
  font-size: 34rpx;
  font-weight: 600;
  color: #1E293B;
  margin-bottom: 8rpx;
}

.group-id {
  font-size: 24rpx;
  color: #94A3B8;
}

/* ===== 区块通用 ===== */
.settings-group {
  background-color: #FFFFFF;
  margin-bottom: 16rpx;
}

.settings-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 32rpx;
  min-height: 88rpx;
  border-bottom: 1rpx solid #F1F5F9;
  transition: background-color 150ms ease;
}

.settings-item:last-child {
  border-bottom: none;
}

.settings-item:active {
  background-color: #F8FAFC;
}

.settings-label {
  font-size: 30rpx;
  color: #1E293B;
  flex-shrink: 0;
}

.settings-right {
  display: flex;
  align-items: center;
  gap: 8rpx;
  min-width: 0;
  flex: 1;
  justify-content: flex-end;
}

.settings-value {
  font-size: 26rpx;
  color: #94A3B8;
}

.settings-value-ellipsis {
  max-width: 400rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.settings-label-danger {
  font-size: 30rpx;
  color: #EF4444;
}

/* ===== 成员概览区 ===== */
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20rpx 32rpx 12rpx;
}

.section-title {
  font-size: 26rpx;
  color: #94A3B8;
  font-weight: 500;
}

.section-action {
  display: flex;
  align-items: center;
  gap: 4rpx;
}

.section-action-text {
  font-size: 24rpx;
  color: #2563EB;
}

.member-preview-scroll {
  white-space: nowrap;
  padding: 16rpx 32rpx 24rpx;
}

.member-preview-list {
  display: flex;
  gap: 24rpx;
}

.member-preview-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 96rpx;
  flex-shrink: 0;
}

.member-preview-avatar {
  width: 80rpx;
  height: 80rpx;
  border-radius: 20rpx;
}

.member-preview-avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
}

.member-preview-char {
  color: #FFFFFF;
  font-size: 28rpx;
  font-weight: 600;
}

.member-preview-name {
  font-size: 22rpx;
  color: #475569;
  margin-top: 8rpx;
  max-width: 96rpx;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: center;
}
</style>
