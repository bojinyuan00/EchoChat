<!--
  管理端用户列表页面

  设计系统：design-system/echochat/pages/admin-user-list.md
  覆盖 MASTER.md 的 Data-Dense Dashboard 风格
  
  功能：
  - 搜索框（用户名/邮箱关键词搜索）
  - 状态筛选下拉
  - 用户数据表格（Element Plus Table）
  - 分页组件
  - 操作按钮（查看详情、启用/禁用、创建用户）

  对应后端 API：GET /api/v1/admin/users
-->
<template>
  <div class="user-list-page">
    <!-- 页面标题 + 操作按钮 -->
    <div class="page-header">
      <h2 class="page-title">用户管理</h2>
      <el-button type="primary" @click="showCreateDialog = true">
        <el-icon><Plus /></el-icon>
        创建用户
      </el-button>
    </div>

    <!-- 搜索和筛选区 -->
    <el-card class="filter-card" shadow="never">
      <el-form :inline="true" class="filter-form" @submit.prevent="handleSearch">
        <el-form-item label="关键词">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索用户名或邮箱"
            clearable
            style="width: 240px"
            @clear="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </el-form-item>

        <el-form-item label="状态">
          <el-select
            v-model="searchStatus"
            placeholder="全部状态"
            clearable
            style="width: 140px"
            @change="handleSearch"
          >
            <el-option label="正常" :value="1" />
            <el-option label="禁用" :value="2" />
            <el-option label="注销" :value="3" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" @click="handleSearch">搜索</el-button>
          <el-button @click="handleReset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 用户数据表格 -->
    <el-card class="table-card" shadow="never">
      <el-table
        v-loading="loading"
        :data="userList"
        stripe
        border
        style="width: 100%"
        row-class-name="table-row"
        @row-click="handleRowClick"
      >
        <el-table-column prop="id" label="ID" width="80" align="center" />
        
        <el-table-column prop="username" label="用户名" min-width="120">
          <template #default="{ row }">
            <span class="username-cell">{{ row.username }}</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="email" label="邮箱" min-width="180" />
        
        <el-table-column prop="nickname" label="昵称" min-width="100" />
        
        <el-table-column prop="roles" label="角色" min-width="120">
          <template #default="{ row }">
            <el-tag
              v-for="role in row.roles"
              :key="role.code"
              :type="getRoleTagType(role.code)"
              size="small"
              class="role-tag"
            >
              {{ role.name }}
            </el-tag>
            <span v-if="!row.roles || row.roles.length === 0" class="no-role">未分配</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="status_text" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ row.status_text }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="last_login_at" label="最后登录" min-width="160">
          <template #default="{ row }">
            {{ row.last_login_at || '从未登录' }}
          </template>
        </el-table-column>
        
        <el-table-column prop="created_at" label="注册时间" min-width="160" />
        
        <el-table-column label="操作" width="200" fixed="right" align="center">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click.stop="goDetail(row.id)">
              详情
            </el-button>
            <template v-if="canManageRow(row)">
              <el-button
                v-if="row.status === 1"
                type="danger"
                link
                size="small"
                @click.stop="handleToggleStatus(row, 2)"
              >
                禁用
              </el-button>
              <el-button
                v-else-if="row.status === 2"
                type="success"
                link
                size="small"
                @click.stop="handleToggleStatus(row, 1)"
              >
                启用
              </el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="fetchUserList"
          @current-change="fetchUserList"
        />
      </div>
    </el-card>

    <!-- 创建用户对话框 -->
    <el-dialog
      v-model="showCreateDialog"
      title="创建用户"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="createFormRef"
        :model="createForm"
        :rules="createRules"
        label-width="80px"
      >
        <el-form-item label="用户名" prop="username">
          <el-input v-model="createForm.username" placeholder="3-50 个字符" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="createForm.email" placeholder="输入邮箱地址" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="createForm.password" type="password" placeholder="至少 6 个字符" show-password />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="createForm.nickname" placeholder="选填，默认使用用户名" />
        </el-form-item>
        <el-form-item label="角色" prop="role_code">
          <el-select v-model="createForm.role_code" placeholder="选择角色" style="width: 100%">
            <el-option
              v-for="role in assignableRoles"
              :key="role.code"
              :label="role.name"
              :value="role.code"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="createLoading" @click="handleCreateUser">确认创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
/**
 * 用户列表页面逻辑
 *
 * 数据流：
 * 1. 组件挂载时调用 fetchUserList 加载用户列表
 * 2. 搜索/筛选/分页变化时重新加载
 * 3. 操作（禁用/启用/创建）成功后刷新列表
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getUserList, updateUserStatus, createUser, getAllRoles } from '@/api/user'
import { useUserStore } from '@/store/user'

const router = useRouter()
const userStore = useUserStore()

/** 加载状态 */
const loading = ref(false)

/** 用户列表数据 */
const userList = ref([])

/** 搜索关键词 */
const searchKeyword = ref('')

/** 状态筛选 */
const searchStatus = ref(null)

/** 分页状态 */
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

/** 系统所有角色列表（从 API 获取，含 level） */
const allRoles = ref([])

/** 当前操作者的最高权限等级（最小 level 值） */
const adminMaxLevel = computed(() => {
  const adminRoleCodes = userStore.roles || []
  if (!allRoles.value.length || !adminRoleCodes.length) return 999
  let minLevel = 999
  for (const r of allRoles.value) {
    if (adminRoleCodes.includes(r.code) && r.level < minLevel) {
      minLevel = r.level
    }
  }
  return minLevel
})

/** 创建用户对话框中可选的角色列表（过滤掉高于操作者等级的角色） */
const assignableRoles = computed(() => {
  return allRoles.value.filter(r => r.level > adminMaxLevel.value)
})

/**
 * 判断操作者是否有权管理目标用户（level 严格小于）
 * @param {Object} row - 用户行数据（含 max_level）
 */
const canManageRow = (row) => {
  const targetLevel = (row.max_level == null) ? 999 : row.max_level
  return adminMaxLevel.value < targetLevel
}

/** 创建用户对话框显示状态 */
const showCreateDialog = ref(false)

/** 创建用户表单引用 */
const createFormRef = ref(null)

/** 创建用户加载状态 */
const createLoading = ref(false)

/** 创建用户表单数据 */
const createForm = reactive({
  username: '',
  email: '',
  password: '',
  nickname: '',
  role_code: 'user'
})

/** 创建用户表单校验规则 */
const createRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 50, message: '用户名长度为 3-50 个字符', trigger: 'blur' }
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 50, message: '密码长度为 6-50 个字符', trigger: 'blur' }
  ]
}

/**
 * 获取角色 Tag 类型（Element Plus tag type）
 * @param {string} role - 角色代码
 */
const getRoleTagType = (role) => {
  const map = { super_admin: 'danger', admin: 'warning', user: '' }
  return map[role] || 'info'
}

/**
 * 获取状态 Tag 类型
 * @param {number} status - 状态码
 */
const getStatusType = (status) => {
  const map = { 1: 'success', 2: 'danger', 3: 'info' }
  return map[status] || 'info'
}

/**
 * 获取用户列表
 * 根据当前搜索、筛选、分页条件查询
 */
const fetchUserList = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize
    }
    if (searchKeyword.value) {
      params.keyword = searchKeyword.value
    }
    if (searchStatus.value !== null && searchStatus.value !== undefined) {
      params.status = searchStatus.value
    }

    const res = await getUserList(params)
    userList.value = res.data?.list || []
    pagination.total = res.data?.total || 0
  } catch (err) {
    console.error('获取用户列表失败:', err)
  } finally {
    loading.value = false
  }
}

/** 搜索按钮点击 */
const handleSearch = () => {
  pagination.page = 1
  fetchUserList()
}

/** 重置搜索条件 */
const handleReset = () => {
  searchKeyword.value = ''
  searchStatus.value = null
  pagination.page = 1
  fetchUserList()
}

/** 点击行跳转详情 */
const handleRowClick = (row) => {
  router.push(`/user/detail/${row.id}`)
}

/** 跳转用户详情页 */
const goDetail = (id) => {
  router.push(`/user/detail/${id}`)
}

/**
 * 切换用户状态（启用/禁用）
 * @param {Object} row - 用户行数据
 * @param {number} newStatus - 目标状态
 */
const handleToggleStatus = async (row, newStatus) => {
  const action = newStatus === 2 ? '禁用' : '启用'
  try {
    await ElMessageBox.confirm(
      `确定要${action}用户「${row.username}」吗？`,
      `${action}确认`,
      { type: 'warning' }
    )

    await updateUserStatus(row.id, newStatus)
    ElMessage.success(`${action}成功`)
    fetchUserList()
  } catch (err) {
    if (err !== 'cancel' && err !== 'close') {
      console.error(`${action}用户失败:`, err)
    }
  }
}

/**
 * 创建用户提交
 */
const handleCreateUser = async () => {
  if (!createFormRef.value) return

  try {
    await createFormRef.value.validate()
  } catch {
    return
  }

  createLoading.value = true
  try {
    await createUser(createForm)
    ElMessage.success('用户创建成功')
    showCreateDialog.value = false
    resetCreateForm()
    fetchUserList()
  } catch (err) {
    console.error('创建用户失败:', err)
  } finally {
    createLoading.value = false
  }
}

/** 重置创建表单 */
const resetCreateForm = () => {
  createForm.username = ''
  createForm.email = ''
  createForm.password = ''
  createForm.nickname = ''
  createForm.role_code = 'user'
}

/** 获取角色列表（用于权限判断和创建用户的角色选项） */
const fetchRoles = async () => {
  try {
    const res = await getAllRoles()
    allRoles.value = res.data || []
  } catch (err) {
    console.error('获取角色列表失败:', err)
  }
}

onMounted(() => {
  fetchUserList()
  fetchRoles()
})
</script>

<style scoped>
/*
 * 用户列表页样式
 * 设计系统：Data-Dense Dashboard 风格
 * 高信息密度、行悬停高亮、清晰的数据层次
 */
.user-list-page {
  padding: 0;
}

/* ---- 页面头部 ---- */
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

/* ---- 筛选区 ---- */
.filter-card {
  margin-bottom: 16px;
}

.filter-form {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 0;
}

/* ---- 表格区 ---- */
.table-card {
  margin-bottom: 16px;
}

.username-cell {
  font-weight: 500;
  color: #2563EB;
  cursor: pointer;
}

.role-tag {
  margin-right: 4px;
}

.no-role {
  color: #94A3B8;
  font-size: 12px;
}

:deep(.table-row) {
  cursor: pointer;
  transition: background-color 200ms ease;
}

:deep(.table-row:hover) {
  background-color: #EFF6FF !important;
}

/* ---- 分页 ---- */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}
</style>
