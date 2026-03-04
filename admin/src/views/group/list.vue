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
    <el-dialog v-model="detailVisible" title="群组详情" width="700px" destroy-on-close>
      <div v-loading="detailLoading">
        <template v-if="currentGroup">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="群 ID">{{ currentGroup.id }}</el-descriptions-item>
            <el-descriptions-item label="会话 ID">{{ currentGroup.conversation_id }}</el-descriptions-item>
            <el-descriptions-item label="群名称">{{ currentGroup.name }}</el-descriptions-item>
            <el-descriptions-item label="群主">{{ currentGroup.owner_name }}</el-descriptions-item>
            <el-descriptions-item label="成员数">{{ currentGroup.member_count }}/{{ currentGroup.max_members }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag v-if="currentGroup.status === 1" type="success" size="small">正常</el-tag>
              <el-tag v-else type="danger" size="small">已解散</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="全体禁言">
              <el-tag v-if="currentGroup.is_all_muted" type="warning" size="small">是</el-tag>
              <el-tag v-else type="info" size="small">否</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="可搜索">
              {{ currentGroup.is_searchable ? '是' : '否' }}
            </el-descriptions-item>
            <el-descriptions-item label="创建时间" :span="2">{{ currentGroup.created_at }}</el-descriptions-item>
            <el-descriptions-item label="群公告" :span="2">
              {{ currentGroup.notice || '暂无公告' }}
            </el-descriptions-item>
          </el-descriptions>

          <!-- 成员列表 -->
          <h4 class="member-title">群成员（{{ currentGroup.members?.length || 0 }}）</h4>
          <el-table :data="currentGroup.members || []" stripe border max-height="300" size="small">
            <el-table-column prop="user_id" label="用户 ID" width="80" align="center" />
            <el-table-column prop="username" label="用户名" min-width="120" />
            <el-table-column prop="nickname" label="群昵称" min-width="120">
              <template #default="{ row }">
                {{ row.nickname || '-' }}
              </template>
            </el-table-column>
            <el-table-column label="角色" width="100" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.role === 2" type="danger" size="small">群主</el-tag>
                <el-tag v-else-if="row.role === 1" type="warning" size="small">管理员</el-tag>
                <el-tag v-else type="info" size="small">成员</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="禁言" width="80" align="center">
              <template #default="{ row }">
                <el-tag v-if="row.is_muted" type="danger" size="small">是</el-tag>
                <span v-else>-</span>
              </template>
            </el-table-column>
            <el-table-column prop="joined_at" label="加入时间" width="160" align="center" />
          </el-table>
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
import { ref, onMounted } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getGroupList, getGroupDetail, dissolveGroup } from '@/api/group'

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

.member-title {
  margin: 20px 0 12px;
  font-size: 15px;
  font-weight: 600;
  color: #1E293B;
}
</style>
