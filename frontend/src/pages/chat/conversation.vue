<!--
  聊天对话页

  布局规则：
  - 自己的消息：靠右对齐，顺序 [状态图标] [蓝色气泡] [自己头像]
  - 对方的消息：靠左对齐，顺序 [对方头像] [白色气泡]
  - 导航栏：返回 / 对方昵称 / 更多
-->
<template>
  <view class="page-wrapper">
    <!-- 导航栏 -->
    <view class="nav-bar">
      <view class="nav-left" @tap="goBack">
        <uni-icons type="back" size="20" color="#1E293B" />
      </view>
      <view class="nav-center">
        <text class="nav-title">{{ peerName || '聊天' }}</text>
        <text v-if="isTyping" class="nav-typing">正在输入...</text>
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
      @scrolltolower="onScrollToLower"
      @scroll="onScroll"
      :lower-threshold="150"
    >
      <view v-if="hasMore" class="load-more" @tap="onLoadMore">
        <text class="load-more-text">{{ loadingMore ? '加载中...' : '加载更多' }}</text>
      </view>

      <view
        v-for="msg in messages"
        :key="msg.client_msg_id || msg.id"
        :id="'msg-' + (msg.id || msg.client_msg_id)"
        class="msg-row"
        :class="isSelf(msg) ? 'msg-row-self' : 'msg-row-other'"
      >
        <!-- ====== 对方消息（左侧）：[头像] [气泡] ====== -->
        <template v-if="!isSelf(msg)">
          <view class="avatar-wrap">
            <image v-if="peerAvatar" class="avatar-img" :src="peerAvatar" mode="aspectFill" />
            <view v-else class="avatar-img avatar-placeholder avatar-peer">
              <text class="avatar-char">{{ (peerName || '?')[0] }}</text>
            </view>
          </view>
          <view
            class="bubble bubble-other"
            :class="{ 'bubble-recalled': msg.status === 2, 'bubble-media': msg.type === 2 }"
            @longpress="onMsgLongPress(msg)"
          >
            <MsgText v-if="msg.type === 1 || msg.type === 10" :msg="msg" :isSelf="false" />
            <MsgImage v-else-if="msg.type === 2" :msg="msg" :isSelf="false" />
            <MsgVoice v-else-if="msg.type === 3" :msg="msg" :isSelf="false" />
            <MsgFile v-else-if="msg.type === 5" :msg="msg" :isSelf="false" />
            <MsgText v-else :msg="msg" :isSelf="false" />
          </view>
        </template>

        <!-- ====== 自己消息（右侧）：[状态] [气泡] [头像] ====== -->
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
                :class="{ 'bubble-recalled': msg.status === 2, 'bubble-media': msg.type === 2 }"
                @longpress="onMsgLongPress(msg)"
              >
                <MsgText v-if="msg.type === 1 || msg.type === 10" :msg="msg" :isSelf="true" />
                <MsgImage v-else-if="msg.type === 2" :msg="msg" :isSelf="true" />
                <MsgVoice v-else-if="msg.type === 3" :msg="msg" :isSelf="true" />
                <MsgFile v-else-if="msg.type === 5" :msg="msg" :isSelf="true" />
                <MsgText v-else :msg="msg" :isSelf="true" />
              </view>
              <view class="avatar-wrap">
                <image v-if="selfAvatar" class="avatar-img" :src="selfAvatar" mode="aspectFill" />
                <view v-else class="avatar-img avatar-placeholder avatar-self">
                  <text class="avatar-char">{{ (selfName || '我')[0] }}</text>
                </view>
              </view>
            </view>
            <text v-if="getReadLabel(msg)" class="read-label" :class="isRead(msg) ? 'read-label-read' : 'read-label-unread'">
              {{ getReadLabel(msg) }}
            </text>
          </view>
        </template>
      </view>

      <view id="msg-bottom" style="height: 2rpx;" />
    </scroll-view>

    <!-- 新消息悬浮提示（仅当用户远离底部且有新消息时显示） -->
    <view v-if="newMsgCount > 0" class="new-msg-hint" @tap="jumpToBottom">
      <uni-icons type="bottom" size="14" color="#2563EB" />
      <text class="new-msg-hint-text">{{ newMsgCount }} 条新消息</text>
    </view>

    <!-- 输入栏 -->
    <view class="input-bar">
      <view class="voice-toggle-btn" @tap="toggleVoiceMode">
        <uni-icons :type="voiceMode ? 'compose' : 'mic'" :size="22" color="#64748B" />
      </view>
      <template v-if="voiceMode">
        <VoiceRecorder @recorded="onVoiceRecorded" />
      </template>
      <template v-else>
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
      </template>
      <view v-if="!voiceMode && inputText.trim()" class="send-btn send-btn-active" @tap="onSend">
        <uni-icons type="paperplane" size="22" color="#FFFFFF" />
      </view>
      <view v-else class="more-btn" @tap="toggleMorePanel">
        <uni-icons type="plusempty" :size="22" color="#64748B" />
      </view>
    </view>
    <MorePanel
      :visible="showMorePanel"
      @choose-image="onChooseImage"
      @choose-file="onChooseFile"
    />
  </view>
</template>

<script>
import { onLoad, onUnload } from '@dcloudio/uni-app'
import { ref, computed, nextTick, watch } from 'vue'
import { useChatStore } from '@/store/chat'
import { useUserStore } from '@/store/user'
import { uploadImage, uploadFile, uploadVoice } from '@/api/file'
import MsgText from '@/components/msg/MsgText.vue'
import MsgImage from '@/components/msg/MsgImage.vue'
import MsgVoice from '@/components/msg/MsgVoice.vue'
import MsgFile from '@/components/msg/MsgFile.vue'
import MorePanel from '@/components/chat/MorePanel.vue'
import VoiceRecorder from '@/components/chat/VoiceRecorder.vue'

export default {
  name: 'ChatConversation',
  components: { MsgText, MsgImage, MsgVoice, MsgFile, MorePanel, VoiceRecorder },
  setup() {
    const chatStore = useChatStore()
    const userStore = useUserStore()

    const conversationId = ref(0)
    const peerId = ref(0)
    const peerName = ref('')
    const peerAvatar = ref('')
    const inputText = ref('')
    const scrollToId = ref('')
    const loadingMore = ref(false)

    // 滚动位置感知：判断用户视图是否"贴近底部"
    // 策略：每次 @scroll 事件都对比"历史最大 scrollTop"（等价于贴底位置），
    // 差值 < 150 视为贴底。此方式不依赖 @scrolltolower 能否在程序化滚动后触发，
    // 对 scroll-into-view + 动态 scrollHeight 都能正确响应。
    const nearBottom = ref(true)
    // 用户未看到的新消息累计数（悬浮按钮显示用）
    const newMsgCount = ref(0)
    // 首次渲染完成前，避免滚动事件误判
    let scrollReady = false
    // 已记录的最大 scrollTop（scrollHeight 增大并滚到底时会同步增长）
    let maxScrollTop = 0

    const messages = computed(() => chatStore.currentMessages)
    const hasMore = computed(() => conversationId.value > 0 && chatStore.hasMoreMap[conversationId.value] !== false)
    const isTyping = computed(() => chatStore.typingMap[conversationId.value] || false)
    const selfAvatar = computed(() => userStore.userInfo?.avatar || '')
    const selfName = computed(() => userStore.userInfo?.nickname || userStore.userInfo?.username || '我')

    const convType = ref(1)
    const voiceMode = ref(false)
    const showMorePanel = ref(false)

    const isSelf = (msg) => {
      const myId = Number(userStore.userInfo?.id) || 0
      return msg.sender_id === myId || msg._sending === true || msg._failed === true
    }

    const isRead = (msg) => {
      if (convType.value === 2) return false
      if (msg.status === 2 || msg._sending || msg._failed || !msg.id) return false
      const lastReadId = chatStore.readStatusMap[conversationId.value] || 0
      return msg.id <= lastReadId
    }

    const getReadLabel = (msg) => {
      if (msg.status === 2 || msg._sending || msg._failed || !msg.id) return ''
      if (convType.value === 2) {
        const count = chatStore.groupReadCountMap[msg.id]
        return count > 0 ? `${count}人已读` : ''
      }
      return isRead(msg) ? '已读' : '未读'
    }

    const tryFindExistingConversation = async () => {
      try {
        await chatStore.fetchConversations()
        const existingConv = chatStore.conversationList.find(c => c.peer_user_id === peerId.value)
        if (existingConv) {
          conversationId.value = existingConv.id
          chatStore.setCurrentConversation(existingConv.id)
          await loadInitialMessages()
        }
      } catch (e) {
        console.warn('[Chat] 查找已有会话失败', e)
      }
    }

    onLoad((query) => {
      conversationId.value = parseInt(query.conversationId) || 0
      peerId.value = parseInt(query.peerId) || 0
      peerName.value = decodeURIComponent(query.peerName || '')
      peerAvatar.value = decodeURIComponent(query.peerAvatar || '')
      convType.value = parseInt(query.convType) || 1

      chatStore.initWsListeners()

      if (conversationId.value) {
        chatStore.setCurrentConversation(conversationId.value)
        loadInitialMessages()
      } else if (peerId.value) {
        tryFindExistingConversation()
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

    const loadInitialMessages = async () => {
      if (!chatStore.messagesMap[conversationId.value]?.length) {
        await chatStore.loadHistoryMessages(conversationId.value)
      }
      scrollToBottom()
      // 初次进入会话且消息已加载：等待渲染后启用滚动感知
      nextTick(() => { scrollReady = true })
    }

    const scrollToBottom = () => {
      nextTick(() => {
        scrollToId.value = ''
        nextTick(() => { scrollToId.value = 'msg-bottom' })
      })
    }

    /**
     * 滚动事件：维护 maxScrollTop，并据此更新 nearBottom
     * - scrollTop 持续增长（贴底滚动）→ maxScrollTop 同步增长 → 贴底
     * - scrollTop 相对 maxScrollTop 回退 > 150 → 用户主动上滑，视为远离底部
     */
    const onScroll = (e) => {
      if (!scrollReady) return
      const scrollTop = e.detail?.scrollTop
      if (typeof scrollTop !== 'number') return
      if (scrollTop > maxScrollTop) maxScrollTop = scrollTop
      const wasNear = nearBottom.value
      nearBottom.value = (maxScrollTop - scrollTop) < 150
      // 由"远离"回到"贴底"的瞬间，清空未读悬浮并标记已读
      if (!wasNear && nearBottom.value) {
        if (newMsgCount.value > 0) newMsgCount.value = 0
        if (conversationId.value) chatStore.markRead(conversationId.value)
      }
    }

    /** 滚动到达下边界（lower-threshold 内）：权威地置贴底 */
    const onScrollToLower = () => {
      nearBottom.value = true
      if (newMsgCount.value > 0) {
        newMsgCount.value = 0
      }
      if (conversationId.value) {
        chatStore.markRead(conversationId.value)
      }
    }

    /** 点击悬浮提示：滚到底部、清零计数、标记已读 */
    const jumpToBottom = () => {
      scrollToBottom()
      newMsgCount.value = 0
      nearBottom.value = true
      if (conversationId.value) {
        chatStore.markRead(conversationId.value)
      }
    }

    // 监听消息数组变化：只对「末尾新增的消息」做处理
    // 关键点：必须对比末尾消息标识，而非仅看 length——加载历史时从头部插入大量消息，
    // length 会增加但 tail 不变，不应计入 newMsgCount
    let lastTailKey = ''
    const getTailKey = (msg) => msg ? String(msg.id || msg.client_msg_id || '') : ''
    watch(() => messages.value.length, (newLen) => {
      if (!scrollReady) return
      const tail = messages.value[newLen - 1]
      const tailKey = getTailKey(tail)
      // 末尾消息未变（仅头部插入历史消息）→ 不处理
      if (!tailKey || tailKey === lastTailKey) {
        lastTailKey = tailKey || lastTailKey
        return
      }
      const prevKey = lastTailKey
      lastTailKey = tailKey
      // 首次初始化（从空数组到有消息）不触发逻辑，由 loadInitialMessages 负责滚动
      if (!prevKey) return

      // 仅对「对方发来的新消息」做累计；自己发出的消息在 onSend/onChoose 等处已显式滚动
      const myId = Number(userStore.userInfo?.id) || 0
      const fromOther = tail.sender_id && tail.sender_id !== myId
      if (!fromOther) return

      if (nearBottom.value) {
        scrollToBottom()
        if (conversationId.value) {
          chatStore.markRead(conversationId.value)
        }
      } else {
        newMsgCount.value += 1
      }
    })

    const onSend = () => {
      const content = inputText.value.trim()
      if (!content) return
      chatStore.sendMessage({
        conversationId: conversationId.value || 0,
        targetUserId: conversationId.value ? 0 : peerId.value,
        content,
        type: 1
      })
      inputText.value = ''
      scrollToBottom()
    }

    let typingTimer = null
    const onInputChange = () => {
      if (typingTimer) return
      if (conversationId.value) {
        chatStore.sendTyping(conversationId.value)
      }
      typingTimer = setTimeout(() => { typingTimer = null }, 3000)
    }

    const onLoadMore = async () => {
      if (loadingMore.value || !hasMore.value) return
      loadingMore.value = true
      try { await chatStore.loadHistoryMessages(conversationId.value) }
      finally { loadingMore.value = false }
    }

    const canRecall = (msg) => {
      if (!msg.created_at) return false
      return (Date.now() - new Date(msg.created_at).getTime()) < 2 * 60 * 1000
    }

    const onMsgLongPress = (msg) => {
      if (!isSelf(msg) || msg.status === 2 || !canRecall(msg)) return
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
            chatStore.sendMessage({ conversationId: conversationId.value, content: msg.content, type: msg.type || 1 })
          }
        }
      })
    }

    const goBack = () => {
      if (getCurrentPages().length > 1) {
        uni.navigateBack()
      } else {
        uni.switchTab({ url: '/pages/chat/index' })
      }
    }

    const toggleVoiceMode = () => {
      voiceMode.value = !voiceMode.value
      showMorePanel.value = false
    }

    const toggleMorePanel = () => {
      showMorePanel.value = !showMorePanel.value
      voiceMode.value = false
    }

    const onChooseImage = async () => {
      showMorePanel.value = false
      uni.chooseImage({
        count: 9,
        sizeType: ['compressed'],
        sourceType: ['album', 'camera'],
        success: async (res) => {
          const images = []
          for (const path of res.tempFilePaths) {
            try {
              uni.showLoading({ title: '上传中...' })
              const result = await uploadImage(path)
              images.push({
                url: result.url,
                thumbnail_url: result.thumbnail_url,
                width: result.width,
                height: result.height,
                size: result.size,
                file_name: result.file_name
              })
            } catch (e) {
              uni.showToast({ title: e?.message || '图片上传失败', icon: 'none' })
            } finally {
              uni.hideLoading()
            }
          }
          if (images.length > 0) {
            chatStore.sendImageMessage({
              conversationId: conversationId.value || 0,
              targetUserId: conversationId.value ? 0 : peerId.value,
              images
            })
            scrollToBottom()
          }
        }
      })
    }

    const onChooseFile = async () => {
      showMorePanel.value = false
      // #ifdef H5
      const input = document.createElement('input')
      input.type = 'file'
      input.onchange = async (e) => {
        const file = e.target.files[0]
        if (!file) return
        if (file.size > 50 * 1024 * 1024) {
          uni.showToast({ title: '文件大小不能超过 50MB', icon: 'none' })
          return
        }
        try {
          uni.showLoading({ title: '上传中...' })
          const result = await uploadFile(URL.createObjectURL(file), file.name)
          chatStore.sendFileMessage({
            conversationId: conversationId.value || 0,
            targetUserId: conversationId.value ? 0 : peerId.value,
            file: {
              url: result.url,
              // 优先使用前端原始文件名（H5 下 blob URL 会丢失原名）
              file_name: file.name || result.file_name,
              size: result.size,
              mime_type: file.type || 'application/octet-stream',
              ext: '.' + (file.name.split('.').pop() || '')
            }
          })
          scrollToBottom()
        } catch (e) {
          uni.showToast({ title: e?.message || '文件上传失败', icon: 'none' })
        } finally {
          uni.hideLoading()
        }
      }
      input.click()
      // #endif
      // #ifndef H5
      uni.chooseFile({
        count: 1,
        success: async (res) => {
          const file = res.tempFiles[0]
          if (file.size > 50 * 1024 * 1024) {
            uni.showToast({ title: '文件大小不能超过 50MB', icon: 'none' })
            return
          }
          try {
            uni.showLoading({ title: '上传中...' })
            const result = await uploadFile(file.path, file.name)
            chatStore.sendFileMessage({
              conversationId: conversationId.value || 0,
              targetUserId: conversationId.value ? 0 : peerId.value,
              file: {
                url: result.url,
                file_name: file.name || result.file_name,
                size: result.size,
                mime_type: file.type || 'application/octet-stream',
                ext: '.' + (file.name.split('.').pop() || '')
              }
            })
            scrollToBottom()
          } catch (e) {
            uni.showToast({ title: e?.message || '文件上传失败', icon: 'none' })
          } finally {
            uni.hideLoading()
          }
        }
      })
      // #endif
    }

    const onVoiceRecorded = async ({ tempFilePath, duration, blob, mimeType }) => {
      try {
        uni.showLoading({ title: '发送中...' })
        // H5 端：根据 mimeType 推断扩展名，传递 Blob 以走 fetch+FormData 精确上传
        const ext = mimeTypeToExt(mimeType)
        const fileName = ext ? `voice-${Date.now()}.${ext}` : undefined
        const result = await uploadVoice(tempFilePath, duration, fileName, blob)
        chatStore.sendVoiceMessage({
          conversationId: conversationId.value || 0,
          targetUserId: conversationId.value ? 0 : peerId.value,
          voice: {
            url: result.url,
            duration: result.duration || duration,
            size: result.size,
            file_name: result.file_name
          }
        })
        scrollToBottom()
      } catch (e) {
        uni.showToast({ title: e?.message || '语音发送失败', icon: 'none' })
      } finally {
        uni.hideLoading()
      }
    }

    // 根据 MediaRecorder 的 mimeType 推断文件扩展名
    const mimeTypeToExt = (mime) => {
      if (!mime) return ''
      const m = mime.toLowerCase()
      if (m.includes('webm')) return 'webm'
      if (m.includes('ogg')) return 'ogg'
      if (m.includes('mp4') || m.includes('aac')) return 'm4a'
      if (m.includes('mpeg') || m.includes('mp3')) return 'mp3'
      if (m.includes('wav')) return 'wav'
      return ''
    }

    const goToSettings = () => {
      uni.navigateTo({
        url: `/pages/chat/settings?conversationId=${conversationId.value}&peerId=${peerId.value}&peerName=${encodeURIComponent(peerName.value)}&peerAvatar=${encodeURIComponent(peerAvatar.value)}`
      })
    }

    return {
      peerName, peerAvatar, selfAvatar, selfName,
      inputText, scrollToId, loadingMore, convType,
      voiceMode, showMorePanel,
      messages, hasMore, isTyping,
      newMsgCount,
      isSelf, isRead, getReadLabel,
      onSend, onInputChange, onLoadMore,
      onScroll, onScrollToLower, jumpToBottom,
      onMsgLongPress, onResend, goBack, goToSettings,
      toggleVoiceMode, toggleMorePanel,
      onChooseImage, onChooseFile, onVoiceRecorded
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
.nav-center { flex: 1; text-align: center; }
.nav-title { font-size: 32rpx; font-weight: 600; color: #1E293B; }
.nav-typing { display: block; font-size: 22rpx; color: #2563EB; margin-top: 2rpx; }

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

/* ===== 消息行 ===== */
.msg-row {
  display: flex;
  align-items: flex-start;
  margin-bottom: 24rpx;
}
.msg-row-other {
  justify-content: flex-start;
}
.msg-row-self {
  justify-content: flex-end;
}

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
.avatar-char { color: #FFFFFF; font-size: 28rpx; font-weight: 600; }

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
}
.read-label-read { color: #2563EB; }
.read-label-unread { color: #94A3B8; }

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

.voice-toggle-btn, .more-btn {
  min-width: 72rpx;
  min-height: 72rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.voice-toggle-btn { margin-right: 8rpx; }
.more-btn { margin-left: 8rpx; }
.voice-toggle-btn:active, .more-btn:active { opacity: 0.6; }

.bubble-media {
  padding: 12rpx !important;
  background-color: transparent !important;
}

/* ===== 新消息悬浮提示（远离底部时展示，点击滚到底并标记已读） ===== */
.new-msg-hint {
  position: fixed;
  right: 32rpx;
  /* 输入栏约 104rpx + 底部安全区 + 16rpx 间距 */
  bottom: calc(120rpx + env(safe-area-inset-bottom, 0rpx));
  display: flex;
  align-items: center;
  padding: 12rpx 24rpx;
  background-color: #FFFFFF;
  border: 1rpx solid #E2E8F0;
  border-radius: 32rpx;
  box-shadow: 0 4rpx 16rpx rgba(15, 23, 42, 0.12);
  z-index: 10;
  transition: opacity 150ms ease, transform 150ms ease;
}
.new-msg-hint:active {
  opacity: 0.85;
  transform: translateY(1rpx);
}
.new-msg-hint-text {
  font-size: 24rpx;
  color: #2563EB;
  margin-left: 6rpx;
  font-weight: 500;
}
</style>
