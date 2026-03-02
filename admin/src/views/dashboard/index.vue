<!--
  仪表盘页面（占位）
  
  设计系统：design-system/echochat/MASTER.md
  
  当前功能：
  - 显示 4 个统计卡片占位（总用户数、在线用户、今日会议、系统消息）
  - 后续 Phase 完善时接入真实数据 API
-->
<template>
  <div class="dashboard-page">
    <h2 class="page-title">仪表盘</h2>

    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stat-row">
      <el-col :xs="24" :sm="12" :lg="6" v-for="item in stats" :key="item.label">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-header">
            <el-icon :size="28" :color="item.color"><component :is="item.icon" /></el-icon>
          </div>
          <div class="stat-value">{{ item.value }}</div>
          <div class="stat-label">{{ item.label }}</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 开发提示 -->
    <el-card class="dev-notice">
      <template #header>
        <span>开发进度</span>
      </template>
      <p>Phase 1: 基础设施与用户认证模块 — <el-tag type="warning">进行中</el-tag></p>
      <p style="margin-top: 12px; color: #94A3B8; font-size: 14px;">
        更多数据面板将在后续阶段中完善（IM 统计、会议统计、系统监控等）
      </p>
    </el-card>
  </div>
</template>

<script setup>
/**
 * 仪表盘数据
 * 当前使用占位数据，后续接入统计 API
 */
import { shallowRef, markRaw } from 'vue'
import { UserFilled, Connection, VideoCameraFilled, ChatDotRound } from '@element-plus/icons-vue'

/** 使用 shallowRef + markRaw 避免 Vue 对组件对象做深层响应式处理 */
const stats = shallowRef([
  { label: '总用户数', value: '--', icon: markRaw(UserFilled), color: '#2563EB' },
  { label: '在线用户', value: '--', icon: markRaw(Connection), color: '#22C55E' },
  { label: '今日会议', value: '--', icon: markRaw(VideoCameraFilled), color: '#F59E0B' },
  { label: '系统消息', value: '--', icon: markRaw(ChatDotRound), color: '#8B5CF6' }
])
</script>

<style scoped>
.dashboard-page {
  max-width: 1400px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #1E293B;
  margin-bottom: 20px;
}

.stat-row {
  margin-bottom: 24px;
}

.stat-card {
  text-align: center;
  padding: 8px 0;
  margin-bottom: 12px;
}

.stat-header {
  margin-bottom: 12px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #1E293B;
}

.stat-label {
  font-size: 14px;
  color: #94A3B8;
  margin-top: 4px;
}

.dev-notice {
  margin-top: 8px;
}
</style>
