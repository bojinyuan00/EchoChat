<!--
  管理端群组列表页面

  设计系统：Data-Dense Dashboard 风格（与用户列表页保持一致）

  功能：
  - 搜索框（群名称关键词搜索）
  - 群组数据表格（Element Plus Table）
  - 分页组件
  - 操作按钮（查看详情、解散群聊）

  对应后端 API：GET /api/v1/admin/groups
-->
<template>
  <div class="group-list-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="page-title">群组管理</h2>
    </div>

    <!-- 搜索区 -->
    <el-card class="filter-card" shadow="never">
      <el-form :inline="true" class="filter-form" @submit.prevent="handleSearch">
        <el-form-item label="关键词">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索群名称"
            clearable
            style="width: 240px"
            @clear="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 群组数据表格 -->
    <el-card class="table-card" shadow="never">
      <el-table
        v-loading="loading"
        :data="groupList"
        stripe
        border
        style="width: 100%"
        row-class-name="table-row"
      >
        <el-table-column prop="id" label="ID" width="80" align="center" />

        <el-table-column prop="name" label="群名称" min-width="160">
          <template #default="{ row }">
            <span class="group-name-cell">{{ row.name }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="owner_name" label="群主" min-width="120" />

        <el-table-column label="成员" width="100" align="center">
          <template #default="{ row }">
            <span>{{ row.member_count }}/{{ row.max_members }}</span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 1" type="success" size="small">正常</el-tag>
            <el-tag v-else-if="row.status === 2" type="danger" size="small">已解散</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="全体禁言" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.is_all_muted" type="warning" size="small">是</el-tag>
            <el-tag v-else type="info" size="small">否</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="创建时间" width="180" align="center" />

        <el-table-column label="操作" width="200" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="viewDetail(row)">
              详情
            </el-button>
            <el-button
              v-if="row.status === 1"
              type="danger"
              link
              size="small"
              @click="handleDissolve(row)"
            >
              解散
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="fetchGroupList"
          @current-change="fetchGroupList"
        />
      </div>
    </el-card>

    <!-- 群组详情弹窗 -->
    <el-dialog v-model="detailVisible" title="群组详情" width="720px" destroy-on-close class="detail-dialog">
      <div v-loading="detailLoading" class="detail-content">
        <template v-if="currentGroup">
          <!-- 群头部卡片 -->
          <div class="detail-header">
            <div class="detail-avatar" :style="{ backgroundColor: avatarColor }">
              {{ currentGroup.name?.[0] || 'G' }}
            </div>
            <div class="detail-header-info">
              <div class="detail-header-name">
                <span class="detail-group-name">{{ currentGroup.name }}</span>
                <el-tag v-if="currentGroup.status === 1" type="success" size="small" effect="light">正常</el-tag>
                <el-tag v-else type="danger" size="small" effect="light">已解散</el-tag>
              </div>
              <div class="detail-header-meta">
                <span class="meta-item">
                  <el-icon :size="14"><User /></el-icon>
                  {{ currentGroup.member_count }}/{{ currentGroup.max_members }} 名成员
                </span>
                <el-divider direction="vertical" />
                <span class="meta-item">群主：{{ currentGroup.owner_name }}</span>
                <el-divider direction="vertical" />
                <span class="meta-item">ID: {{ currentGroup.id }}</span>
              </div>
            </div>
          </div>

          <!-- 群信息网格 -->
          <div class="detail-grid">
            <div class="detail-grid-item">
              <span class="grid-label">会话 ID</span>
              <span class="grid-value">{{ currentGroup.conversation_id }}</span>
            </div>
            <div class="detail-grid-item">
              <span class="grid-label">全体禁言</span>
              <span class="grid-value">
                <el-tag v-if="currentGroup.is_all_muted" type="warning" size="small" effect="plain">开启</el-tag>
                <span v-else class="text-muted">未开启</span>
              </span>
            </div>
            <div class="detail-grid-item">
              <span class="grid-label">可被搜索</span>
              <span class="grid-value">
                <el-tag v-if="currentGroup.is_searchable" type="success" size="small" effect="plain">开启</el-tag>
                <span v-else class="text-muted">关闭</span>
              </span>
            </div>
            <div class="detail-grid-item">
              <span class="grid-label">创建时间</span>
              <span class="grid-value">{{ currentGroup.created_at }}</span>
            </div>
          </div>

          <!-- 群公告 -->
          <div class="detail-section">
            <div class="section-header">
              <el-icon :size="16"><ChatLineSquare /></el-icon>
              <span>群公告</span>
            </div>
            <div class="notice-content" :class="{ 'notice-empty': !currentGroup.notice }">
              {{ currentGroup.notice || '暂无公告' }}
            </div>
          </div>

          <!-- 成员列表 -->
          <div class="detail-section">
            <div class="section-header">
              <el-icon :size="16"><UserFilled /></el-icon>
              <span>群成员（{{ currentGroup.members?.length || 0 }}）</span>
            </div>
            <el-table
              :data="currentGroup.members || []"
              stripe
              max-height="280"
              size="small"
              class="member-table"
            >
              <el-table-column prop="user_id" label="ID" width="60" align="center" />
              <el-table-column label="用户" min-width="140">
                <template #default="{ row }">
                  <div class="member-cell">
                    <div class="member-mini-avatar" :style="{ backgroundColor: getMemberColor(row.username || row.nickname) }">
                      {{ (row.username || row.nickname || '?')[0] }}
                    </div>
                    <span class="member-cell-name">{{ row.username || row.nickname || '-' }}</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="群昵称" min-width="100">
                <template #default="{ row }">
                  <span class="text-muted">{{ row.nickname || '-' }}</span>
                </template>
              </el-table-column>
              <el-table-column label="角色" width="90" align="center">
                <template #default="{ row }">
                  <el-tag v-if="row.role === 2" type="danger" size="small" effect="light" round>群主</el-tag>
                  <el-tag v-else-if="row.role === 1" type="warning" size="small" effect="light" round>管理员</el-tag>
                  <el-tag v-else size="small" effect="plain" round>成员</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="禁言" width="70" align="center">
                <template #default="{ row }">
                  <el-icon v-if="row.is_muted" color="#EF4444" :size="16"><CircleCloseFilled /></el-icon>
                  <span v-else class="text-muted">-</span>
                </template>
              </el-table-column>
              <el-table-column prop="joined_at" label="加入时间" width="150" align="center" />
            </el-table>
          </div>
        </template>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
/**
 * 管理端群组列表逻辑
 *
 * 数据流：API → 列表渲染 → 分页 / 搜索 → 详情弹窗
 */
import { ref, computed, onMounted } from 'vue'
import { Search, User, UserFilled, ChatLineSquare, CircleCloseFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getGroupList, getGroupDetail, dissolveGroup } from '@/api/group'

const AVATAR_COLORS = ['#2563EB', '#7C3AED', '#059669', '#D97706', '#DC2626', '#0891B2', '#4F46E5', '#BE185D']
const avatarColor = computed(() => {
  if (!currentGroup.value?.name) return AVATAR_COLORS[0]
  const idx = currentGroup.value.name.charCodeAt(0) % AVATAR_COLORS.length
  return AVATAR_COLORS[idx]
})
const getMemberColor = (name) => {
  if (!name) return '#94A3B8'
  return AVATAR_COLORS[name.charCodeAt(0) % AVATAR_COLORS.length]
}

const loading = ref(false)
const groupList = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)
const searchKeyword = ref('')

const detailVisible = ref(false)
const detailLoading = ref(false)
const currentGroup = ref(null)

/** 获取群组列表 */
const fetchGroupList = async () => {
  loading.value = true
  try {
    const res = await getGroupList({
      page: currentPage.value,
      page_size: pageSize.value,
      keyword: searchKeyword.value || undefined
    })
    groupList.value = res.data?.list || []
    total.value = res.data?.total || 0
  } catch (e) {
    ElMessage.error(e?.data?.message || '获取群组列表失败')
  } finally {
    loading.value = false
  }
}

/** 搜索 */
const handleSearch = () => {
  currentPage.value = 1
  fetchGroupList()
}

/** 重置搜索条件 */
const handleReset = () => {
  searchKeyword.value = ''
  currentPage.value = 1
  fetchGroupList()
}

/** 查看群组详情 */
const viewDetail = async (row) => {
  detailVisible.value = true
  detailLoading.value = true
  currentGroup.value = null
  try {
    const res = await getGroupDetail(row.id)
    currentGroup.value = res.data
  } catch (e) {
    ElMessage.error(e?.data?.message || '获取群组详情失败')
  } finally {
    detailLoading.value = false
  }
}

/** 解散群聊（二次确认） */
const handleDissolve = (row) => {
  ElMessageBox.confirm(
    `确定要解散群聊「${row.name}」吗？此操作不可恢复。`,
    '解散群聊',
    { confirmButtonText: '确定解散', cancelButtonText: '取消', type: 'warning' }
  ).then(async () => {
    try {
      await dissolveGroup(row.id)
      ElMessage.success('群聊已解散')
      fetchGroupList()
    } catch (e) {
      ElMessage.error(e?.data?.message || '解散群聊失败')
    }
  }).catch(() => {})
}

onMounted(() => {
  fetchGroupList()
})
</script>

<style scoped>
.group-list-page {
  padding: 0;
}

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

.filter-card {
  margin-bottom: 16px;
}

.filter-form {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.table-card {
  margin-bottom: 16px;
}

.group-name-cell {
  font-weight: 500;
  color: #1E293B;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}

.table-row {
  cursor: default;
}

/* ===== 群组详情弹窗 ===== */

.detail-content {
  min-height: 200px;
}

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
  width: 56px;
  height: 56px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  font-weight: 700;
  color: #FFFFFF;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.2);
}

.detail-header-info {
  flex: 1;
  min-width: 0;
}

.detail-header-name {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.detail-group-name {
  font-size: 18px;
  font-weight: 700;
  color: #0F172A;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.text-muted {
  color: #94A3B8;
}

.detail-section {
  margin-bottom: 20px;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: #334155;
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid #E2E8F0;
}

.notice-content {
  padding: 12px 16px;
  background: #F8FAFC;
  border-radius: 8px;
  font-size: 14px;
  color: #334155;
  line-height: 1.6;
  border: 1px solid #E2E8F0;
}

.notice-empty {
  color: #94A3B8;
  font-style: italic;
}

.member-table {
  border-radius: 8px;
  overflow: hidden;
}

.member-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.member-mini-avatar {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  color: #FFFFFF;
  flex-shrink: 0;
}

.member-cell-name {
  font-weight: 500;
  color: #1E293B;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
