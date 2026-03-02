<!--
  好友关系管理页面（管理端）

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / Danger #EF4444 / Text #1E293B
  ui-ux-pro-max 规范：数据表格 / 分页 / 确认删除 / loading 状态

  功能：
  - 好友关系列表（分页表格）
  - 删除好友关系（双向解除）
  - 搜索暂不支持（后端预留）
-->
<template>
  <div class="contact-manage-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="page-title">好友关系管理</h2>
    </div>

    <!-- 数据表格 -->
    <el-card shadow="never" class="table-card">
      <el-table
        v-loading="loading"
        :data="contactList"
        stripe
        border
        style="width: 100%"
        empty-text="暂无好友关系数据"
      >
        <el-table-column prop="id" label="ID" width="80" align="center" />
        <el-table-column label="用户 A" min-width="150">
          <template #default="{ row }">
            <div class="user-cell">
              <span class="user-id">#{{ row.user_id }}</span>
              <span class="username">{{ row.username || '--' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="用户 B" min-width="150">
          <template #default="{ row }">
            <div class="user-cell">
              <span class="user-id">#{{ row.friend_id }}</span>
              <span class="username">{{ row.friend_username || '--' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="120">
          <template #default="{ row }">
            {{ row.remark || '--' }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" min-width="160" />
        <el-table-column label="操作" width="120" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
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

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="fetchList"
          @current-change="fetchList"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAllContacts, deleteContact } from '@/api/contact'

const loading = ref(false)
const contactList = ref([])
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const getStatusType = (status) => {
  const map = { 0: 'warning', 1: 'success', 2: 'info', 3: 'danger' }
  return map[status] || 'info'
}

const getStatusText = (status) => {
  const map = { 0: '待确认', 1: '已通过', 2: '已拒绝', 3: '已拉黑' }
  return map[status] || '未知'
}

const fetchList = async () => {
  loading.value = true
  try {
    const res = await getAllContacts({
      page: pagination.page,
      page_size: pagination.pageSize
    })
    contactList.value = res.data?.list || []
    pagination.total = res.data?.total || 0
  } catch (e) {
    console.error('获取好友关系列表失败', e)
  } finally {
    loading.value = false
  }
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除用户 ${row.username || row.user_id} 与 ${row.friend_username || row.friend_id} 之间的好友关系吗？此操作将双向解除。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' }
    )
    await deleteContact(row.id)
    ElMessage.success('好友关系已删除')
    fetchList()
  } catch (err) {
    if (err !== 'cancel' && err !== 'close') {
      console.error('删除好友关系失败', err)
    }
  }
}

onMounted(() => {
  fetchList()
})
</script>

<style scoped>
.contact-manage-page {
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

.table-card {
  margin-bottom: 16px;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.user-id {
  font-size: 12px;
  color: #94A3B8;
  font-family: monospace;
}

.username {
  font-weight: 500;
  color: #1E293B;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}
</style>
