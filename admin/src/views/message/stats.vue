<!--
  管理端消息统计仪表板

  设计系统：Data-Dense Dashboard 风格
  图表库：ECharts
  主题色：Primary #2563EB

  功能：
  - 顶部统计卡片（总消息数 + 今日消息数）
  - 消息趋势折线图（7/30 天切换）
  - 消息类型分布饼图
  - 活跃用户排行水平柱状图（Top 10）
  - 活跃群组排行水平柱状图（Top 10）

  对应后端 API：GET /api/v1/admin/messages/stats
-->
<template>
  <div class="stats-page">
    <div class="page-header">
      <h2 class="page-title">消息统计</h2>
      <div class="header-actions">
        <el-radio-group v-model="days" size="default" @change="fetchStats">
          <el-radio-button :value="7">近 7 天</el-radio-button>
          <el-radio-button :value="30">近 30 天</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stat-cards">
      <div class="stat-card">
        <div class="stat-card-icon total-icon">
          <el-icon :size="24"><ChatDotRound /></el-icon>
        </div>
        <div class="stat-card-info">
          <span class="stat-label">消息总数</span>
          <span class="stat-value">{{ formatNumber(stats.total_count) }}</span>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-card-icon today-icon">
          <el-icon :size="24"><Sunrise /></el-icon>
        </div>
        <div class="stat-card-info">
          <span class="stat-label">今日消息</span>
          <span class="stat-value">{{ formatNumber(stats.today_count) }}</span>
        </div>
      </div>
    </div>

    <!-- 图表区域 -->
    <div v-loading="loading" class="charts-grid">
      <!-- 消息趋势折线图 -->
      <el-card class="chart-card chart-card-wide" shadow="never">
        <template #header>
          <span class="chart-title">消息趋势</span>
        </template>
        <div ref="trendChartRef" class="chart-container"></div>
      </el-card>

      <!-- 消息类型分布饼图 -->
      <el-card class="chart-card" shadow="never">
        <template #header>
          <span class="chart-title">类型分布</span>
        </template>
        <div ref="typeChartRef" class="chart-container"></div>
      </el-card>

      <!-- 活跃用户排行 -->
      <el-card class="chart-card" shadow="never">
        <template #header>
          <span class="chart-title">活跃用户 Top 10</span>
        </template>
        <div ref="userChartRef" class="chart-container"></div>
      </el-card>

      <!-- 活跃群组排行 -->
      <el-card class="chart-card" shadow="never">
        <template #header>
          <span class="chart-title">活跃群组 Top 10</span>
        </template>
        <div ref="groupChartRef" class="chart-container"></div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
/**
 * 管理端消息统计逻辑
 *
 * 使用 ECharts 渲染四个图表，数据从 stats API 获取
 */
import { ref, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { ChatDotRound, Sunrise } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'
import { getMessageStats } from '@/api/message'

const PRIMARY = '#2563EB'
const CHART_COLORS = ['#2563EB', '#7C3AED', '#059669', '#D97706', '#DC2626', '#0891B2']

const days = ref(7)
const loading = ref(false)
const stats = ref({
  total_count: 0,
  today_count: 0,
  type_distribution: [],
  daily_trend: [],
  active_users: [],
  active_groups: []
})

const trendChartRef = ref(null)
const typeChartRef = ref(null)
const userChartRef = ref(null)
const groupChartRef = ref(null)

let trendChart = null
let typeChart = null
let userChart = null
let groupChart = null

const formatNumber = (n) => {
  if (n == null) return '0'
  return n.toLocaleString()
}

/** 获取统计数据 */
const fetchStats = async () => {
  loading.value = true
  try {
    const res = await getMessageStats({ days: days.value })
    stats.value = res.data || {}
    await nextTick()
    renderCharts()
  } catch (e) {
    ElMessage.error(e?.message || '获取统计数据失败')
  } finally {
    loading.value = false
  }
}

/** 渲染所有图表 */
const renderCharts = () => {
  renderTrendChart()
  renderTypeChart()
  renderUserChart()
  renderGroupChart()
}

/** 消息趋势折线图 */
const renderTrendChart = () => {
  if (!trendChartRef.value) return
  if (!trendChart) trendChart = echarts.init(trendChartRef.value)
  const trend = stats.value.daily_trend || []
  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 50, right: 20, top: 20, bottom: 30 },
    xAxis: {
      type: 'category',
      data: trend.map(d => d.date?.substring(5) || ''),
      axisLabel: { color: '#64748B', fontSize: 11 },
      axisLine: { lineStyle: { color: '#E2E8F0' } }
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#94A3B8', fontSize: 11 },
      splitLine: { lineStyle: { color: '#F1F5F9' } }
    },
    series: [{
      type: 'line',
      data: trend.map(d => d.count),
      smooth: true,
      symbol: 'circle',
      symbolSize: 6,
      lineStyle: { color: PRIMARY, width: 2 },
      itemStyle: { color: PRIMARY },
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(37, 99, 235, 0.2)' },
          { offset: 1, color: 'rgba(37, 99, 235, 0.02)' }
        ])
      }
    }]
  })
}

/** 消息类型分布饼图 */
const renderTypeChart = () => {
  if (!typeChartRef.value) return
  if (!typeChart) typeChart = echarts.init(typeChartRef.value)
  const dist = stats.value.type_distribution || []
  typeChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      textStyle: { color: '#64748B', fontSize: 12 }
    },
    color: CHART_COLORS,
    series: [{
      type: 'pie',
      radius: ['40%', '65%'],
      center: ['40%', '50%'],
      avoidLabelOverlap: true,
      label: { show: false },
      emphasis: { label: { show: true, fontWeight: 'bold' } },
      data: dist.map(d => ({ value: d.count, name: d.label || `类型${d.type}` }))
    }]
  })
}

/** 活跃用户水平柱状图 */
const renderUserChart = () => {
  if (!userChartRef.value) return
  if (!userChart) userChart = echarts.init(userChartRef.value)
  const users = (stats.value.active_users || []).slice(0, 10).reverse()
  userChart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { left: 90, right: 30, top: 10, bottom: 20 },
    xAxis: {
      type: 'value',
      axisLabel: { color: '#94A3B8', fontSize: 11 },
      splitLine: { lineStyle: { color: '#F1F5F9' } }
    },
    yAxis: {
      type: 'category',
      data: users.map(u => u.nickname || `用户${u.user_id}`),
      axisLabel: { color: '#64748B', fontSize: 11, width: 70, overflow: 'truncate' }
    },
    series: [{
      type: 'bar',
      data: users.map(u => u.count),
      barWidth: 14,
      itemStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
          { offset: 0, color: '#2563EB' },
          { offset: 1, color: '#60A5FA' }
        ]),
        borderRadius: [0, 4, 4, 0]
      }
    }]
  })
}

/** 活跃群组水平柱状图 */
const renderGroupChart = () => {
  if (!groupChartRef.value) return
  if (!groupChart) groupChart = echarts.init(groupChartRef.value)
  const groups = (stats.value.active_groups || []).slice(0, 10).reverse()
  groupChart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { left: 90, right: 30, top: 10, bottom: 20 },
    xAxis: {
      type: 'value',
      axisLabel: { color: '#94A3B8', fontSize: 11 },
      splitLine: { lineStyle: { color: '#F1F5F9' } }
    },
    yAxis: {
      type: 'category',
      data: groups.map(g => g.name || `群组${g.group_id}`),
      axisLabel: { color: '#64748B', fontSize: 11, width: 70, overflow: 'truncate' }
    },
    series: [{
      type: 'bar',
      data: groups.map(g => g.count),
      barWidth: 14,
      itemStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
          { offset: 0, color: '#7C3AED' },
          { offset: 1, color: '#A78BFA' }
        ]),
        borderRadius: [0, 4, 4, 0]
      }
    }]
  })
}

/** 窗口 resize 时自动调整图表尺寸 */
const handleResize = () => {
  trendChart?.resize()
  typeChart?.resize()
  userChart?.resize()
  groupChart?.resize()
}

onMounted(() => {
  fetchStats()
  window.addEventListener('resize', handleResize)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
  typeChart?.dispose()
  userChart?.dispose()
  groupChart?.dispose()
})
</script>

<style scoped>
.stats-page { padding: 0; }

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}
.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #1E293B;
  margin: 0;
}

/* 统计卡片 */
.stat-cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 24px;
  background: #FFFFFF;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  border: 1px solid #E2E8F0;
}
.stat-card-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.total-icon {
  background-color: rgba(37, 99, 235, 0.1);
  color: #2563EB;
}
.today-icon {
  background-color: rgba(5, 150, 105, 0.1);
  color: #059669;
}
.stat-card-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.stat-label {
  font-size: 13px;
  color: #94A3B8;
  font-weight: 500;
}
.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #0F172A;
  line-height: 1.2;
}

/* 图表网格 */
.charts-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}
.chart-card { border-radius: 12px; }
.chart-card-wide { grid-column: span 2; }
.chart-title {
  font-size: 14px;
  font-weight: 600;
  color: #334155;
}
.chart-container {
  width: 100%;
  height: 300px;
}
</style>
