<!--
  在线监控页面（管理端）

  设计系统：design-system/echochat/MASTER.md
  色板：Primary #2563EB / Text #1E293B
  ui-ux-pro-max 规范：数据表格 / 自动刷新 / loading 状态

  功能：
  - 在线用户数量统计卡片
  - 在线用户列表（表格展示）
  - 自动刷新（30 秒轮询）
  - 手动刷新按钮
-->
<template>
  <div class="online-monitor-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="page-title">在线监控</h2>
      <div class="header-actions">
        <el-tag :type="autoRefresh ? 'success' : 'info'" size="small" class="refresh-tag">
          {{ autoRefresh ? '自动刷新中' : '已暂停' }}
        </el-tag>
        <el-switch v-model="autoRefresh" active-text="自动" inactive-text="手动" />
        <el-button :icon="Refresh" @click="fetchData" :loading="loading">刷新</el-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stat-cards">
      <el-col :span="8">
        <el-card shadow="never" class="stat-card">
          <div class="stat-content">
            <div class="stat-number" :class="{ 'stat-loading': loading }">
              {{ onlineCount }}
            </div>
            <div class="stat-label">当前在线</div>
          </div>
          <div class="stat-indicator stat-indicator--online"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="never" class="stat-card">
          <div class="stat-content">
            <div class="stat-number">{{ lastRefreshTime }}</div>
            <div class="stat-label">最后刷新</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="never" class="stat-card">
          <div class="stat-content">
            <div class="stat-number">30s</div>
            <div class="stat-label">刷新间隔</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 在线用户表格 -->
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span>在线用户列表</span>
          <el-tag size="small" type="success">{{ onlineUsers.length }} 人</el-tag>
        </div>
      </template>

      <el-table
        v-loading="loading"
        :data="onlineUsers"
        stripe
        border
        style="width: 100%"
        empty-text="暂无在线用户"
      >
        <el-table-column type="index" label="#" width="60" align="center" />
        <el-table-column prop="user_id" label="用户 ID" width="120" align="center" />
        <el-table-column prop="username" label="用户名" min-width="180">
          <template #default="{ row }">
            <span class="username-cell">{{ row.username }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120" align="center">
          <template #default>
            <div class="online-status">
              <span class="online-dot"></span>
              <span>在线</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { getOnlineUsers, getOnlineCount } from '@/api/monitor'

const loading = ref(false)
const onlineCount = ref(0)
const onlineUsers = ref([])
const autoRefresh = ref(true)
const lastRefreshTime = ref('--')
let timer = null

const formatTime = () => {
  const now = new Date()
  const h = String(now.getHours()).padStart(2, '0')
  const m = String(now.getMinutes()).padStart(2, '0')
  const s = String(now.getSeconds()).padStart(2, '0')
  return `${h}:${m}:${s}`
}

const fetchData = async () => {
  loading.value = true
  try {
    const [countRes, usersRes] = await Promise.all([
      getOnlineCount(),
      getOnlineUsers()
    ])
    onlineCount.value = countRes.data?.count || 0
    onlineUsers.value = usersRes.data || []
    lastRefreshTime.value = formatTime()
  } catch (e) {
    console.error('获取在线数据失败', e)
  } finally {
    loading.value = false
  }
}

const startAutoRefresh = () => {
  stopAutoRefresh()
  timer = setInterval(fetchData, 30000)
}

const stopAutoRefresh = () => {
  if (timer) {
    clearInterval(timer)
    timer = null
  }
}

watch(autoRefresh, (val) => {
  if (val) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
})

onMounted(() => {
  fetchData()
  if (autoRefresh.value) startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.online-monitor-page {
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

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.refresh-tag {
  font-size: 12px;
}

/* 统计卡片 */
.stat-cards {
  margin-bottom: 16px;
}

.stat-card {
  position: relative;
  overflow: hidden;
}

.stat-content {
  text-align: center;
  padding: 8px 0;
}

.stat-number {
  font-size: 28px;
  font-weight: 700;
  color: #1E293B;
  margin-bottom: 4px;
  transition: opacity 200ms ease;
}

.stat-loading {
  opacity: 0.5;
}

.stat-label {
  font-size: 13px;
  color: #94A3B8;
}

.stat-indicator {
  position: absolute;
  top: 0;
  left: 0;
  width: 4px;
  height: 100%;
}

.stat-indicator--online {
  background-color: #10B981;
}

/* 表格 */
.table-card {
  margin-bottom: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.username-cell {
  font-weight: 500;
  color: #2563EB;
}

.online-status {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-size: 13px;
  color: #059669;
}

.online-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #10B981;
}
</style>
