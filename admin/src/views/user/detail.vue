<!--
  管理端用户详情页面

  设计系统：design-system/echochat/pages/admin-user-list.md
  
  功能：
  - 基本信息展示（ID、用户名、邮箱、昵称、性别、手机、状态、注册/更新时间）
  - 角色管理（Checkbox Group 多选，权限等级管控）
  - 状态切换（启用/禁用，受角色层级约束）
  - 会议记录 Tab（占位，后续完善）

  对应后端 API：
  - GET /api/v1/admin/users/:id
  - PUT /api/v1/admin/users/:id/status
  - PUT /api/v1/admin/users/:id/roles
  - GET /api/v1/admin/roles
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
              <div class="user-actions" v-if="canManageTarget">
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
              <div class="user-actions" v-else>
                <el-tag type="info" size="small">权限不足，无法操作该用户</el-tag>
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
              <h4 class="section-title">角色分配</h4>
              <p class="role-hint" v-if="!canManageTarget" style="margin-bottom: 12px">
                该用户权限等级高于或等于您，无法修改其角色。
              </p>
              <el-checkbox-group v-model="selectedRoleCodes" class="role-checkbox-group">
                <el-checkbox
                  v-for="role in allRoles"
                  :key="role.code"
                  :value="role.code"
                  :disabled="isRoleDisabled(role)"
                  class="role-checkbox-item"
                >
                  <div class="role-checkbox-label">
                    <el-tag :type="getRoleTagType(role.code)" size="small">{{ role.name }}</el-tag>
                    <span class="role-level-badge">Lv.{{ role.level }}</span>
                  </div>
                </el-checkbox>
              </el-checkbox-group>
              <p class="role-hint" style="margin-top: 12px">
                灰色不可选的角色等级高于或等于您的权限，无法分配。勾选后点击"保存"生效。
              </p>
            </div>

            <div class="role-actions" v-if="canManageTarget">
              <el-button
                type="primary"
                :loading="roleLoading"
                :disabled="!hasRoleChanges"
                @click="handleSaveRoles"
              >
                保存角色
              </el-button>
              <el-button @click="resetRoleSelection">重置</el-button>
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
 * 2. 并行调用 getUserDetail + getAllRoles 获取用户详情和角色列表
 * 3. 通过比较操作者 level 和目标用户 level 控制权限
 * 4. 角色分配使用 Checkbox Group 多选，高等级角色禁用
 */
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getUserDetail, updateUserStatus, setUserRoles, getAllRoles } from '@/api/user'
import { useUserStore } from '@/store/user'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const loading = ref(false)
const userInfo = ref(null)
const activeTab = ref('info')
const roleLoading = ref(false)

/** 系统所有角色列表（从 API 获取，含 level） */
const allRoles = ref([])

/** Checkbox Group 绑定值：选中的角色 code 列表 */
const selectedRoleCodes = ref([])

/** 记录用户原始角色 codes，用于比较是否有变更 */
const originalRoleCodes = ref([])

const userId = computed(() => Number(route.params.id))

const avatarLetter = computed(() => {
  if (!userInfo.value) return '?'
  const name = userInfo.value.nickname || userInfo.value.username || '?'
  return name.charAt(0).toUpperCase()
})

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

/** 目标用户的最高权限等级（后端无角色时返回 MaxInt32，前端以 999 作为等效兜底） */
const targetMaxLevel = computed(() => {
  if (!userInfo.value || userInfo.value.max_level == null) return 999
  return userInfo.value.max_level
})

/** 操作者是否有权管理目标用户（level 严格小于） */
const canManageTarget = computed(() => {
  return adminMaxLevel.value < targetMaxLevel.value
})

/** Checkbox 是否有变更 */
const hasRoleChanges = computed(() => {
  if (selectedRoleCodes.value.length !== originalRoleCodes.value.length) return true
  const sorted1 = [...selectedRoleCodes.value].sort()
  const sorted2 = [...originalRoleCodes.value].sort()
  return sorted1.some((v, i) => v !== sorted2[i])
})

const getStatusType = (status) => {
  const map = { 1: 'success', 2: 'danger', 3: 'info' }
  return map[status] || 'info'
}

const getGenderText = (gender) => {
  const map = { 0: '未知', 1: '男', 2: '女' }
  return map[gender] || '未知'
}

const getRoleTagType = (role) => {
  const map = { super_admin: 'danger', admin: 'warning', user: '' }
  return map[role] || 'info'
}

/**
 * 判断角色 checkbox 是否禁用
 * 规则：角色 level <= 操作者 level 时禁用（不能分配高于或等于自身等级的角色）
 * 如果无法管理目标用户，全部禁用
 */
const isRoleDisabled = (role) => {
  if (!canManageTarget.value) return true
  return role.level <= adminMaxLevel.value
}

/** 重置角色选择到原始状态 */
const resetRoleSelection = () => {
  selectedRoleCodes.value = [...originalRoleCodes.value]
}

const fetchData = async () => {
  loading.value = true
  try {
    const [userRes, rolesRes] = await Promise.all([
      getUserDetail(userId.value),
      getAllRoles()
    ])
    userInfo.value = userRes.data
    allRoles.value = rolesRes.data || []

    const codes = (userRes.data.roles || []).map(r => r.code)
    selectedRoleCodes.value = [...codes]
    originalRoleCodes.value = [...codes]
  } catch (err) {
    console.error('获取数据失败:', err)
    userInfo.value = null
  } finally {
    loading.value = false
  }
}

const goBack = () => {
  router.push('/user/list')
}

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
    fetchData()
  } catch (err) {
    if (err !== 'cancel' && err !== 'close') {
      console.error(`${action}失败:`, err)
    }
  }
}

const handleSaveRoles = async () => {
  if (!hasRoleChanges.value) return

  roleLoading.value = true
  try {
    await setUserRoles(userId.value, selectedRoleCodes.value)
    ElMessage.success('角色设置成功')
    fetchData()
  } catch (err) {
    console.error('角色设置失败:', err)
  } finally {
    roleLoading.value = false
  }
}

onMounted(() => {
  fetchData()
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
  margin-bottom: 16px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: #1E293B;
  margin: 0 0 12px 0;
}

.role-checkbox-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.role-checkbox-item {
  height: auto;
}

.role-checkbox-label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.role-level-badge {
  font-size: 11px;
  color: #94A3B8;
  background: #F1F5F9;
  padding: 1px 6px;
  border-radius: 4px;
}

.role-actions {
  display: flex;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid #E2E8F0;
}

.role-hint {
  font-size: 12px;
  color: #94A3B8;
  margin-top: 8px;
  margin-bottom: 0;
}
</style>
