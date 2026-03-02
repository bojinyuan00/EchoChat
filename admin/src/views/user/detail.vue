<!--
  管理端用户详情页面

  设计系统：design-system/echochat/pages/admin-user-list.md
  
  功能：
  - 基本信息展示（ID、用户名、邮箱、昵称、性别、手机、状态、注册/更新时间）
  - 角色管理（查看当前角色 + 分配新角色）
  - 状态切换（启用/禁用）
  - 会议记录 Tab（占位，后续完善）

  对应后端 API：
  - GET /api/v1/admin/users/:id
  - PUT /api/v1/admin/users/:id/status
  - PUT /api/v1/admin/users/:id/role
-->
<template>
  <div class="user-detail-page" v-loading="loading">
    <!-- 返回按钮 + 页面标题 -->
    <div class="page-header">
      <el-button text @click="goBack">
        <el-icon><ArrowLeft /></el-icon>
        返回列表
      </el-button>
      <h2 class="page-title">用户详情</h2>
    </div>

    <template v-if="userInfo">
      <!-- Tab 切换 -->
      <el-tabs v-model="activeTab" class="detail-tabs">
        <!-- 基本信息 Tab -->
        <el-tab-pane label="基本信息" name="info">
          <el-card shadow="never" class="info-card">
            <!-- 用户头部区域 -->
            <div class="user-header">
              <el-avatar :size="64" class="user-avatar">
                {{ avatarLetter }}
              </el-avatar>
              <div class="user-header-info">
                <h3 class="user-display-name">{{ userInfo.nickname || userInfo.username }}</h3>
                <span class="user-id">ID: {{ userInfo.id }}</span>
                <el-tag :type="getStatusType(userInfo.status)" size="small">
                  {{ userInfo.status_text }}
                </el-tag>
              </div>
              <div class="user-actions">
                <el-button
                  v-if="userInfo.status === 1"
                  type="danger"
                  @click="handleToggleStatus(2)"
                >
                  禁用账号
                </el-button>
                <el-button
                  v-else-if="userInfo.status === 2"
                  type="success"
                  @click="handleToggleStatus(1)"
                >
                  启用账号
                </el-button>
              </div>
            </div>

            <!-- 详细信息表格 -->
            <el-descriptions :column="2" border class="info-descriptions">
              <el-descriptions-item label="用户名">{{ userInfo.username }}</el-descriptions-item>
              <el-descriptions-item label="邮箱">{{ userInfo.email }}</el-descriptions-item>
              <el-descriptions-item label="昵称">{{ userInfo.nickname || '-' }}</el-descriptions-item>
              <el-descriptions-item label="性别">{{ getGenderText(userInfo.gender) }}</el-descriptions-item>
              <el-descriptions-item label="手机号">{{ userInfo.phone || '未绑定' }}</el-descriptions-item>
              <el-descriptions-item label="最后登录 IP">{{ userInfo.last_login_ip || '-' }}</el-descriptions-item>
              <el-descriptions-item label="最后登录时间">{{ userInfo.last_login_at || '从未登录' }}</el-descriptions-item>
              <el-descriptions-item label="注册时间">{{ userInfo.created_at }}</el-descriptions-item>
              <el-descriptions-item label="更新时间">{{ userInfo.updated_at }}</el-descriptions-item>
            </el-descriptions>
          </el-card>
        </el-tab-pane>

        <!-- 角色管理 Tab -->
        <el-tab-pane label="角色管理" name="roles">
          <el-card shadow="never" class="role-card">
            <div class="role-section">
              <h4 class="section-title">当前角色</h4>
              <div class="current-roles">
                <el-tag
                  v-for="role in userInfo.roles"
                  :key="role"
                  :type="getRoleTagType(role)"
                  size="large"
                  class="role-tag"
                >
                  {{ getRoleName(role) }}
                </el-tag>
                <span v-if="!userInfo.roles || userInfo.roles.length === 0" class="no-role">
                  暂未分配角色
                </span>
              </div>
            </div>

            <el-divider />

            <div class="role-section">
              <h4 class="section-title">分配角色</h4>
              <div class="assign-role-form">
                <el-select v-model="selectedRole" placeholder="选择角色" style="width: 200px">
                  <el-option label="普通用户" value="user" />
                  <el-option label="管理员" value="admin" />
                  <el-option label="超级管理员" value="super_admin" />
                </el-select>
                <el-button
                  type="primary"
                  :loading="roleLoading"
                  :disabled="!selectedRole"
                  @click="handleAssignRole"
                >
                  分配角色
                </el-button>
              </div>
              <p class="role-hint">
                分配角色将为该用户添加所选角色权限。已有角色不会受影响。
              </p>
            </div>
          </el-card>
        </el-tab-pane>

        <!-- 会议记录 Tab（占位） -->
        <el-tab-pane label="会议记录" name="meetings">
          <el-card shadow="never">
            <el-empty description="会议记录功能将在后续版本中实现" />
          </el-card>
        </el-tab-pane>
      </el-tabs>
    </template>

    <!-- 用户不存在 -->
    <el-empty v-else-if="!loading" description="用户不存在" />
  </div>
</template>

<script setup>
/**
 * 用户详情页面逻辑
 *
 * 数据流：
 * 1. 从路由参数获取 userId
 * 2. 调用 API 获取用户详情
 * 3. 支持状态切换和角色分配操作
 */
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getUserDetail, updateUserStatus, assignUserRole } from '@/api/user'

const route = useRoute()
const router = useRouter()

/** 加载状态 */
const loading = ref(false)

/** 用户详情数据 */
const userInfo = ref(null)

/** 当前 Tab */
const activeTab = ref('info')

/** 角色分配选中值 */
const selectedRole = ref('')

/** 角色分配加载状态 */
const roleLoading = ref(false)

/** 路由参数中的用户 ID */
const userId = computed(() => Number(route.params.id))

/** 头像占位字母 */
const avatarLetter = computed(() => {
  if (!userInfo.value) return '?'
  const name = userInfo.value.nickname || userInfo.value.username || '?'
  return name.charAt(0).toUpperCase()
})

/**
 * 获取状态 Tag 类型
 * @param {number} status
 */
const getStatusType = (status) => {
  const map = { 1: 'success', 2: 'danger', 3: 'info' }
  return map[status] || 'info'
}

/**
 * 获取性别文本
 * @param {number} gender
 */
const getGenderText = (gender) => {
  const map = { 0: '未知', 1: '男', 2: '女' }
  return map[gender] || '未知'
}

/**
 * 获取角色 Tag 类型
 * @param {string} role
 */
const getRoleTagType = (role) => {
  const map = { super_admin: 'danger', admin: 'warning', user: '' }
  return map[role] || 'info'
}

/**
 * 获取角色中文名称
 * @param {string} role
 */
const getRoleName = (role) => {
  const map = { super_admin: '超级管理员', admin: '管理员', user: '普通用户' }
  return map[role] || role
}

/**
 * 获取用户详情
 */
const fetchUserDetail = async () => {
  loading.value = true
  try {
    const res = await getUserDetail(userId.value)
    userInfo.value = res.data
  } catch (err) {
    console.error('获取用户详情失败:', err)
    userInfo.value = null
  } finally {
    loading.value = false
  }
}

/** 返回列表 */
const goBack = () => {
  router.push('/user/list')
}

/**
 * 切换用户状态
 * @param {number} newStatus
 */
const handleToggleStatus = async (newStatus) => {
  const action = newStatus === 2 ? '禁用' : '启用'
  try {
    await ElMessageBox.confirm(
      `确定要${action}该用户吗？`,
      `${action}确认`,
      { type: 'warning' }
    )

    await updateUserStatus(userId.value, newStatus)
    ElMessage.success(`${action}成功`)
    fetchUserDetail()
  } catch (err) {
    if (err !== 'cancel') {
      console.error(`${action}失败:`, err)
    }
  }
}

/**
 * 分配角色
 */
const handleAssignRole = async () => {
  if (!selectedRole.value) return

  roleLoading.value = true
  try {
    await assignUserRole(userId.value, selectedRole.value)
    ElMessage.success('角色分配成功')
    selectedRole.value = ''
    fetchUserDetail()
  } catch (err) {
    console.error('角色分配失败:', err)
  } finally {
    roleLoading.value = false
  }
}

onMounted(() => {
  fetchUserDetail()
})
</script>

<style scoped>
/*
 * 用户详情页样式
 * 设计系统：Data-Dense Dashboard
 * 清晰的信息层次、分区卡片、操作按钮明确
 */
.user-detail-page {
  padding: 0;
}

/* ---- 页面头部 ---- */
.page-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #1E293B;
  margin: 0;
}

/* ---- Tabs ---- */
.detail-tabs {
  margin-top: 8px;
}

/* ---- 用户头部区域 ---- */
.user-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding-bottom: 20px;
  border-bottom: 1px solid #E2E8F0;
}

.user-avatar {
  background-color: #2563EB;
  color: #FFFFFF;
  font-size: 24px;
  font-weight: 600;
  flex-shrink: 0;
}

.user-header-info {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-display-name {
  font-size: 18px;
  font-weight: 600;
  color: #1E293B;
  margin: 0;
}

.user-id {
  font-size: 13px;
  color: #94A3B8;
}

.user-actions {
  flex-shrink: 0;
}

/* ---- 信息描述 ---- */
.info-descriptions {
  margin-top: 8px;
}

.info-card {
  margin-bottom: 0;
}

/* ---- 角色管理 ---- */
.role-card {
  margin-bottom: 0;
}

.role-section {
  margin-bottom: 8px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: #1E293B;
  margin: 0 0 12px 0;
}

.current-roles {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.role-tag {
  font-size: 14px;
}

.no-role {
  color: #94A3B8;
  font-size: 14px;
}

.assign-role-form {
  display: flex;
  align-items: center;
  gap: 12px;
}

.role-hint {
  font-size: 12px;
  color: #94A3B8;
  margin-top: 8px;
  margin-bottom: 0;
}
</style>
