<!--
  群聊对话页

  布局规则：
  - 自己的消息：靠右对齐，顺序 [状态图标] [蓝色气泡] [自己头像]
  - 他人的消息：靠左对齐，顺序 [头像] [昵称 + 白色气泡]
  - 系统消息（type === 10）：居中灰色文字，无头像无气泡
  - 导航栏：返回 / 群名称 / 群设置入口
-->
<template>
  <view class="page-wrapper">
    <!-- 导航栏 -->
    <view class="nav-bar">
      <view class="nav-left" @tap="goBack">
        <uni-icons type="back" size="20" color="#1E293B" />
      </view>
      <view class="nav-center">
        <text class="nav-title">{{ groupName || '群聊' }}</text>
        <text v-if="memberCount" class="nav-subtitle">({{ memberCount }})</text>
      </view>
      <view class="nav-right" @tap="goToSettings">
        <uni-icons type="more-filled" size="20" color="#475569" />
      </view>
    </view>

    <!-- 消息列表 -->
    <scroll-view
      scroll-y
      class="msg-list"
      :scroll-into-view="scrollToId"
      :scroll-with-animation="true"
      @scrolltoupper="onLoadMore"
    >
      <view v-if="hasMore" class="load-more" @tap="onLoadMore">
        <text class="load-more-text">{{ loadingMore ? '加载中...' : '加载更多' }}</text>
      </view>

      <view
        v-for="msg in messages"
        :key="msg.client_msg_id || msg.id"
        :id="'msg-' + (msg.id || msg.client_msg_id)"
      >
        <!-- ====== 系统消息（居中灰色文字） ====== -->
        <view v-if="msg.type === 10 || msg.sender_id === 0" class="msg-row-system">
          <text class="system-text">{{ msg.content }}</text>
        </view>

        <!-- ====== 普通消息 ====== -->
        <view
          v-else
          class="msg-row"
          :class="isSelf(msg) ? 'msg-row-self' : 'msg-row-other'"
        >
          <!-- 他人消息（左侧）：[头像] [昵称 + 气泡] -->
          <template v-if="!isSelf(msg)">
            <view class="avatar-wrap">
              <image
                v-if="getMemberAvatar(msg.sender_id)"
                class="avatar-img"
                :src="getMemberAvatar(msg.sender_id)"
                mode="aspectFill"
              />
              <view v-else class="avatar-img avatar-placeholder avatar-peer">
                <text class="avatar-char">{{ getMemberName(msg.sender_id)[0] || '?' }}</text>
              </view>
            </view>
            <view class="bubble-col">
              <text class="sender-name">{{ getMemberName(msg.sender_id) }}</text>
              <view
                class="bubble bubble-other"
                :class="{ 'bubble-recalled': msg.status === 2 }"
                @longpress="onMsgLongPress(msg)"
              >
                <text v-if="msg.status === 2" class="recalled-text">消息已撤回</text>
                <text v-else class="msg-text">{{ msg.content }}</text>
              </view>
            </view>
          </template>

          <!-- 自己消息（右侧）：[状态] [气泡] [头像] -->
          <template v-else>
            <view class="self-msg-col">
              <view class="self-msg-row">
                <view v-if="msg._sending" class="msg-status">
                  <uni-icons type="loop" size="16" color="#94A3B8" />
                </view>
                <view v-if="msg._failed" class="msg-status msg-status-tap" @tap="onResend(msg)">
                  <uni-icons type="info-filled" size="18" color="#EF4444" />
                </view>
                <view
                  class="bubble bubble-self"
                  :class="{ 'bubble-recalled': msg.status === 2 }"
                  @longpress="onMsgLongPress(msg)"
                >
                  <text v-if="msg.status === 2" class="recalled-text">消息已撤回</text>
                  <text v-else class="msg-text msg-text-self">{{ msg.content }}</text>
                </view>
                <view class="avatar-wrap">
                  <image v-if="selfAvatar" class="avatar-img" :src="selfAvatar" mode="aspectFill" />
                  <view v-else class="avatar-img avatar-placeholder avatar-self">
                    <text class="avatar-char">{{ (selfName || '我')[0] }}</text>
                  </view>
                </view>
              </view>
              <!-- 群聊已读计数 -->
              <text
                v-if="getReadLabel(msg)"
                class="read-label"
                @tap="goToReadDetail(msg)"
              >
                {{ getReadLabel(msg) }}
              </text>
            </view>
          </template>
        </view>
      </view>

      <view id="msg-bottom" style="height: 2rpx;" />
    </scroll-view>

    <!-- @成员选择器 -->
    <view v-if="showAtPicker" class="at-overlay" @tap="showAtPicker = false">
      <view class="at-panel" @tap.stop>
        <view class="at-header">
          <text class="at-header-text">选择成员</text>
          <view class="at-close" @tap="showAtPicker = false">
            <uni-icons type="close" size="18" color="#475569" />
          </view>
        </view>
        <scroll-view scroll-y class="at-member-list">
          <view class="at-member-item" @tap="onSelectAtMember(null)">
            <view class="avatar-img avatar-placeholder avatar-all">
              <text class="avatar-char">@</text>
            </view>
            <text class="at-member-name">所有人</text>
          </view>
          <view
            v-for="member in atMemberList"
            :key="member.user_id"
            class="at-member-item"
            @tap="onSelectAtMember(member)"
          >
            <image
              v-if="member.avatar"
              class="avatar-img at-avatar"
              :src="member.avatar"
              mode="aspectFill"
            />
            <view v-else class="avatar-img avatar-placeholder avatar-peer at-avatar">
              <text class="avatar-char">{{ (member.nickname || member.user_nickname || '?')[0] }}</text>
            </view>
            <text class="at-member-name">{{ member.nickname || member.user_nickname }}</text>
            <text v-if="member.role === GROUP_ROLE.OWNER" class="at-role-tag">群主</text>
            <text v-else-if="member.role === GROUP_ROLE.ADMIN" class="at-role-tag">管理</text>
          </view>
        </scroll-view>
      </view>
    </view>

    <!-- 输入栏 -->
    <view v-if="isMuted" class="input-bar input-bar-muted">
      <text class="muted-tip">{{ isAllMuted ? '当前群聊已开启全体禁言' : '你已被禁言，无法发送消息' }}</text>
    </view>
    <view v-else class="input-bar">
      <view class="input-wrap">
        <input
          class="msg-input"
          v-model="inputText"
          placeholder="输入消息..."
          placeholder-style="color: #94A3B8"
          confirm-type="send"
          @confirm="onSend"
          @input="onInputChange"
          :adjust-position="true"
        />
      </view>
      <view class="send-btn" :class="{ 'send-btn-active': inputText.trim() }" @tap="onSend">
        <uni-icons type="paperplane" size="22" color="#FFFFFF" />
      </view>
    </view>
  </view>
</template>

<script>
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { ref, computed, nextTick, watch } from 'vue'
import { useChatStore } from '@/store/chat'
import { useGroupStore } from '@/store/group'
import { useUserStore } from '@/store/user'
import { GROUP_ROLE } from '@/constants/group'

export default {
  name: 'GroupConversation',
  setup() {
    const chatStore = useChatStore()
    const groupStore = useGroupStore()
    const userStore = useUserStore()

    const conversationId = ref(0)
    const groupId = ref(0)
    const groupName = ref('')
    const groupAvatar = ref('')
    const inputText = ref('')
    const scrollToId = ref('')
    const loadingMore = ref(false)
    const showAtPicker = ref(false)
    const atUserIds = ref([])

    const messages = computed(() => chatStore.currentMessages)
    const hasMore = computed(() => chatStore.hasMoreMap[conversationId.value] !== false)
    const selfAvatar = computed(() => userStore.userInfo?.avatar || '')
    const selfName = computed(() => userStore.userInfo?.nickname || userStore.userInfo?.username || '我')
    const myId = computed(() => Number(userStore.userInfo?.id) || 0)
    const memberCount = computed(() => groupStore.currentMembers.length || 0)

    /** 构建成员 ID → 信息的映射，用于显示发送者昵称 / 头像 */
    const memberMap = computed(() => {
      const map = {}
      for (const m of groupStore.currentMembers) {
        map[m.user_id] = m
      }
      return map
    })

    /** @选择器中排除自己的成员列表 */
    const atMemberList = computed(() => {
      return groupStore.currentMembers.filter(m => m.user_id !== myId.value)
    })

    /** 当前用户在群中的角色（用于判断撤回权限） */
    const myRole = computed(() => {
      const me = memberMap.value[myId.value]
      return me ? me.role : GROUP_ROLE.MEMBER
    })

    /** 当前群是否全体禁言 */
    const isAllMuted = computed(() => {
      return groupStore.currentGroup?.is_all_muted === true
    })

    /** 当前用户是否被禁言（个人禁言或全体禁言且非管理员/群主） */
    const isMuted = computed(() => {
      const me = memberMap.value[myId.value]
      if (!me) return false
      if (me.is_muted) return true
      if (isAllMuted.value && me.role === GROUP_ROLE.MEMBER) return true
      return false
    })

    const isSelf = (msg) => {
      return msg.sender_id === myId.value
    }

    const getMemberName = (userId) => {
      const member = memberMap.value[userId]
      if (member) return member.nickname || member.user_nickname || '未知'
      return '未知成员'
    }

    const getMemberAvatar = (userId) => {
      const member = memberMap.value[userId]
      return member ? member.avatar : ''
    }

    /** 群聊已读标签：自己发送的消息始终显示 "X人已读"（含 0 人） */
    const getReadLabel = (msg) => {
      if (msg.status === 2 || msg._sending || msg._failed || !msg.id) return ''
      if (msg.sender_id !== myId.value) return ''
      const count = chatStore.groupReadCountMap[msg.id] || 0
      return `${count}人已读`
    }

    // ==================== 生命周期 ====================

    onLoad(async (query) => {
      conversationId.value = parseInt(query.conversationId) || 0
      groupId.value = parseInt(query.groupId) || 0
      groupName.value = decodeURIComponent(query.peerName || '')
      groupAvatar.value = decodeURIComponent(query.peerAvatar || '')

      chatStore.initWsListeners()
      groupStore.initWsListeners()

      if (conversationId.value) {
        chatStore.setCurrentConversation(conversationId.value)
        loadInitialMessages()
      }

      if (!groupId.value && conversationId.value) {
        const conv = chatStore.conversationList.find(c => c.id === conversationId.value)
        if (conv && conv.group_id) {
          groupId.value = conv.group_id
        }
      }

      if (groupId.value) {
        groupStore.fetchMembers(groupId.value)
        groupStore.fetchGroupDetail(groupId.value)
      }
    })

    onUnload(() => {
      chatStore.setCurrentConversation(null)
    })

    watch(() => chatStore.currentConversationId, (newId) => {
      if (newId && newId !== conversationId.value) {
        conversationId.value = newId
      }
    })

    // ==================== 消息加载 ====================

    const loadInitialMessages = async () => {
      if (!chatStore.messagesMap[conversationId.value]?.length) {
        await chatStore.loadHistoryMessages(conversationId.value)
      }
      scrollToBottom()
      markVisibleMessagesRead()
    }

    const scrollToBottom = () => {
      nextTick(() => {
        scrollToId.value = ''
        nextTick(() => { scrollToId.value = 'msg-bottom' })
      })
    }

    /** 标记当前可见消息为已读 */
    const markVisibleMessagesRead = () => {
      const msgs = chatStore.messagesMap[conversationId.value] || []
      const messageIds = msgs.filter(m => m.id && m.sender_id !== myId.value).map(m => m.id)
      if (messageIds.length > 0) {
        chatStore.markGroupRead(conversationId.value, messageIds)
      }
    }

    // ==================== 消息发送 ====================

    const onSend = () => {
      const content = inputText.value.trim()
      if (!content) return
      chatStore.sendMessage({
        conversationId: conversationId.value,
        content,
        type: 1,
        at_user_ids: atUserIds.value.length > 0 ? [...atUserIds.value] : undefined
      })
      inputText.value = ''
      atUserIds.value = []
      scrollToBottom()
    }

    /** 输入变化检测：输入 @ 时弹出成员选择器 */
    const onInputChange = (e) => {
      const val = e.detail.value || inputText.value
      if (val.endsWith('@')) {
        showAtPicker.value = true
      }
    }

    /** 选择 @成员（null 表示 @所有人） */
    const onSelectAtMember = (member) => {
      showAtPicker.value = false
      if (member === null) {
        atUserIds.value.push(0)
        inputText.value = inputText.value + '所有人 '
      } else {
        atUserIds.value.push(member.user_id)
        const name = member.nickname || member.user_nickname || ''
        inputText.value = inputText.value + name + ' '
      }
    }

    // ==================== 加载更多 ====================

    const onLoadMore = async () => {
      if (loadingMore.value || !hasMore.value) return
      loadingMore.value = true
      try { await chatStore.loadHistoryMessages(conversationId.value) }
      finally { loadingMore.value = false }
    }

    // ==================== 消息撤回 ====================

    /**
     * 撤回权限判断
     * - 管理员/群主：可撤回任何人消息，无时间限制
     * - 普通成员：仅可撤回自己消息，2 分钟内
     */
    const canRecall = (msg) => {
      if (msg.status === 2 || !msg.id) return false
      const isAdmin = myRole.value === GROUP_ROLE.OWNER || myRole.value === GROUP_ROLE.ADMIN
      if (isAdmin) return true
      if (!isSelf(msg)) return false
      if (!msg.created_at) return false
      return (Date.now() - new Date(msg.created_at).getTime()) < 2 * 60 * 1000
    }

    const onMsgLongPress = (msg) => {
      if (msg.status === 2 || !canRecall(msg)) return
      uni.showActionSheet({
        itemList: ['撤回'],
        success: (res) => {
          if (res.tapIndex === 0 && msg.id) chatStore.recallMessage(msg.id)
        }
      })
    }

    const onResend = (msg) => {
      uni.showModal({
        title: '提示',
        content: '是否重新发送？',
        success: (res) => {
          if (res.confirm) {
            chatStore.sendMessage({
              conversationId: conversationId.value,
              content: msg.content,
              type: msg.type || 1
            })
          }
        }
      })
    }

    // ==================== 导航 ====================

    const goBack = () => {
      if (getCurrentPages().length > 1) {
        uni.navigateBack()
      } else {
        uni.switchTab({ url: '/pages/chat/index' })
      }
    }

    const goToSettings = () => {
      uni.navigateTo({
        url: `/pages/group/settings?groupId=${groupId.value}&conversationId=${conversationId.value}&peerName=${encodeURIComponent(groupName.value)}&peerAvatar=${encodeURIComponent(groupAvatar.value)}`
      })
    }

    const goToReadDetail = (msg) => {
      if (!msg.id) return
      uni.navigateTo({
        url: `/pages/chat/read-detail?messageId=${msg.id}`
      })
    }

    // ==================== 监听新消息到达后标记已读 ====================

    watch(
      () => messages.value.length,
      () => {
        scrollToBottom()
        markVisibleMessagesRead()
      }
    )

    return {
      GROUP_ROLE,
      groupName, groupAvatar, selfAvatar, selfName,
      inputText, scrollToId, loadingMore, memberCount,
      messages, hasMore, showAtPicker, atMemberList, isMuted, isAllMuted,
      isSelf, getMemberName, getMemberAvatar, getReadLabel,
      onSend, onInputChange, onSelectAtMember,
      onLoadMore, onMsgLongPress, onResend,
      goBack, goToSettings, goToReadDetail
    }
  }
}
</script>

<style scoped>
.page-wrapper {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background-color: #F1F5F9;
  overflow: hidden;
}

/* ===== 导航栏 ===== */
.nav-bar {
  display: flex;
  align-items: center;
  height: 88rpx;
  padding: 0 24rpx;
  padding-top: var(--status-bar-height, 44px);
  background-color: #FFFFFF;
  border-bottom: 1rpx solid #E2E8F0;
}
.nav-left, .nav-right {
  min-width: 88rpx;
  min-height: 88rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: opacity 150ms ease;
}
.nav-left:active, .nav-right:active { opacity: 0.6; }
.nav-center {
  flex: 1;
  display: flex;
  align-items: baseline;
  justify-content: center;
}
.nav-title { font-size: 32rpx; font-weight: 600; color: #1E293B; }
.nav-subtitle { font-size: 24rpx; color: #94A3B8; margin-left: 8rpx; }

/* ===== 消息列表 ===== */
.msg-list {
  flex: 1;
  height: 0;
  min-height: 0;
  padding: 16rpx 24rpx;
  box-sizing: border-box;
  width: 100%;
  overflow: hidden;
}
.load-more { text-align: center; padding: 16rpx 0; }
.load-more-text { font-size: 24rpx; color: #94A3B8; }

/* ===== 系统消息 ===== */
.msg-row-system {
  text-align: center;
  padding: 16rpx 48rpx;
  margin-bottom: 24rpx;
}
.system-text {
  font-size: 24rpx;
  color: #94A3B8;
  line-height: 36rpx;
  background-color: rgba(148, 163, 184, 0.12);
  padding: 8rpx 20rpx;
  border-radius: 12rpx;
}

/* ===== 消息行 ===== */
.msg-row {
  display: flex;
  align-items: flex-start;
  margin-bottom: 24rpx;
}
.msg-row-other { justify-content: flex-start; }
.msg-row-self { justify-content: flex-end; }

/* ===== 头像 ===== */
.avatar-wrap { flex-shrink: 0; }
.msg-row-other .avatar-wrap { margin-right: 16rpx; }
.msg-row-self .avatar-wrap { margin-left: 16rpx; }

.avatar-img {
  width: 72rpx;
  height: 72rpx;
  border-radius: 18rpx;
}
.avatar-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
}
.avatar-peer { background-color: #2563EB; }
.avatar-self { background-color: #64748B; }
.avatar-all { background-color: #F59E0B; }
.avatar-char { color: #FFFFFF; font-size: 28rpx; font-weight: 600; }

/* ===== 发送者昵称 + 气泡列 ===== */
.bubble-col {
  display: flex;
  flex-direction: column;
  max-width: 65vw;
}
.sender-name {
  font-size: 22rpx;
  color: #94A3B8;
  margin-bottom: 6rpx;
  line-height: 30rpx;
}

/* ===== 气泡 ===== */
.bubble {
  max-width: 65vw;
  padding: 20rpx 24rpx;
  border-radius: 24rpx;
  word-break: break-word;
  overflow-wrap: break-word;
}
.bubble-self {
  background-color: #2563EB;
  border-bottom-right-radius: 8rpx;
}
.bubble-other {
  background-color: #FFFFFF;
  border-bottom-left-radius: 8rpx;
}
.bubble-recalled {
  background-color: transparent !important;
  padding: 8rpx 16rpx;
}

.msg-text { font-size: 30rpx; line-height: 42rpx; color: #1E293B; }
.msg-text-self { color: #FFFFFF; }
.recalled-text { font-size: 24rpx; color: #94A3B8; font-style: italic; }

/* ===== 自己消息布局（含已读标记） ===== */
.self-msg-col {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}
.self-msg-row {
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
}
.read-label {
  font-size: 20rpx;
  margin-top: 4rpx;
  margin-right: 88rpx;
  color: #2563EB;
}

/* ===== 发送状态 ===== */
.msg-status {
  display: flex;
  align-items: center;
  align-self: center;
  margin: 0 8rpx;
}
.msg-status-tap {
  min-width: 44rpx;
  min-height: 44rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* ===== @成员选择器 ===== */
.at-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.35);
  z-index: 999;
  display: flex;
  align-items: flex-end;
}
.at-panel {
  width: 100%;
  max-height: 60vh;
  background-color: #FFFFFF;
  border-radius: 24rpx 24rpx 0 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.at-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24rpx 32rpx;
  border-bottom: 1rpx solid #E2E8F0;
}
.at-header-text { font-size: 30rpx; font-weight: 600; color: #1E293B; }
.at-close {
  min-width: 44rpx;
  min-height: 44rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}
.at-member-list {
  flex: 1;
  max-height: 50vh;
  padding-bottom: env(safe-area-inset-bottom, 0);
}
.at-member-item {
  display: flex;
  align-items: center;
  padding: 20rpx 32rpx;
  transition: background-color 150ms ease;
}
.at-member-item:active { background-color: #F8FAFC; }
.at-avatar { width: 64rpx; height: 64rpx; border-radius: 16rpx; }
.at-member-name {
  flex: 1;
  font-size: 28rpx;
  color: #1E293B;
  margin-left: 20rpx;
}
.at-role-tag {
  font-size: 22rpx;
  color: #2563EB;
  background-color: rgba(37, 99, 235, 0.08);
  padding: 4rpx 12rpx;
  border-radius: 8rpx;
  margin-left: 12rpx;
}

/* ===== 输入栏 ===== */
.input-bar {
  display: flex;
  align-items: center;
  padding: 16rpx 24rpx;
  padding-bottom: calc(16rpx + env(safe-area-inset-bottom, 0));
  background-color: #FFFFFF;
  border-top: 1rpx solid #E2E8F0;
}
.input-wrap {
  flex: 1;
  background-color: #F1F5F9;
  border-radius: 36rpx;
  padding: 0 28rpx;
  height: 72rpx;
  display: flex;
  align-items: center;
}
.msg-input { flex: 1; font-size: 28rpx; color: #1E293B; }
.send-btn {
  min-width: 72rpx;
  min-height: 72rpx;
  margin-left: 16rpx;
  border-radius: 50%;
  background-color: #CBD5E1;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 200ms ease;
}
.send-btn-active { background-color: #2563EB; }
.send-btn:active { opacity: 0.85; }

/* ===== 禁言状态 ===== */
.input-bar-muted {
  justify-content: center;
}
.muted-tip {
  font-size: 26rpx;
  color: #94A3B8;
  text-align: center;
}
</style>
