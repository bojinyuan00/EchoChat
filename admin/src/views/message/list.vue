<!--
  管理端消息列表页面

  设计系统：Data-Dense Dashboard 风格（与群组/用户列表页保持一致）

  功能：
  - 多条件筛选：关键词 / 消息类型 / 发送者 / 状态 / 时间范围
  - 消息数据表格（Element Plus Table）
  - 分页组件
  - 操作按钮（查看详情 / 撤回 / 删除）
  - 消息详情弹窗（含富媒体预览）

  对应后端 API：
  - GET    /api/v1/admin/messages
  - GET    /api/v1/admin/messages/:id
  - DELETE /api/v1/admin/messages/:id
  - PUT    /api/v1/admin/messages/:id/recall
-->
<template>
  <div class="message-list-page">
    <div class="page-header">
      <h2 class="page-title">消息管理</h2>
    </div>

    <!-- 筛选区 -->
    <el-card class="filter-card" shadow="never">
      <el-form :inline="true" class="filter-form" @submit.prevent="handleSearch">
        <el-form-item label="关键词">
          <el-input
            v-model="filters.keyword"
            placeholder="搜索消息内容"
            clearable
            style="width: 200px"
            @clear="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="类型">
          <el-select v-model="filters.type" placeholder="全部" clearable style="width: 120px">
            <el-option label="文本" :value="1" />
            <el-option label="图片" :value="2" />
            <el-option label="语音" :value="3" />
            <el-option label="文件" :value="5" />
            <el-option label="系统" :value="10" />
          </el-select>
        </el-form-item>

        <el-form-item label="状态">
          <el-select v-model="filters.status" placeholder="全部" clearable style="width: 120px">
            <el-option label="正常" :value="1" />
            <el-option label="已撤回" :value="2" />
            <el-option label="已删除" :value="3" />
          </el-select>
        </el-form-item>

        <el-form-item label="发送者ID">
          <el-input
            v-model="filters.sender_id"
            placeholder="用户ID"
            clearable
            style="width: 120px"
            type="number"
          />
        </el-form-item>

        <el-form-item label="时间范围">
          <el-date-picker
            v-model="dateRange"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            value-format="YYYY-MM-DD"
            style="width: 260px"
            @change="onDateRangeChange"
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 数据表格 -->
    <el-card class="table-card" shadow="never">
      <el-table
        v-loading="loading"
        :data="messageList"
        stripe
        border
        style="width: 100%"
        row-class-name="table-row"
      >
        <el-table-column prop="id" label="ID" width="80" align="center" />

        <el-table-column label="发送者" min-width="150">
          <template #default="{ row }">
            <div class="sender-cell">
              <div
                class="sender-avatar"
                :style="{ backgroundColor: getAvatarColor(row.sender_nickname) }"
              >
                <img v-if="row.sender_avatar" :src="row.sender_avatar" class="sender-avatar-img" />
                <span v-else>{{ (row.sender_nickname || '?')[0] }}</span>
              </div>
              <div class="sender-info">
                <span class="sender-name">{{ row.sender_nickname || '-' }}</span>
                <span class="sender-id">ID: {{ row.sender_id }}</span>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="类型" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="typeTagMap[row.type]?.type || 'info'" size="small" effect="light">
              {{ row.type_label || typeTagMap[row.type]?.label || '未知' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="内容" min-width="240">
          <template #default="{ row }">
            <span class="content-preview">{{ getContentPreview(row) }}</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagMap[row.status]?.type || 'info'" size="small">
              {{ row.status_label || statusTagMap[row.status]?.label || '未知' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="conversation_id" label="会话ID" width="90" align="center" />

        <el-table-column prop="created_at" label="发送时间" width="180" align="center" />

        <el-table-column label="操作" width="200" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="viewDetail(row)">
              详情
            </el-button>
            <el-button
              v-if="row.status === 1"
              type="warning"
              link
              size="small"
              @click="handleRecall(row)"
            >
              撤回
            </el-button>
            <el-button
              v-if="row.status !== 3"
              type="danger"
              link
              size="small"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="fetchMessageList"
          @current-change="fetchMessageList"
        />
      </div>
    </el-card>

    <!-- 消息详情弹窗 -->
    <el-dialog
      v-model="detailVisible"
      title="消息详情"
      width="640px"
      destroy-on-close
      class="detail-dialog"
    >
      <div v-loading="detailLoading" class="detail-content">
        <template v-if="currentMsg">
          <!-- 消息头部 -->
          <div class="detail-header">
            <div
              class="detail-avatar"
              :style="{ backgroundColor: getAvatarColor(currentMsg.sender_nickname) }"
            >
              <img
                v-if="currentMsg.sender_avatar"
                :src="currentMsg.sender_avatar"
                class="detail-avatar-img"
              />
              <span v-else>{{ (currentMsg.sender_nickname || '?')[0] }}</span>
            </div>
            <div class="detail-header-info">
              <div class="detail-header-name">
                <span class="detail-sender-name">{{ currentMsg.sender_nickname }}</span>
                <el-tag
                  :type="statusTagMap[currentMsg.status]?.type || 'info'"
                  size="small"
                  effect="light"
                >
                  {{ currentMsg.status_label }}
                </el-tag>
              </div>
              <div class="detail-header-meta">
                <span class="meta-item">ID: {{ currentMsg.id }}</span>
                <el-divider direction="vertical" />
                <span class="meta-item">发送者ID: {{ currentMsg.sender_id }}</span>
                <el-divider direction="vertical" />
                <span class="meta-item">会话ID: {{ currentMsg.conversation_id }}</span>
              </div>
            </div>
          </div>

          <!-- 消息元数据 -->
          <div class="detail-grid">
            <div class="detail-grid-item">
              <span class="grid-label">消息类型</span>
              <span class="grid-value">
                <el-tag
                  :type="typeTagMap[currentMsg.type]?.type || 'info'"
                  size="small"
                  effect="plain"
                >
                  {{ currentMsg.type_label }}
                </el-tag>
              </span>
            </div>
            <div class="detail-grid-item">
              <span class="grid-label">发送时间</span>
              <span class="grid-value">{{ currentMsg.created_at }}</span>
            </div>
          </div>

          <!-- 消息内容 -->
          <div class="detail-section">
            <div class="section-header">消息内容</div>
            <div class="msg-content-box">
              <template v-if="currentMsg.type === 1 || currentMsg.type === 10">
                <p class="msg-text-content">{{ currentMsg.content }}</p>
              </template>
              <template v-else-if="currentMsg.type === 2 && parsedExtra">
                <div class="msg-image-grid">
                  <img
                    v-for="(img, idx) in parsedExtra.images || []"
                    :key="idx"
                    :src="img.thumbnail_url || img.url"
                    class="msg-image-thumb"
                    @click="previewImage(img.url)"
                  />
                </div>
                <p v-if="currentMsg.content" class="msg-text-content">{{ currentMsg.content }}</p>
              </template>
              <template v-else-if="currentMsg.type === 3 && parsedExtra">
                <div class="msg-voice-info">
                  <el-icon :size="20"><Microphone /></el-icon>
                  <span>语音消息 {{ parsedExtra.duration || 0 }}"</span>
                  <a v-if="parsedExtra.url" :href="parsedExtra.url" target="_blank" class="voice-link">播放</a>
                </div>
              </template>
              <template v-else-if="currentMsg.type === 5 && parsedExtra">
                <div class="msg-file-info">
                  <el-icon :size="20"><Document /></el-icon>
                  <div class="file-meta">
                    <span class="file-name">{{ parsedExtra.file_name || '未知文件' }}</span>
                    <span class="file-size">{{ formatFileSize(parsedExtra.size || 0) }}</span>
                  </div>
                  <a v-if="parsedExtra.url" :href="parsedExtra.url" target="_blank" class="file-link">
                    下载
                  </a>
                </div>
              </template>
              <template v-else>
                <p class="msg-text-content">{{ currentMsg.content || '（无文本内容）' }}</p>
              </template>
            </div>
          </div>

          <!-- extra 原始数据（可折叠） -->
          <div v-if="currentMsg.extra" class="detail-section">
            <el-collapse>
              <el-collapse-item title="Extra 原始数据">
                <pre class="extra-raw">{{ formatJSON(currentMsg.extra) }}</pre>
              </el-collapse-item>
            </el-collapse>
          </div>
        </template>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
/**
 * 管理端消息列表逻辑
 *
 * 数据流：API → 列表渲染 → 筛选 / 分页 → 详情弹窗 → 撤回 / 删除
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { Search, Microphone, Document } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getMessageList, getMessageDetail, deleteMessage, recallMessage } from '@/api/message'

const AVATAR_COLORS = ['#2563EB', '#7C3AED', '#059669', '#D97706', '#DC2626', '#0891B2', '#4F46E5', '#BE185D']

const typeTagMap = {
  1: { label: '文本', type: '' },
  2: { label: '图片', type: 'success' },
  3: { label: '语音', type: 'warning' },
  5: { label: '文件', type: 'primary' },
  10: { label: '系统', type: 'info' }
}

const statusTagMap = {
  1: { label: '正常', type: 'success' },
  2: { label: '已撤回', type: 'warning' },
  3: { label: '已删除', type: 'danger' }
}

const loading = ref(false)
const messageList = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const dateRange = ref(null)

const filters = reactive({
  keyword: '',
  type: null,
  status: null,
  sender_id: null
})

const detailVisible = ref(false)
const detailLoading = ref(false)
const currentMsg = ref(null)

const parsedExtra = computed(() => {
  if (!currentMsg.value?.extra) return null
  try {
    return JSON.parse(currentMsg.value.extra)
  } catch {
    return null
  }
})

const getAvatarColor = (name) => {
  if (!name) return '#94A3B8'
  return AVATAR_COLORS[name.charCodeAt(0) % AVATAR_COLORS.length]
}

const getContentPreview = (row) => {
  if (row.status === 2) return '[已撤回]'
  if (row.status === 3) return '[已删除]'
  if (row.type === 2) return '[图片]'
  if (row.type === 3) return `[语音]`
  if (row.type === 5) {
    try {
      const extra = JSON.parse(row.extra || '{}')
      return `[文件] ${extra.file_name || ''}`
    } catch {
      return '[文件]'
    }
  }
  return row.content || ''
}

const formatFileSize = (bytes) => {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

const formatJSON = (str) => {
  try {
    return JSON.stringify(JSON.parse(str), null, 2)
  } catch {
    return str
  }
}

const previewImage = (url) => {
  window.open(url, '_blank')
}

const onDateRangeChange = (val) => {
  if (!val) {
    filters.start_time = undefined
    filters.end_time = undefined
  }
}

/** 获取消息列表 */
const fetchMessageList = async () => {
  loading.value = true
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    if (filters.keyword) params.keyword = filters.keyword
    if (filters.type != null) params.type = filters.type
    if (filters.status != null) params.status = filters.status
    if (filters.sender_id) params.sender_id = Number(filters.sender_id)
    if (dateRange.value && dateRange.value.length === 2) {
      params.start_time = dateRange.value[0]
      params.end_time = dateRange.value[1]
    }

    const res = await getMessageList(params)
    messageList.value = res.data?.list || []
    total.value = res.data?.total || 0
  } catch (e) {
    ElMessage.error(e?.message || '获取消息列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  fetchMessageList()
}

const handleReset = () => {
  filters.keyword = ''
  filters.type = null
  filters.status = null
  filters.sender_id = null
  dateRange.value = null
  currentPage.value = 1
  fetchMessageList()
}

/** 查看消息详情 */
const viewDetail = async (row) => {
  detailVisible.value = true
  detailLoading.value = true
  currentMsg.value = null
  try {
    const res = await getMessageDetail(row.id)
    currentMsg.value = res.data
  } catch (e) {
    ElMessage.error(e?.message || '获取消息详情失败')
  } finally {
    detailLoading.value = false
  }
}

/** 撤回消息 */
const handleRecall = (row) => {
  ElMessageBox.confirm(
    `确定要撤回消息 #${row.id} 吗？撤回后该消息对所有用户不可见。`,
    '撤回消息',
    { confirmButtonText: '确定撤回', cancelButtonText: '取消', type: 'warning' }
  ).then(async () => {
    try {
      await recallMessage(row.id)
      ElMessage.success('消息已撤回')
      fetchMessageList()
    } catch (e) {
      ElMessage.error(e?.message || '撤回失败')
    }
  }).catch(() => {})
}

/** 删除消息 */
const handleDelete = (row) => {
  ElMessageBox.confirm(
    `确定要删除消息 #${row.id} 吗？`,
    '删除消息',
    { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
  ).then(async () => {
    try {
      await deleteMessage(row.id)
      ElMessage.success('消息已删除')
      fetchMessageList()
    } catch (e) {
      ElMessage.error(e?.message || '删除失败')
    }
  }).catch(() => {})
}

onMounted(() => {
  fetchMessageList()
})
</script>

<style scoped>
.message-list-page { padding: 0; }

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #1E293B;
  margin: 0;
}

.filter-card { margin-bottom: 16px; }
.filter-form {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.table-card { margin-bottom: 16px; }

/* 发送者单元格 */
.sender-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}
.sender-avatar {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  color: #FFFFFF;
  flex-shrink: 0;
  overflow: hidden;
}
.sender-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.sender-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.sender-name {
  font-weight: 500;
  color: #1E293B;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sender-id {
  font-size: 11px;
  color: #94A3B8;
}

.content-preview {
  font-size: 13px;
  color: #475569;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}

/* ===== 消息详情弹窗 ===== */
.detail-content { min-height: 200px; }

.detail-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: linear-gradient(135deg, #F8FAFC 0%, #EFF6FF 100%);
  border-radius: 12px;
  margin-bottom: 20px;
}
.detail-avatar {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 700;
  color: #FFFFFF;
  flex-shrink: 0;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.2);
}
.detail-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.detail-header-info { flex: 1; min-width: 0; }
.detail-header-name {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.detail-sender-name {
  font-size: 16px;
  font-weight: 700;
  color: #0F172A;
}
.detail-header-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #64748B;
}
.meta-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
  margin-bottom: 20px;
}
.detail-grid-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 16px;
  background: #F8FAFC;
  border-radius: 8px;
  border: 1px solid #E2E8F0;
}
.grid-label {
  font-size: 12px;
  color: #94A3B8;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}
.grid-value {
  font-size: 14px;
  color: #1E293B;
  font-weight: 500;
}

.detail-section { margin-bottom: 20px; }
.section-header {
  font-size: 14px;
  font-weight: 600;
  color: #334155;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid #E2E8F0;
}

.msg-content-box {
  padding: 16px;
  background: #F8FAFC;
  border-radius: 8px;
  border: 1px solid #E2E8F0;
}
.msg-text-content {
  margin: 0;
  font-size: 14px;
  color: #334155;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.msg-image-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}
.msg-image-thumb {
  width: 120px;
  height: 120px;
  object-fit: cover;
  border-radius: 8px;
  cursor: pointer;
  border: 1px solid #E2E8F0;
  transition: transform 0.2s;
}
.msg-image-thumb:hover { transform: scale(1.05); }

.msg-voice-info, .msg-file-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
}
.file-meta {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}
.file-name {
  font-size: 14px;
  font-weight: 500;
  color: #1E293B;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-size {
  font-size: 12px;
  color: #94A3B8;
}
.voice-link, .file-link {
  color: #2563EB;
  font-size: 13px;
  text-decoration: none;
}
.voice-link:hover, .file-link:hover { text-decoration: underline; }

.extra-raw {
  font-size: 12px;
  color: #64748B;
  background: #F1F5F9;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  max-height: 200px;
  margin: 0;
}
</style>
