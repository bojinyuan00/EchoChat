<!--
  会议视频网格（Task 11 / Task 15 柔性网格升级）

  职责：
  - 根据成员数量自适应网格布局（1/2/3/4-9/10+）
  - Task 15 原创特色 2：2 人左右 65/35 非等分；3 人「大+两个小」三角布局；4+ 人维持等分
  - 本地 tile 由上层 SelfVideoFloat 承接浮窗，当传入 tiles 仅含远端时 layout-* 数字即可对应视觉人数
  - 入会滑入动效：新成员 tile 挂载时向上位移 + 渐显（Task 15 原创特色 5）
  - 不关心业务数据，仅负责排版；视频块由 slot #tile 自定义渲染，保持数据与组件解耦
-->
<template>
  <view class="grid-root">
    <view class="grid" :class="layoutClass" :style="gridStyle">
      <view
        v-for="(tile, idx) in tiles"
        :key="tile.key || tile.userId || idx"
        class="grid-cell"
        :class="cellClassFor(idx)"
      >
        <slot name="tile" :tile="tile" :index="idx" />
      </view>
    </view>
    <view v-if="!tiles.length" class="empty">
      <text>暂无成员，等待其他人加入…</text>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  tiles: { type: Array, default: () => [] }
})

/**
 * 布局档位：按成员数量决定非等分策略
 * - 1: 单屏
 * - 2: 左右 65/35
 * - 3: 左大 + 右两叠
 * - 4-9: 3×3 等分兜底
 * - 10+: sqrt 自适应
 */
const layoutClass = computed(() => {
  const n = props.tiles.length
  if (n === 0) return 'layout-empty'
  if (n === 1) return 'layout-1'
  if (n === 2) return 'layout-2-flex'
  if (n === 3) return 'layout-3-tri'
  if (n <= 9) return 'layout-grid-3'
  return 'layout-grid-sqrt'
})

/**
 * grid-template 计算：
 * - layout-2-flex 用 grid-template-columns: 65fr 35fr（单行）
 * - layout-3-tri  用 2 列，左列 1 行、右列 2 行（cell 手动控制 span）
 * - layout-grid-3 同现有逻辑（3 列）
 * - layout-grid-sqrt 同现有逻辑（sqrt 列）
 */
const gridStyle = computed(() => {
  const n = props.tiles.length
  if (n === 2) {
    return {
      'grid-template-columns': '65fr 35fr',
      'grid-template-rows': '1fr'
    }
  }
  if (n === 3) {
    return {
      'grid-template-columns': '60fr 40fr',
      'grid-template-rows': '1fr 1fr'
    }
  }
  if (n === 1) {
    return {
      'grid-template-columns': '1fr',
      'grid-template-rows': '1fr'
    }
  }
  if (n <= 9) {
    const cols = n <= 4 ? 2 : 3
    const rows = Math.ceil(n / cols)
    return {
      'grid-template-columns': `repeat(${cols}, minmax(0, 1fr))`,
      'grid-template-rows': `repeat(${rows}, minmax(0, 1fr))`
    }
  }
  const cols = Math.ceil(Math.sqrt(n))
  const rows = Math.ceil(n / cols)
  return {
    'grid-template-columns': `repeat(${cols}, minmax(0, 1fr))`,
    'grid-template-rows': `repeat(${rows}, minmax(0, 1fr))`
  }
})

/** 单元格特殊 class：3 人三角布局里，首个 tile 左列跨 2 行 */
const cellClassFor = (idx) => {
  if (props.tiles.length === 3 && idx === 0) return 'cell-span-row-2'
  return ''
}
</script>

<style scoped>
.grid-root {
  position: relative;
  width: 100%;
  height: 100%;
  padding: 16rpx;
  box-sizing: border-box;
  overflow: hidden;
}

.grid {
  width: 100%;
  height: 100%;
  display: grid;
  gap: 12rpx;
}

.grid-cell {
  position: relative;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  animation: tile-slide-in 0.28s cubic-bezier(0.2, 0.8, 0.2, 1) both;
}

/* 3 人布局：首个 tile 左列占据整列高度（2 行） */
.cell-span-row-2 {
  grid-row: span 2;
}

.empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.55);
  font-size: 26rpx;
  letter-spacing: 1rpx;
}

/* Task 15 原创特色 5：入会滑入动效 */
@keyframes tile-slide-in {
  from { transform: translateY(16rpx); opacity: 0; }
  to   { transform: translateY(0);     opacity: 1; }
}

/* 移动端（小屏）：柔性布局回退为纵向等分，避免 65/35 在窄屏上过度挤压 */
@media (max-width: 750px) {
  .grid.layout-2-flex {
    grid-template-columns: 1fr !important;
    grid-template-rows: 1fr 1fr !important;
  }
  .grid.layout-3-tri {
    grid-template-columns: 1fr !important;
    grid-template-rows: 1fr 1fr 1fr !important;
  }
  .cell-span-row-2 {
    grid-row: span 1;
  }
  .grid.layout-1 {
    grid-template-rows: 1fr !important;
  }
}

/* 可访问性：系统偏好减少动效时禁用滑入 */
@media (prefers-reduced-motion: reduce) {
  .grid-cell { animation: none; }
}
</style>
